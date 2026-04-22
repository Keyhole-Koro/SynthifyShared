package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Keyhole-Koro/SynthifyShared/domain"
	"github.com/Keyhole-Koro/SynthifyShared/repository/postgres/sqlcgen"
)

func (s *Store) ListDocuments(wsID string) []*domain.Document {
	rows, err := s.q().ListDocuments(context.Background(), wsID)
	if err != nil {
		return nil
	}
	var docs []*domain.Document
	for _, row := range rows {
		docs = append(docs, toDocument(row))
	}
	return docs
}

func (s *Store) GetDocument(id string) (*domain.Document, bool) {
	row, err := s.q().GetDocument(context.Background(), id)
	if err != nil {
		return nil, false
	}
	return toDocument(row), true
}

func (s *Store) CreateDocument(wsID, uploadedBy, filename, mimeType string, fileSize int64) (*domain.Document, string) {
	createdAt := nowTime()
	doc := &domain.Document{
		DocumentID:  newID(),
		WorkspaceID: wsID,
		UploadedBy:  uploadedBy,
		Filename:    filename,
		MimeType:    mimeType,
		FileSize:    fileSize,
		CreatedAt:   createdAt.Format(time.RFC3339),
	}
	err := s.q().CreateDocument(context.Background(), sqlcgen.CreateDocumentParams{
		DocumentID:  doc.DocumentID,
		WorkspaceID: doc.WorkspaceID,
		UploadedBy:  doc.UploadedBy,
		Filename:    doc.Filename,
		MimeType:    doc.MimeType,
		FileSize:    doc.FileSize,
		CreatedAt:   createdAt,
	})
	if err != nil {
		return nil, ""
	}
	return doc, s.uploadURLGenerator(wsID, doc.DocumentID)
}

