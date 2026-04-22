package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Keyhole-Koro/SynthifyShared/domain"
	"github.com/Keyhole-Koro/SynthifyShared/repository/postgres/sqlcgen"
)

func (s *Store) GetNode(nodeID string) (*domain.Node, []*domain.Edge, bool) {
	ctx := context.Background()
	row, err := s.q().GetNode(ctx, nodeID)
	if err != nil {
		return nil, nil, false
	}
	edgeRows, err := s.q().ListNodeEdges(ctx, nodeID)
	if err != nil {
		return nil, nil, false
	}
	var edges []*domain.Edge
	for _, edgeRow := range edgeRows {
		edges = append(edges, toEdge(edgeRow))
	}
	return toNode(row), edges, true
}

func (s *Store) CreateNode(graphID, label, description, parentNodeID, createdBy string) *domain.Node {
	node := s.createStructuredNodeDirect(graphID, label, 0, description, "", createdBy)
	if node == nil || parentNodeID == "" {
		return node
	}
	if s.createEdgeDirect(graphID, parentNodeID, node.NodeID, "hierarchical", "") == nil {
		return nil
	}
	return node
}

func (s *Store) createStructuredNodeDirect(graphID, label string, level int, description, summaryHTML, createdBy string) *domain.Node {
	createdAt := nowTime()
	node := &domain.Node{
		NodeID:      newID(),
		GraphID:     graphID,
		Label:       label,
		Level:       level,
		Description: description,
		SummaryHTML: summaryHTML,
		CreatedBy:   createdBy,
		CreatedAt:   createdAt.Format(time.RFC3339),
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil
	}
	defer tx.Rollback()
	qtx := s.q().WithTx(tx)
	ctx := context.Background()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO nodes (node_id, graph_id, label, category, level, description, summary_html, created_by, created_at)
		VALUES ($1, $2, $3, '', $4, $5, $6, $7, $8)
	`, node.NodeID, node.GraphID, node.Label, node.Level, node.Description, node.SummaryHTML, node.CreatedBy, createdAt); err != nil {
		return nil
	}
	if summaryHTML != "" {
		node.SummaryHTML = summaryHTML
	}
	_ = qtx.UpdateGraphTimestamp(ctx, sqlcgen.UpdateGraphTimestampParams{
		GraphID:   graphID,
		UpdatedAt: nowTime(),
	})
	if err := tx.Commit(); err != nil {
		return nil
	}
	return node
}

func (s *Store) CreateStructuredNodeWithCapability(capability *domain.JobCapability, jobID, documentID, graphID, label string, level int, description, summaryHTML, createdBy string, sourceChunkIDs []string) *domain.Node {
	if !s.canMutateGraph(capability, domain.JobOperationCreateNode, graphID, documentID) {
		return nil
	}
	if capability.MaxNodeCreations > 0 && s.countJobMutations(jobID, "node") >= capability.MaxNodeCreations {
		return nil
	}
	createdAt := nowTime()
	node := &domain.Node{
		NodeID:            newID(),
		GraphID:           graphID,
		Label:             label,
		Level:             level,
		Description:       description,
		SummaryHTML:       summaryHTML,
		CreatedBy:         createdBy,
		GovernanceState:   string(domain.NodeGovernanceStateSystemGenerated),
		LastMutationJobID: jobID,
		CreatedAt:         createdAt.Format(time.RFC3339),
	}
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(context.Background(), `
		INSERT INTO nodes (
			node_id, graph_id, label, category, level, description, summary_html, created_by, governance_state, last_mutation_job_id, created_at
		) VALUES ($1, $2, $3, '', $4, $5, $6, $7, $8, $9, $10)
	`, node.NodeID, node.GraphID, node.Label, node.Level, node.Description, node.SummaryHTML, node.CreatedBy, node.GovernanceState, node.LastMutationJobID, createdAt); err != nil {
		return nil
	}
	if err := s.logMutationTx(tx, &domain.JobMutationLog{
		MutationID:     newID(),
		JobID:          jobID,
		CapabilityID:   capability.CapabilityID,
		GraphID:        graphID,
		TargetType:     "node",
		TargetID:       node.NodeID,
		MutationType:   "append",
		RiskTier:       "tier_1",
		BeforeJSON:     "{}",
		AfterJSON:      mustJSON(node),
		ProvenanceJSON: mustJSON(map[string]any{"document_id": documentID, "source_chunk_ids": sourceChunkIDs}),
		CreatedAt:      createdAt.Format(time.RFC3339),
	}); err != nil {
		return nil
	}
	if _, err := tx.ExecContext(context.Background(), `UPDATE graphs SET updated_at = $2 WHERE graph_id = $1`, graphID, nowTime()); err != nil {
		return nil
	}
	if err := tx.Commit(); err != nil {
		return nil
	}
	return node
}

func (s *Store) createEdgeDirect(graphID, sourceNodeID, targetNodeID, edgeType, description string) *domain.Edge {
	createdAt := nowTime()
	edge := &domain.Edge{
		EdgeID:       newID(),
		GraphID:      graphID,
		SourceNodeID: sourceNodeID,
		TargetNodeID: targetNodeID,
		EdgeType:     edgeType,
		Description:  description,
		CreatedAt:    createdAt.Format(time.RFC3339),
	}
	if _, err := s.db.ExecContext(context.Background(), `
		INSERT INTO edges (edge_id, graph_id, source_node_id, target_node_id, edge_type, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, edge.EdgeID, edge.GraphID, edge.SourceNodeID, edge.TargetNodeID, edge.EdgeType, edge.Description, createdAt); err != nil {
		return nil
	}
	_ = s.q().UpdateGraphTimestamp(context.Background(), sqlcgen.UpdateGraphTimestampParams{
		GraphID:   graphID,
		UpdatedAt: nowTime(),
	})
	return edge
}

