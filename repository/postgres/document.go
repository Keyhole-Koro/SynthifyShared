package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Keyhole-Koro/SynthifyShared/domain"
	treev1 "github.com/Keyhole-Koro/SynthifyShared/gen/synthify/tree/v1"
	"github.com/Keyhole-Koro/SynthifyShared/repository/postgres/sqlcgen"
)

func (s *Store) ListDocuments(wsID string) []*domain.Document {
	rows, err := s.q().ListDocuments(context.Background(), wsID)
	if err != nil {
		return nil
	}

	docs := make([]*domain.Document, 0, len(rows))
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

func (s *Store) GetDocumentChunks(documentID string) ([]*domain.DocumentChunk, bool) {
	rows, err := s.q().ListDocumentChunks(context.Background(), documentID)
	if err != nil {
		return nil, false
	}

	chunks := make([]*domain.DocumentChunk, 0, len(rows))
	for _, row := range rows {
		chunks = append(chunks, toDocumentChunk(row))
	}
	return chunks, true
}

func (s *Store) GetJobPlanningSignals(documentID, workspaceID, treeID string) (*domain.JobPlanningSignals, bool) {
	signals := &domain.JobPlanningSignals{
		DocumentID:  documentID,
		WorkspaceID: workspaceID,
	}
	if strings.TrimSpace(documentID) == "" || strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(treeID) == "" {
		return signals, true
	}

	sameDocumentItemCount, err := s.q().CountSameDocumentItems(context.Background(), sqlcgen.CountSameDocumentItemsParams{
		DocumentID:  documentID,
		WorkspaceID: treeID,
	})
	if err != nil {
		return nil, false
	}
	approvedAliasCount, err := s.q().CountApprovedAliases(context.Background(), sqlcgen.CountApprovedAliasesParams{
		WorkspaceID:   workspaceID,
		WorkspaceID_2: treeID,
	})
	if err != nil {
		return nil, false
	}
	protectedAliasCount, err := s.q().CountProtectedAliases(context.Background(), sqlcgen.CountProtectedAliasesParams{
		WorkspaceID:   workspaceID,
		WorkspaceID_2: treeID,
	})
	if err != nil {
		return nil, false
	}

	signals.SameDocumentItemCount = int(sameDocumentItemCount)
	signals.ApprovedAliasCount = int(approvedAliasCount)
	signals.ProtectedAliasCount = int(protectedAliasCount)
	return signals, true
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
	row, err := s.q().GetLatestProcessingJob(context.Background(), docID)
	if err != nil {
		return nil, false
	}
	return toProcessingJob(row), true
}

func (s *Store) GetProcessingJob(jobID string) (*domain.DocumentProcessingJob, bool) {
	row, err := s.q().GetProcessingJob(context.Background(), jobID)
	if err != nil {
		return nil, false
	}
	return toProcessingJob(row), true
}

func (s *Store) GetJobCapability(jobID string) (*domain.JobCapability, bool) {
	row, err := s.q().GetJobCapability(context.Background(), jobID)
	if err != nil {
		return nil, false
	}
	capability, err := toJobCapability(row)
	if err != nil {
		return nil, false
	}
	return capability, true
}

func (s *Store) GetJobExecutionPlan(jobID string) (*domain.JobExecutionPlan, bool) {
	row, err := s.q().GetJobExecutionPlan(context.Background(), jobID)
	if err != nil {
		return nil, false
	}
	return toJobExecutionPlan(row), true
}

func (s *Store) UpsertJobExecutionPlan(jobID string, plan *domain.JobExecutionPlan) bool {
	if plan == nil {
		return false
	}

	now := nowTime()
	if strings.TrimSpace(plan.PlanID) == "" {
		plan.PlanID = "plan_" + jobID
	}
	if strings.TrimSpace(plan.Status) == "" {
		plan.Status = "draft"
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return false
	}
	defer tx.Rollback()

	qtx := s.q().WithTx(tx)
	if err := qtx.UpsertJobExecutionPlan(context.Background(), sqlcgen.UpsertJobExecutionPlanParams{
		PlanID:    plan.PlanID,
		JobID:     jobID,
		Status:    plan.Status,
		Summary:   plan.Summary,
		PlanJson:  plan.PlanJSON,
		CreatedBy: firstNonEmptyNonSQL(plan.CreatedBy, "planner"),
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return false
	}
	if err := qtx.UpdateProcessingJobPlanState(context.Background(), sqlcgen.UpdateProcessingJobPlanStateParams{
		JobID:           jobID,
		ExecutionPlanID: plan.PlanID,
		PlanStatus:      plan.Status,
		UpdatedAt:       now,
	}); err != nil {
		return false
	}
	if err := tx.Commit(); err != nil {
		return false
	}

	plan.CreatedAt = now.UTC().Format(time.RFC3339)
	plan.UpdatedAt = plan.CreatedAt
	return true
}

func (s *Store) EvaluateJob(jobID string) (*domain.JobEvaluationResult, bool) {
	job, ok := s.GetProcessingJob(jobID)
	if !ok || job == nil {
		return nil, false
	}
	plan, _ := s.GetJobExecutionPlan(jobID)
	mutationCount, err := s.q().CountJobMutationLogs(context.Background(), jobID)
	if err != nil {
		return nil, false
	}

	result := &domain.JobEvaluationResult{
		JobID:         job.JobID,
		Passed:        job.Status == treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_SUCCEEDED && job.EvaluationStatus != "failed",
		Status:        job.EvaluationStatus,
		MutationCount: int32(mutationCount),
	}
	if plan != nil {
		result.PlanID = plan.PlanID
	}
	if result.Passed {
		result.Summary = "job completed successfully"
		result.Score = 100
		if mutationCount == 0 {
			result.Findings = append(result.Findings, "job completed without any recorded tree mutations")
			result.Score = 70
		}
	} else {
		result.Summary = firstNonEmptyNonSQL(job.ErrorMessage, "job has not reached a passing evaluation state")
		result.Score = 0
		if strings.TrimSpace(job.ErrorMessage) != "" {
			result.Findings = append(result.Findings, job.ErrorMessage)
		} else {
			result.Findings = append(result.Findings, "job status is not completed")
		}
	}
	if result.Status == "" {
		if result.Passed {
			result.Status = "passed"
		} else {
			result.Status = "failed"
		}
	}
	return result, true
}

func (s *Store) ListJobApprovalRequests(jobID string) ([]*domain.JobApprovalRequest, bool) {
	rows, err := s.q().ListJobApprovalRequests(context.Background(), jobID)
	if err != nil {
		return nil, false
	}

	requests := make([]*domain.JobApprovalRequest, 0, len(rows))
	for _, row := range rows {
		req, convErr := toJobApprovalRequest(row)
		if convErr != nil {
			return nil, false
		}
		requests = append(requests, req)
	}
	return requests, true
}

func (s *Store) RequestJobApproval(jobID, requestedBy, reason string) (*domain.JobApprovalRequest, bool) {
	job, ok := s.GetProcessingJob(jobID)
	if !ok || job == nil {
		return nil, false
	}
	plan, ok := s.GetJobExecutionPlan(jobID)
	if !ok || plan == nil {
		return nil, false
	}
	if requests, ok := s.ListJobApprovalRequests(jobID); ok {
		for _, req := range requests {
			if req.Status == "pending" {
				return req, true
			}
		}
	}

	now := nowTime()
	requestedOperations := []treev1.JobOperation{
		treev1.JobOperation_JOB_OPERATION_EMIT_PLAN,
		treev1.JobOperation_JOB_OPERATION_EMIT_EVAL,
	}
	requestedOpsJSON, err := marshalJobOperations(requestedOperations)
	if err != nil {
		return nil, false
	}
	approval := &domain.JobApprovalRequest{
		ApprovalID:          "apr_" + newID(),
		JobID:               jobID,
		PlanID:              plan.PlanID,
		Status:              "pending",
		RequestedOperations: requestedOperations,
		Reason:              firstNonEmptyNonSQL(reason, "approval required before execution"),
		RiskTier:            plan.HighestRiskTier(),
		RequestedBy:         firstNonEmptyNonSQL(requestedBy, "system"),
		RequestedAt:         now.UTC().Format(time.RFC3339),
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, false
	}
	defer tx.Rollback()

	qtx := s.q().WithTx(tx)
	if err := qtx.CreateJobApprovalRequest(context.Background(), sqlcgen.CreateJobApprovalRequestParams{
		ApprovalID:              approval.ApprovalID,
		JobID:                   approval.JobID,
		PlanID:                  approval.PlanID,
		Status:                  approval.Status,
		RequestedOperationsJson: string(requestedOpsJSON),
		Reason:                  approval.Reason,
		RiskTier:                approval.RiskTier,
		RequestedBy:             approval.RequestedBy,
		ReviewedBy:              "",
		RequestedAt:             now,
		ReviewedAt:              sql.NullTime{},
	}); err != nil {
		return nil, false
	}
	if _, err := qtx.UpdateJobExecutionPlanStatus(context.Background(), sqlcgen.UpdateJobExecutionPlanStatusParams{
		PlanID:    approval.PlanID,
		Status:    "pending_approval",
		UpdatedAt: now,
	}); err != nil {
		return nil, false
	}
	if err := qtx.UpdateProcessingJobPlanState(context.Background(), sqlcgen.UpdateProcessingJobPlanStateParams{
		JobID:           approval.JobID,
		ExecutionPlanID: approval.PlanID,
		PlanStatus:      "pending_approval",
		UpdatedAt:       now,
	}); err != nil {
		return nil, false
	}
	if err := tx.Commit(); err != nil {
		return nil, false
	}

	job.PlanStatus = "pending_approval"
	return approval, true
}

func (s *Store) ApproveJobApproval(jobID, approvalID, reviewedBy string) bool {
	now := nowTime()
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return false
	}
	defer tx.Rollback()

	qtx := s.q().WithTx(tx)
	planID, err := qtx.GetJobApprovalPlanID(context.Background(), sqlcgen.GetJobApprovalPlanIDParams{
		JobID:      jobID,
		ApprovalID: approvalID,
	})
	if err != nil {
		return false
	}
	if _, err := qtx.ApproveJobApproval(context.Background(), sqlcgen.ApproveJobApprovalParams{
		JobID:      jobID,
		ApprovalID: approvalID,
		ReviewedBy: firstNonEmptyNonSQL(reviewedBy, "reviewer"),
		ReviewedAt: sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		return false
	}
	if _, err := qtx.UpdateJobExecutionPlanStatus(context.Background(), sqlcgen.UpdateJobExecutionPlanStatusParams{
		PlanID:    planID,
		Status:    "approved",
		UpdatedAt: now,
	}); err != nil {
		return false
	}
	if err := qtx.UpdateProcessingJobPlanState(context.Background(), sqlcgen.UpdateProcessingJobPlanStateParams{
		JobID:           jobID,
		ExecutionPlanID: planID,
		PlanStatus:      "approved",
		UpdatedAt:       now,
	}); err != nil {
		return false
	}

	return tx.Commit() == nil
}

func (s *Store) RejectJobApproval(jobID, approvalID, reviewedBy, reason string) bool {
	now := nowTime()
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return false
	}
	defer tx.Rollback()

	qtx := s.q().WithTx(tx)
	planID, err := qtx.GetJobApprovalPlanID(context.Background(), sqlcgen.GetJobApprovalPlanIDParams{
		JobID:      jobID,
		ApprovalID: approvalID,
	})
	if err != nil {
		return false
	}
	if _, err := qtx.RejectJobApproval(context.Background(), sqlcgen.RejectJobApprovalParams{
		JobID:      jobID,
		ApprovalID: approvalID,
		ReviewedBy: firstNonEmptyNonSQL(reviewedBy, "reviewer"),
		ReviewedAt: sql.NullTime{Time: now, Valid: true},
		Column5:    strings.TrimSpace(reason),
	}); err != nil {
		return false
	}
	if _, err := qtx.UpdateJobExecutionPlanStatus(context.Background(), sqlcgen.UpdateJobExecutionPlanStatusParams{
		PlanID:    planID,
		Status:    "rejected",
		UpdatedAt: now,
	}); err != nil {
		return false
	}
	if err := qtx.UpdateProcessingJobPlanState(context.Background(), sqlcgen.UpdateProcessingJobPlanStateParams{
		JobID:           jobID,
		ExecutionPlanID: planID,
		PlanStatus:      "rejected",
		UpdatedAt:       now,
	}); err != nil {
		return false
	}

	return tx.Commit() == nil
}

func (s *Store) CreateProcessingJob(docID, workspaceID string, jobType treev1.JobType) *domain.DocumentProcessingJob {
	createdAt := nowTime()
	jobID := newID()
	doc, ok := s.GetDocument(docID)
	if !ok {
		return nil
	}

	capability := domain.DefaultJobCapability(jobID, doc.WorkspaceID, docID, createdAt)
	allowedDocumentIDsJSON, err := json.Marshal(capability.AllowedDocumentIDs)
	if err != nil {
		return nil
	}
	allowedItemIDsJSON, err := json.Marshal(capability.AllowedItemIDs)
	if err != nil {
		return nil
	}
	allowedOperationsJSON, err := marshalJobOperations(capability.AllowedOperations)
	if err != nil {
		return nil
	}
	budgetJSON, err := json.Marshal(map[string]int{
		"max_llm_calls":      capability.MaxLLMCalls,
		"max_tool_runs":      capability.MaxToolRuns,
		"max_item_creations": capability.MaxItemCreations,
	})
	if err != nil {
		return nil
	}

	planID := "plan_" + jobID
	planJSON, err := json.Marshal(map[string]any{
		"summary": "default document processing pipeline",
		"steps": []map[string]any{
			{
				"title":      "document_pipeline",
				"risk_tier":  "tier_1",
				"operations": capability.AllowedOperations,
				"documents":  capability.AllowedDocumentIDs,
			},
		},
	})
	if err != nil {
		return nil
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil
	}
	defer tx.Rollback()

	qtx := s.q().WithTx(tx)
	if err := qtx.CreateProcessingJob(context.Background(), sqlcgen.CreateProcessingJobParams{
		JobID:            jobID,
		DocumentID:       docID,
		WorkspaceID:      workspaceID,
		JobType:          formatJobType(jobType),
		Status:           "queued",
		CurrentStage:     "",
		ErrorMessage:     "",
		ParamsJson:       "{}",
		RequestedBy:      "system",
		CapabilityID:     capability.CapabilityID,
		ExecutionPlanID:  planID,
		PlanStatus:       "approved",
		EvaluationStatus: "pending",
		RetryCount:       0,
		BudgetJson:       string(budgetJSON),
		CreatedAt:        createdAt,
	}); err != nil {
		return nil
	}
	if err := qtx.CreateJobCapability(context.Background(), sqlcgen.CreateJobCapabilityParams{
		CapabilityID:           capability.CapabilityID,
		JobID:                  capability.JobID,
		WorkspaceID:            capability.WorkspaceID,
		AllowedDocumentIdsJson: string(allowedDocumentIDsJSON),
		AllowedItemIdsJson:     string(allowedItemIDsJSON),
		AllowedOperationsJson:  string(allowedOperationsJSON),
		MaxLlmCalls:            int32(capability.MaxLLMCalls),
		MaxToolRuns:            int32(capability.MaxToolRuns),
		MaxItemCreations:       int32(capability.MaxItemCreations),
		ExpiresAt:              mustParseTime(capability.ExpiresAt),
		CreatedAt:              mustParseTime(capability.CreatedAt),
	}); err != nil {
		return nil
	}
	if err := qtx.UpsertJobExecutionPlan(context.Background(), sqlcgen.UpsertJobExecutionPlanParams{
		PlanID:    planID,
		JobID:     jobID,
		Status:    "approved",
		Summary:   "default document processing pipeline",
		PlanJson:  string(planJSON),
		CreatedBy: "planner",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}); err != nil {
		return nil
	}
	if err := tx.Commit(); err != nil {
		return nil
	}

	return &domain.DocumentProcessingJob{
		JobID:            jobID,
		DocumentID:       docID,
		WorkspaceID:      workspaceID,
		JobType:          jobType,
		Status:           treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_QUEUED,
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
	rowsAffected, err := s.q().MarkProcessingJobRunning(context.Background(), sqlcgen.MarkProcessingJobRunningParams{
		JobID:     jobID,
		UpdatedAt: nowTime(),
	})
	return err == nil && rowsAffected > 0
}

func (s *Store) UpdateProcessingJobStage(jobID, stage string) bool {
	return s.q().UpdateProcessingJobStage(context.Background(), sqlcgen.UpdateProcessingJobStageParams{
		JobID:        jobID,
		CurrentStage: stage,
		UpdatedAt:    nowTime(),
	}) == nil
}

func (s *Store) FailProcessingJob(jobID, errorMessage string) bool {
	rowsAffected, err := s.q().FailProcessingJob(context.Background(), sqlcgen.FailProcessingJobParams{
		JobID:        jobID,
		ErrorMessage: errorMessage,
		UpdatedAt:    nowTime(),
	})
	return err == nil && rowsAffected > 0
}

func (s *Store) CompleteProcessingJob(jobID string) bool {
	rowsAffected, err := s.q().CompleteProcessingJob(context.Background(), sqlcgen.CompleteProcessingJobParams{
		JobID:     jobID,
		UpdatedAt: nowTime(),
	})
	return err == nil && rowsAffected > 0
}

func (s *Store) ListAllJobs() ([]*domain.DocumentProcessingJob, bool) {
	rows, err := s.q().ListAllJobs(context.Background())
	if err != nil {
		return nil, false
	}
	var res []*domain.DocumentProcessingJob
	for _, row := range rows {
		res = append(res, toProcessingJob(row))
	}
	return res, true
}

func (s *Store) SaveDocumentChunks(documentID string, chunks []*domain.DocumentChunk) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.q().WithTx(tx)
	if err := qtx.DeleteDocumentChunks(context.Background(), documentID); err != nil {
		return err
	}
	for i, chunk := range chunks {
		chunkID := chunk.ChunkID
		if chunkID == "" {
			chunkID = "chk_" + documentID + "_" + leftPadIndex(i)
		}
		if err := qtx.CreateDocumentChunk(context.Background(), sqlcgen.CreateDocumentChunkParams{
			ChunkID:    chunkID,
			DocumentID: documentID,
			Heading:    chunk.Heading,
			Text:       chunk.Text,
			SourcePage: sql.NullInt32{Int32: int32(chunk.SourcePage), Valid: true},
		}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func toDocumentChunk(row sqlcgen.DocumentChunk) *domain.DocumentChunk {
	return &domain.DocumentChunk{
		ChunkID:    row.ChunkID,
		DocumentID: row.DocumentID,
		Heading:    row.Heading,
		Text:       row.Text,
		SourcePage: int(row.SourcePage.Int32),
	}
}

func toJobCapability(row sqlcgen.JobCapability) (*domain.JobCapability, error) {
	var allowedDocumentIDs []string
	if err := json.Unmarshal([]byte(row.AllowedDocumentIdsJson), &allowedDocumentIDs); err != nil {
		return nil, err
	}
	var allowedItemIDs []string
	if err := json.Unmarshal([]byte(row.AllowedItemIdsJson), &allowedItemIDs); err != nil {
		return nil, err
	}
	var opNames []string
	if err := json.Unmarshal([]byte(row.AllowedOperationsJson), &opNames); err != nil {
		return nil, err
	}
	allowedOperations := make([]treev1.JobOperation, 0, len(opNames))
	for _, opName := range opNames {
		allowedOperations = append(allowedOperations, parseJobOperation(opName))
	}

	return &domain.JobCapability{
		CapabilityID:       row.CapabilityID,
		JobID:              row.JobID,
		WorkspaceID:        row.WorkspaceID,
		AllowedDocumentIDs: allowedDocumentIDs,
		AllowedItemIDs:     allowedItemIDs,
		AllowedOperations:  allowedOperations,
		MaxLLMCalls:        int(row.MaxLlmCalls),
		MaxToolRuns:        int(row.MaxToolRuns),
		MaxItemCreations:   int(row.MaxItemCreations),
		ExpiresAt:          row.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt:          row.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func toJobExecutionPlan(row sqlcgen.JobExecutionPlan) *domain.JobExecutionPlan {
	return &domain.JobExecutionPlan{
		PlanID:    row.PlanID,
		JobID:     row.JobID,
		Status:    row.Status,
		Summary:   row.Summary,
		PlanJSON:  row.PlanJson,
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toJobApprovalRequest(row sqlcgen.JobApprovalRequest) (*domain.JobApprovalRequest, error) {
	var opNames []string
	if err := json.Unmarshal([]byte(row.RequestedOperationsJson), &opNames); err != nil {
		return nil, err
	}
	requestedOperations := make([]treev1.JobOperation, 0, len(opNames))
	for _, opName := range opNames {
		requestedOperations = append(requestedOperations, parseJobOperation(opName))
	}

	req := &domain.JobApprovalRequest{
		ApprovalID:          row.ApprovalID,
		JobID:               row.JobID,
		PlanID:              row.PlanID,
		Status:              row.Status,
		RequestedOperations: requestedOperations,
		Reason:              row.Reason,
		RiskTier:            row.RiskTier,
		RequestedBy:         row.RequestedBy,
		ReviewedBy:          row.ReviewedBy,
		RequestedAt:         row.RequestedAt.UTC().Format(time.RFC3339),
	}
	if row.ReviewedAt.Valid {
		req.ReviewedAt = row.ReviewedAt.Time.UTC().Format(time.RFC3339)
	}
	return req, nil
}

func parseJobOperation(value string) treev1.JobOperation {
	if enumValue, ok := treev1.JobOperation_value[value]; ok {
		return treev1.JobOperation(enumValue)
	}
	return treev1.JobOperation_JOB_OPERATION_UNSPECIFIED
}

func marshalJobOperations(ops []treev1.JobOperation) ([]byte, error) {
	names := make([]string, 0, len(ops))
	for _, op := range ops {
		names = append(names, op.String())
	}
	return json.Marshal(names)
}

func mustParseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func firstNonEmptyNonSQL(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func leftPadIndex(i int) string {
	if i < 10 {
		return "00" + strconv.Itoa(i)
	}
	if i < 100 {
		return "0" + strconv.Itoa(i)
	}
	return strconv.Itoa(i)
}

func toDocument(row sqlcgen.Document) *domain.Document {
	return &domain.Document{
		DocumentID:  row.DocumentID,
		WorkspaceID: row.WorkspaceID,
		UploadedBy:  row.UploadedBy,
		Filename:    row.Filename,
		MimeType:    row.MimeType,
		FileSize:    row.FileSize,
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *Store) UpsertJobEvaluation(jobID string, result *domain.JobEvaluationResult) bool {
	if result == nil {
		return false
	}

	now := nowTime()
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return false
	}
	defer tx.Rollback()

	qtx := s.q().WithTx(tx)
	if err := qtx.UpdateProcessingJobEvaluationState(context.Background(), sqlcgen.UpdateProcessingJobEvaluationStateParams{
		JobID:            jobID,
		EvaluationStatus: result.Status,
		UpdatedAt:        now,
	}); err != nil {
		return false
	}

	// findings can be stored in a JSON column if the schema supports it,
	// or we can just update the job status.
	// For now, we update the job evaluation state.

	return tx.Commit() == nil
}

func (s *Store) LogToolCall(ctx context.Context, jobID, toolName, inputJSON, outputJSON string, durationMs int64) error {
	job, ok := s.GetProcessingJob(jobID)
	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}

	return s.q().CreateJobMutationLog(ctx, sqlcgen.CreateJobMutationLogParams{
		MutationID:     newID(),
		JobID:          jobID,
		WorkspaceID:    job.WorkspaceID,
		TargetType:     "tool_call",
		TargetID:       toolName,
		MutationType:   "execute",
		RiskTier:       "tier_0",
		BeforeJson:     inputJSON,
		AfterJson:      outputJSON,
		ProvenanceJson: fmt.Sprintf(`{"duration_ms": %d}`, durationMs),
		CreatedAt:      nowTime(),
	})
}

func buildLikePattern(query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return "%"
	}
	return "%" + q + "%"
}

func (s *Store) SearchRelatedChunks(ctx context.Context, workspaceID, query string, limit int) ([]*domain.DocumentChunk, error) {
	if limit <= 0 {
		limit = 8
	}
	rows, err := s.q().SearchWorkspaceDocumentChunks(ctx, sqlcgen.SearchWorkspaceDocumentChunksParams{
		WorkspaceID: workspaceID,
		Pattern:     buildLikePattern(query),
		ResultLimit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	chunks := make([]*domain.DocumentChunk, 0, len(rows))
	for _, row := range rows {
		chunks = append(chunks, toDocumentChunk(row))
	}
	return chunks, nil
}

func toProcessingJob(row sqlcgen.DocumentProcessingJob) *domain.DocumentProcessingJob {
	return &domain.DocumentProcessingJob{
		JobID:            row.JobID,
		DocumentID:       row.DocumentID,
		WorkspaceID:      row.WorkspaceID,
		JobType:          parseJobType(row.JobType),
		Status:           parseJobStatus(row.Status),
		CurrentStage:     row.CurrentStage,
		ErrorMessage:     row.ErrorMessage,
		ParamsJSON:       row.ParamsJson,
		RequestedBy:      row.RequestedBy,
		CapabilityID:     row.CapabilityID,
		ExecutionPlanID:  row.ExecutionPlanID,
		PlanStatus:       row.PlanStatus,
		EvaluationStatus: row.EvaluationStatus,
		BudgetJSON:       row.BudgetJson,
		CreatedAt:        row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *Store) ListJobMutationLogs(jobID string) ([]*domain.JobMutationLog, bool) {
	rows, err := s.q().ListJobMutationLogs(context.Background(), jobID)
	if err != nil {
		return nil, false
	}
	var res []*domain.JobMutationLog
	for _, row := range rows {
		res = append(res, toMutationLog(row))
	}
	return res, true
}

func toMutationLog(row sqlcgen.JobMutationLog) *domain.JobMutationLog {
	return &domain.JobMutationLog{
		MutationID:     row.MutationID,
		JobID:          row.JobID,
		WorkspaceID:    row.WorkspaceID,
		TargetType:     row.TargetType,
		TargetID:       row.TargetID,
		MutationType:   row.MutationType,
		RiskTier:       row.RiskTier,
		BeforeJSON:     row.BeforeJson,
		AfterJSON:      row.AfterJson,
		ProvenanceJSON: row.ProvenanceJson,
		CreatedAt:      row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func parseJobStatus(s string) treev1.JobLifecycleState {
	switch s {
	case "queued":
		return treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_QUEUED
	case "running":
		return treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_RUNNING
	case "succeeded":
		return treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_SUCCEEDED
	case "failed":
		return treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_FAILED
	default:
		return treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_UNSPECIFIED
	}
}

func parseJobType(s string) treev1.JobType {
	switch s {
	case "process_document":
		return treev1.JobType_JOB_TYPE_PROCESS_DOCUMENT
	case "reprocess_document":
		return treev1.JobType_JOB_TYPE_REPROCESS_DOCUMENT
	default:
		return treev1.JobType_JOB_TYPE_UNSPECIFIED
	}
}

func formatJobType(t treev1.JobType) string {
	switch t {
	case treev1.JobType_JOB_TYPE_PROCESS_DOCUMENT:
		return "process_document"
	case treev1.JobType_JOB_TYPE_REPROCESS_DOCUMENT:
		return "reprocess_document"
	default:
		return "unspecified"
	}
}
