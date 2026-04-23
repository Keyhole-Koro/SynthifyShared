package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Keyhole-Koro/SynthifyShared/domain"
	treev1 "github.com/Keyhole-Koro/SynthifyShared/gen/synthify/tree/v1"
	"github.com/Keyhole-Koro/SynthifyShared/repository/postgres/sqlcgen"
)

func (s *Store) GetItem(itemID string) (*domain.Item, bool) {
	row, err := s.q().GetItem(context.Background(), itemID)
	if err != nil {
		return nil, false
	}
	return toItemFromGetRow(row), true
}

func (s *Store) CreateItem(workspaceID, label, description, parentID, createdBy string) *domain.Item {
	return s.createStructuredItemDirect(workspaceID, label, 0, description, "", createdBy, parentID)
}

func (s *Store) createStructuredItemDirect(workspaceID, label string, level int, description, summaryHTML, createdBy, parentID string) *domain.Item {
	createdAt := nowTime()
	item := &domain.Item{
		ItemID:      newID(),
		WorkspaceID: workspaceID,
		ParentID:    parentID,
		Label:       label,
		Level:       level,
		Description: description,
		SummaryHTML: summaryHTML,
		CreatedBy:   createdBy,
		CreatedAt:   createdAt.Format(time.RFC3339),
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil
	}
	defer tx.Rollback()

	if err := s.q().WithTx(tx).CreateItem(context.Background(), sqlcgen.CreateItemParams{
		ID:          item.ItemID,
		WorkspaceID: workspaceID,
		ParentID: sql.NullString{
			String: parentID,
			Valid:  parentID != "",
		},
		Label:       item.Label,
		Level:       int32(item.Level),
		Description: item.Description,
		SummaryHtml: item.SummaryHTML,
		CreatedBy:   item.CreatedBy,
		CreatedAt:   createdAt,
	}); err != nil {
		return nil
	}
	if err := tx.Commit(); err != nil {
		return nil
	}
	_ = s.q().UpdateTreeTimestamp(context.Background(), sqlcgen.UpdateTreeTimestampParams{
		ID:        item.ItemID,
		UpdatedAt: nowTime(),
	})
	return item
}

func (s *Store) CreateStructuredItemWithCapability(capability *domain.JobCapability, jobID, documentID, workspaceID, label string, level int, description, summaryHTML, createdBy, parentID string, sourceChunkIDs []string) *domain.Item {
	if !s.canMutateTree(capability, treev1.JobOperation_JOB_OPERATION_CREATE_ITEM, workspaceID, documentID) {
		return nil
	}
	if capability.MaxItemCreations > 0 && s.countJobMutations(jobID, "item") >= capability.MaxItemCreations {
		return nil
	}

	createdAt := nowTime()
	item := &domain.Item{
		ItemID:            newID(),
		WorkspaceID:       workspaceID,
		ParentID:          parentID,
		Label:             label,
		Level:             level,
		Description:       description,
		SummaryHTML:       summaryHTML,
		CreatedBy:         createdBy,
		GovernanceState:   treev1.ItemGovernanceState_ITEM_GOVERNANCE_STATE_SYSTEM_GENERATED,
		LastMutationJobID: jobID,
		CreatedAt:         createdAt.Format(time.RFC3339),
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil
	}
	defer tx.Rollback()

	qtx := s.q().WithTx(tx)
	if err := qtx.CreateStructuredItem(context.Background(), sqlcgen.CreateStructuredItemParams{
		ID:          item.ItemID,
		WorkspaceID: workspaceID,
		ParentID: sql.NullString{
			String: parentID,
			Valid:  parentID != "",
		},
		Label:             item.Label,
		Level:             int32(item.Level),
		Description:       item.Description,
		SummaryHtml:       item.SummaryHTML,
		CreatedBy:         item.CreatedBy,
		GovernanceState:   "system_generated",
		LastMutationJobID: item.LastMutationJobID,
		CreatedAt:         createdAt,
	}); err != nil {
		return nil
	}
	if err := s.logMutationTx(tx, &domain.JobMutationLog{
		MutationID:     newID(),
		JobID:          jobID,
		CapabilityID:   capability.CapabilityID,
		WorkspaceID:    workspaceID,
		TargetType:     "item",
		TargetID:       item.ItemID,
		MutationType:   "append",
		RiskTier:       "tier_1",
		BeforeJSON:     "{}",
		AfterJSON:      mustJSON(item),
		ProvenanceJSON: mustJSON(map[string]any{"document_id": documentID, "source_chunk_ids": sourceChunkIDs}),
		CreatedAt:      createdAt.Format(time.RFC3339),
	}); err != nil {
		return nil
	}
	if err := tx.Commit(); err != nil {
		return nil
	}
	return item
}