func (s *Store) CreateEdgeWithCapability(capability *domain.JobCapability, jobID, documentID, graphID, sourceNodeID, targetNodeID, edgeType, description string, sourceChunkIDs []string) *domain.Edge {
	if !s.canMutateGraph(capability, domain.JobOperationCreateEdge, graphID, documentID) {
		return nil
	}
	if !capability.AllowsNode(sourceNodeID) || !capability.AllowsNode(targetNodeID) {
		return nil
	}
	if capability.MaxEdgeMutations > 0 && s.countJobMutations(jobID, "edge") >= capability.MaxEdgeMutations {
		return nil
	}
	createdAt := nowTime()
	edge := &domain.Edge{
		EdgeID:       newID(),
		GraphID:      graphID,
		SourceNodeID: sourceNodeID,
		TargetNodeID: targetNodeID,
		EdgeType:     edgeType,
		Description:  description,
		CreatedAt:    createdAt.Format(time.RFC3339),
	}
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(context.Background(), `
		INSERT INTO edges (edge_id, graph_id, source_node_id, target_node_id, edge_type, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, edge.EdgeID, edge.GraphID, edge.SourceNodeID, edge.TargetNodeID, edge.EdgeType, edge.Description, createdAt); err != nil {
		return nil
	}
	if err := s.logMutationTx(tx, &domain.JobMutationLog{
		MutationID:     newID(),
		JobID:          jobID,
		CapabilityID:   capability.CapabilityID,
		GraphID:        graphID,
		TargetType:     "edge",
		TargetID:       edge.EdgeID,
		MutationType:   "append",
		RiskTier:       "tier_1",
		BeforeJSON:     "{}",
		AfterJSON:      mustJSON(edge),
		ProvenanceJSON: mustJSON(map[string]any{"document_id": documentID, "source_chunk_ids": sourceChunkIDs}),
		CreatedAt:      createdAt.Format(time.RFC3339),
	}); err != nil {
		return nil
	}
	if _, err := tx.ExecContext(context.Background(), `UPDATE graphs SET updated_at = $2 WHERE graph_id = $1`, graphID, nowTime()); err != nil {
		return nil
	}
	if err := tx.Commit(); err != nil {
		return nil
	}
	return edge
}

func (s *Store) UpsertNodeSource(nodeID, documentID, chunkID, sourceText string, confidence float64) error {
	return s.q().UpsertNodeSource(context.Background(), sqlcgen.UpsertNodeSourceParams{
		NodeID:     nodeID,
		DocumentID: documentID,
		ChunkID:    chunkID,
		SourceText: sourceText,
		Confidence: sql.NullFloat64{Float64: confidence, Valid: confidence > 0},
	})
}

func (s *Store) UpsertEdgeSource(edgeID, documentID, chunkID, sourceText string, confidence float64) error {
	return s.q().UpsertEdgeSource(context.Background(), sqlcgen.UpsertEdgeSourceParams{
		EdgeID:     edgeID,
		DocumentID: documentID,
		ChunkID:    chunkID,
		SourceText: sourceText,
		Confidence: sql.NullFloat64{Float64: confidence, Valid: confidence > 0},
	})
}

func (s *Store) UpdateNodeSummaryHTMLWithCapability(capability *domain.JobCapability, jobID, nodeID, summaryHTML string) bool {
	if capability == nil || !capability.Allows(domain.JobOperationUpdateNode) || capability.IsExpired(nowTime()) {
		return false
	}
	row := s.db.QueryRowContext(context.Background(), `
		SELECT graph_id, COALESCE(summary_html, ''), COALESCE(governance_state, 'system_generated')
		FROM nodes
		WHERE node_id = $1
	`, nodeID)
	var graphID, beforeSummary, governanceState string
	if err := row.Scan(&graphID, &beforeSummary, &governanceState); err != nil {
		return false
	}
	if capability.GraphID != "" && capability.GraphID != graphID {
		return false
	}
	if !capability.AllowsNode(nodeID) {
		return false
	}
	if governanceState == string(domain.NodeGovernanceStateHumanCurated) || governanceState == string(domain.NodeGovernanceStateLocked) {
		return false
	}
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return false
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(context.Background(), `
		UPDATE nodes
		SET summary_html = $2, last_mutation_job_id = $3
		WHERE node_id = $1
	`, nodeID, summaryHTML, jobID)
	if err != nil {
		return false
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return false
	}
	if err := s.logMutationTx(tx, &domain.JobMutationLog{
		MutationID:     newID(),
		JobID:          jobID,
		CapabilityID:   capability.CapabilityID,
		GraphID:        graphID,
		TargetType:     "node",
		TargetID:       nodeID,
		MutationType:   "revise",
		RiskTier:       "tier_1",
		BeforeJSON:     mustJSON(map[string]string{"summary_html": beforeSummary}),
		AfterJSON:      mustJSON(map[string]string{"summary_html": summaryHTML}),
		ProvenanceJSON: mustJSON(map[string]any{"field": "summary_html"}),
		CreatedAt:      nowTime().Format(time.RFC3339),
	}); err != nil {
		return false
	}
	return tx.Commit() == nil
}

func (s *Store) ApproveAlias(wsID, canonicalNodeID, aliasNodeID string) bool {
	return s.q().UpsertApprovedAlias(context.Background(), sqlcgen.UpsertApprovedAliasParams{
		WorkspaceID:     wsID,
		CanonicalNodeID: canonicalNodeID,
		AliasNodeID:     aliasNodeID,
		UpdatedAt:       nowTime(),
	}) == nil
}

func (s *Store) RejectAlias(wsID, canonicalNodeID, aliasNodeID string) bool {
	return s.q().UpsertRejectedAlias(context.Background(), sqlcgen.UpsertRejectedAliasParams{
		WorkspaceID:     wsID,
		CanonicalNodeID: canonicalNodeID,
		AliasNodeID:     aliasNodeID,
		UpdatedAt:       nowTime(),
	}) == nil
}

func (s *Store) canMutateGraph(capability *domain.JobCapability, op domain.JobOperation, graphID, documentID string) bool {
	if capability == nil || capability.IsExpired(nowTime()) {
		return false
	}
	if !capability.Allows(op) {
		return false
	}
	if capability.GraphID != "" && capability.GraphID != graphID {
		return false
	}
	return capability.AllowsDocument(documentID)
}

func (s *Store) countJobMutations(jobID, targetType string) int {
	row := s.db.QueryRowContext(context.Background(), `
		SELECT COUNT(*)
		FROM job_mutation_logs
		WHERE job_id = $1 AND target_type = $2
	`, jobID, targetType)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Store) logMutationTx(tx *sql.Tx, entry *domain.JobMutationLog) error {
	createdAt, err := time.Parse(time.RFC3339, entry.CreatedAt)
	if err != nil {
		createdAt = nowTime()
	}
	_, err = tx.ExecContext(context.Background(), `
		INSERT INTO job_mutation_logs (
			mutation_id, job_id, plan_id, capability_id, graph_id, target_type, target_id, mutation_type,
			risk_tier, before_json, after_json, provenance_json, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, entry.MutationID, entry.JobID, entry.PlanID, entry.CapabilityID, entry.GraphID, entry.TargetType, entry.TargetID, entry.MutationType, entry.RiskTier, entry.BeforeJSON, entry.AfterJSON, entry.ProvenanceJSON, createdAt)
	return err
}

func mustJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