func (s *Store) GetLatestProcessingJob(docID string) (*domain.DocumentProcessingJob, bool) {
	row := s.db.QueryRowContext(context.Background(), `
		SELECT job_id, document_id, COALESCE(graph_id, ''), job_type, status, current_stage, error_message, params_json,
		       requested_by, capability_id, execution_plan_id, plan_status, evaluation_status, retry_count, budget_json,
		       created_at, updated_at
		FROM document_processing_jobs
		WHERE document_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, docID)
	job, err := scanProcessingJob(row)
	if err != nil {
		return nil, false
	}
	return job, true
}

func (s *Store) GetJobCapability(jobID string) (*domain.JobCapability, bool) {
	row := s.db.QueryRowContext(context.Background(), `
		SELECT capability_id, job_id, workspace_id, graph_id, allowed_document_ids_json, allowed_node_ids_json,
		       allowed_operations_json, max_llm_calls, max_tool_runs, max_node_creations, max_edge_mutations,
		       expires_at, created_at
		FROM job_capabilities
		WHERE job_id = $1
	`, jobID)
	var capability domain.JobCapability
	var allowedDocumentIDsJSON, allowedNodeIDsJSON, allowedOperationsJSON string
	var expiresAt, createdAt time.Time
	if err := row.Scan(
		&capability.CapabilityID,
		&capability.JobID,
		&capability.WorkspaceID,
		&capability.GraphID,
		&allowedDocumentIDsJSON,
		&allowedNodeIDsJSON,
		&allowedOperationsJSON,
		&capability.MaxLLMCalls,
		&capability.MaxToolRuns,
		&capability.MaxNodeCreations,
		&capability.MaxEdgeMutations,
		&expiresAt,
		&createdAt,
	); err != nil {
		return nil, false
	}
	if err := json.Unmarshal([]byte(allowedDocumentIDsJSON), &capability.AllowedDocumentIDs); err != nil {
		return nil, false
	}
	if err := json.Unmarshal([]byte(allowedNodeIDsJSON), &capability.AllowedNodeIDs); err != nil {
		return nil, false
	}
	var allowedOperations []string
	if err := json.Unmarshal([]byte(allowedOperationsJSON), &allowedOperations); err != nil {
		return nil, false
	}
	for _, op := range allowedOperations {
		capability.AllowedOperations = append(capability.AllowedOperations, domain.JobOperation(op))
	}
	capability.ExpiresAt = expiresAt.UTC().Format(time.RFC3339)
	capability.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return &capability, true
}

func (s *Store) CreateProcessingJob(docID, graphID, jobType string) *domain.DocumentProcessingJob {
	createdAt := nowTime()
	jobID := newID()
	doc, ok := s.GetDocument(docID)
	if !ok {
		return nil
	}
	capability := domain.DefaultJobCapability(jobID, doc.WorkspaceID, graphID, docID, createdAt)
	allowedDocumentIDsJSON, _ := json.Marshal(capability.AllowedDocumentIDs)
	allowedNodeIDsJSON, _ := json.Marshal(capability.AllowedNodeIDs)
	allowedOperations := make([]string, 0, len(capability.AllowedOperations))
	for _, op := range capability.AllowedOperations {
		allowedOperations = append(allowedOperations, string(op))
	}
	allowedOperationsJSON, _ := json.Marshal(allowedOperations)
	budgetJSON, _ := json.Marshal(map[string]int{
		"max_llm_calls":      capability.MaxLLMCalls,
		"max_tool_runs":      capability.MaxToolRuns,
		"max_node_creations": capability.MaxNodeCreations,
		"max_edge_mutations": capability.MaxEdgeMutations,
	})
	planID := "plan_" + jobID
	planJSON, _ := json.Marshal(map[string]any{
		"summary": "default document processing pipeline",
		"steps": []map[string]any{
			{
				"title":      "document_pipeline",
				"risk_tier":  "tier_1",
				"operations": allowedOperations,
				"documents":  capability.AllowedDocumentIDs,
			},
		},
	})
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(context.Background(), `
		INSERT INTO document_processing_jobs (
			job_id, document_id, graph_id, job_type, status, current_stage, error_message, params_json,
			requested_by, capability_id, execution_plan_id, plan_status, evaluation_status, retry_count, budget_json,
			created_at, updated_at
		) VALUES ($1, $2, NULLIF($3, ''), $4, 'queued', '', '', '{}', 'system', $5, $6, 'approved', 'pending', 0, $7, $8, $8)
	`, jobID, docID, graphID, jobType, capability.CapabilityID, planID, string(budgetJSON), createdAt); err != nil {
		return nil
	}
	if _, err := tx.ExecContext(context.Background(), `
		INSERT INTO job_capabilities (
			capability_id, job_id, workspace_id, graph_id, allowed_document_ids_json, allowed_node_ids_json,
			allowed_operations_json, max_llm_calls, max_tool_runs, max_node_creations, max_edge_mutations, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, capability.CapabilityID, jobID, capability.WorkspaceID, capability.GraphID, string(allowedDocumentIDsJSON), string(allowedNodeIDsJSON), string(allowedOperationsJSON), capability.MaxLLMCalls, capability.MaxToolRuns, capability.MaxNodeCreations, capability.MaxEdgeMutations, createdAt.Add(24*time.Hour), createdAt); err != nil {
		return nil
	}
	if _, err := tx.ExecContext(context.Background(), `
		INSERT INTO job_execution_plans (plan_id, job_id, status, summary, plan_json, created_by, created_at, updated_at)
		VALUES ($1, $2, 'approved', 'default document processing pipeline', $3, 'planner', $4, $4)
	`, planID, jobID, string(planJSON), createdAt); err != nil {
		return nil
	}
	if err := tx.Commit(); err != nil {
		return nil
	}
	return &domain.DocumentProcessingJob{
		JobID:            jobID,
		DocumentID:       docID,
		GraphID:          graphID,
		JobType:          jobType,
		Status:           "queued",
		RequestedBy:      "system",
		CapabilityID:     capability.CapabilityID,
		ExecutionPlanID:  planID,
		PlanStatus:       "approved",
		EvaluationStatus: "pending",
		BudgetJSON:       string(budgetJSON),
		CreatedAt:        createdAt.Format(time.RFC3339),
		UpdatedAt:        createdAt.Format(time.RFC3339),
	}
}

func (s *Store) MarkProcessingJobRunning(jobID string) bool {
	result, err := s.db.ExecContext(context.Background(), `
		UPDATE document_processing_jobs
		SET status = 'running', error_message = '', plan_status = CASE WHEN plan_status IN ('', 'approved') THEN 'executing' ELSE plan_status END, updated_at = $2
		WHERE job_id = $1
	`, jobID, nowTime())
	if err != nil {
		return false
	}
	affected, _ := result.RowsAffected()
	return affected > 0
}

func (s *Store) UpdateProcessingJobStage(jobID, stage string) bool {
	return s.q().UpdateProcessingJobStage(context.Background(), sqlcgen.UpdateProcessingJobStageParams{
		JobID:        jobID,
		CurrentStage: stage,
		UpdatedAt:    nowTime(),
	}) == nil
}

func (s *Store) FailProcessingJob(jobID, errorMessage string) bool {
	affected, err := s.db.ExecContext(context.Background(), `
		UPDATE document_processing_jobs
		SET status = 'failed', error_message = $2, evaluation_status = 'failed', updated_at = $3
		WHERE job_id = $1
	`, jobID, errorMessage, nowTime())
	if err != nil {
		return false
	}
	rowsAffected, _ := affected.RowsAffected()
	return rowsAffected > 0
}

func (s *Store) CompleteProcessingJob(jobID string) bool {
	affected, err := s.db.ExecContext(context.Background(), `
		UPDATE document_processing_jobs
		SET status = 'completed', current_stage = '', plan_status = 'completed', evaluation_status = 'passed', updated_at = $2
		WHERE job_id = $1
	`, jobID, nowTime())
	if err != nil {
		return false
	}
	rowsAffected, _ := affected.RowsAffected()
	return rowsAffected > 0
}

func scanProcessingJob(row scanner) (*domain.DocumentProcessingJob, error) {
	var job domain.DocumentProcessingJob
	var createdAt, updatedAt time.Time
	if err := row.Scan(
		&job.JobID,
		&job.DocumentID,
		&job.GraphID,
		&job.JobType,
		&job.Status,
		&job.CurrentStage,
		&job.ErrorMessage,
		&job.ParamsJSON,
		&job.RequestedBy,
		&job.CapabilityID,
		&job.ExecutionPlanID,
		&job.PlanStatus,
		&job.EvaluationStatus,
		&job.RetryCount,
		&job.BudgetJSON,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	job.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	job.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return &job, nil
}

func (s *Store) SaveDocumentChunks(documentID string, chunks []*domain.DocumentChunk) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(context.Background(), `DELETE FROM document_chunks WHERE document_id = $1`, documentID); err != nil {
		return err
	}
	for i, chunk := range chunks {
		chunkID := chunk.ChunkID
		if chunkID == "" {
			chunkID = fmt.Sprintf("chk_%s_%03d", documentID, i)
		}
		if _, err := tx.ExecContext(context.Background(), `
			INSERT INTO document_chunks (chunk_id, document_id, heading, text, source_page)
			VALUES ($1, $2, $3, $4, $5)
		`, chunkID, documentID, chunk.Heading, chunk.Text, chunk.SourcePage); err != nil {
			return err
		}
	}
	return tx.Commit()
}