func (s *Store) UpsertItemSource(itemID, documentID, chunkID, sourceText string, confidence float64) error {
	return s.q().UpsertItemSource(context.Background(), sqlcgen.UpsertItemSourceParams{
		ItemID:     itemID,
		DocumentID: documentID,
		ChunkID:    chunkID,
		SourceText: sourceText,
		Confidence: sql.NullFloat64{Float64: confidence, Valid: confidence > 0},
	})
}

func (s *Store) UpdateItemSummaryHTMLWithCapability(capability *domain.JobCapability, jobID, itemID, summaryHTML string) bool {
	if capability == nil || !capability.Allows(treev1.JobOperation_JOB_OPERATION_UPDATE_ITEM) || capability.IsExpired(nowTime()) {
		return false
	}

	row, err := s.q().GetItemSummaryUpdateContext(context.Background(), itemID)
	if err != nil {
		return false
	}
	if capability.WorkspaceID != "" && capability.WorkspaceID != row.WorkspaceID {
		return false
	}
	if !capability.AllowsItem(itemID) {
		return false
	}
	if row.GovernanceState == "human_curated" || row.GovernanceState == "locked" {
		return false
	}

	now := nowTime()
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return false
	}
	defer tx.Rollback()

	qtx := s.q().WithTx(tx)
	affected, err := qtx.UpdateItemSummaryAndMutation(context.Background(), sqlcgen.UpdateItemSummaryAndMutationParams{
		ID:                itemID,
		SummaryHtml:       summaryHTML,
		LastMutationJobID: jobID,
		UpdatedAt:         now,
	})
	if err != nil || affected == 0 {
		return false
	}
	if err := s.logMutationTx(tx, &domain.JobMutationLog{
		MutationID:     newID(),
		JobID:          jobID,
		CapabilityID:   capability.CapabilityID,
		WorkspaceID:    row.WorkspaceID,
		TargetType:     "item",
		TargetID:       itemID,
		MutationType:   "revise",
		RiskTier:       "tier_1",
		BeforeJSON:     mustJSON(map[string]string{"summary_html": row.SummaryHtml}),
		AfterJSON:      mustJSON(map[string]string{"summary_html": summaryHTML}),
		ProvenanceJSON: mustJSON(map[string]any{"field": "summary_html"}),
		CreatedAt:      now.Format(time.RFC3339),
	}); err != nil {
		return false
	}
	return tx.Commit() == nil
}

func (s *Store) ApproveAlias(wsID, canonicalItemID, aliasItemID string) bool {
	return s.q().UpsertApprovedAlias(context.Background(), sqlcgen.UpsertApprovedAliasParams{
		WorkspaceID:     wsID,
		CanonicalItemID: canonicalItemID,
		AliasItemID:     aliasItemID,
		UpdatedAt:       nowTime(),
	}) == nil
}

func (s *Store) RejectAlias(wsID, canonicalItemID, aliasItemID string) bool {
	return s.q().UpsertRejectedAlias(context.Background(), sqlcgen.UpsertRejectedAliasParams{
		WorkspaceID:     wsID,
		CanonicalItemID: canonicalItemID,
		AliasItemID:     aliasItemID,
		UpdatedAt:       nowTime(),
	}) == nil
}

func (s *Store) canMutateTree(capability *domain.JobCapability, op treev1.JobOperation, workspaceID, documentID string) bool {
	if capability == nil || capability.IsExpired(nowTime()) {
		return false
	}
	if !capability.Allows(op) {
		return false
	}
	if capability.WorkspaceID != "" && capability.WorkspaceID != workspaceID {
		return false
	}
	return capability.AllowsDocument(documentID)
}

func (s *Store) countJobMutations(jobID, targetType string) int {
	count, err := s.q().CountJobMutationsByTarget(context.Background(), sqlcgen.CountJobMutationsByTargetParams{
		JobID:      jobID,
		TargetType: targetType,
	})
	if err != nil {
		return 0
	}
	return int(count)
}

func (s *Store) logMutationTx(tx *sql.Tx, entry *domain.JobMutationLog) error {
	createdAt, err := time.Parse(time.RFC3339, entry.CreatedAt)
	if err != nil {
		createdAt = nowTime()
	}
	return s.q().WithTx(tx).InsertJobMutationLog(context.Background(), sqlcgen.InsertJobMutationLogParams{
		MutationID:     entry.MutationID,
		JobID:          entry.JobID,
		PlanID:         entry.PlanID,
		CapabilityID:   entry.CapabilityID,
		WorkspaceID:    entry.WorkspaceID,
		TargetType:     entry.TargetType,
		TargetID:       entry.TargetID,
		MutationType:   entry.MutationType,
		RiskTier:       entry.RiskTier,
		BeforeJson:     entry.BeforeJSON,
		AfterJson:      entry.AfterJSON,
		ProvenanceJson: entry.ProvenanceJSON,
		CreatedAt:      createdAt,
	})
}

func mustJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
