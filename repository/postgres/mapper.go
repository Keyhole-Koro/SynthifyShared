package postgres

import (
	"time"

	"github.com/Keyhole-Koro/SynthifyShared/domain"
	treev1 "github.com/Keyhole-Koro/SynthifyShared/gen/synthify/tree/v1"
	"github.com/Keyhole-Koro/SynthifyShared/repository/postgres/sqlcgen"
)

func toAccount(row sqlcgen.Account) *domain.Account {
	return &domain.Account{
		AccountID:          row.AccountID,
		Name:               row.Name,
		Plan:               row.Plan,
		StorageQuotaBytes:  row.StorageQuotaBytes,
		StorageUsedBytes:   row.StorageUsedBytes,
		MaxFileSizeBytes:   row.MaxFileSizeBytes,
		MaxUploadsPerFiveH: int64(row.MaxUploadsPer5h),
		MaxUploadsPerWeek:  int64(row.MaxUploadsPer1week),
		CreatedAt:          row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toWorkspace(row sqlcgen.Workspace) *domain.Workspace {
	return &domain.Workspace{
		WorkspaceID: row.WorkspaceID,
		AccountID:   row.AccountID,
		Name:        row.Name,
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
	}
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

func toProcessingJob(row sqlcgen.DocumentProcessingJob) *domain.DocumentProcessingJob {
	return &domain.DocumentProcessingJob{
		JobID:        row.JobID,
		DocumentID:   row.DocumentID,
		WorkspaceID:  row.WorkspaceID,
		JobType:      parseJobType(row.JobType),
		Status:       parseJobStatus(row.Status),
		CurrentStage: row.CurrentStage,
		ErrorMessage: row.ErrorMessage,
		ParamsJSON:   row.ParamsJson,
		CreatedAt:    row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toItemFromItemRow(row sqlcgen.ListItemsByWorkspaceRow) *domain.Item {
	return &domain.Item{
		ItemID:          row.ID,
		WorkspaceID:     row.WorkspaceID,
		ParentID:        row.ParentID.String,
		Label:           row.Label,
		Level:           int(row.Level),
		Description:     row.Description,
		SummaryHTML:     row.SummaryHtml,
		CreatedBy:       row.CreatedBy,
		GovernanceState: treev1.ItemGovernanceState_ITEM_GOVERNANCE_STATE_SYSTEM_GENERATED,
		CreatedAt:       row.CreatedAt.UTC().Format(time.RFC3339),
		Scope:           treev1.TreeProjectionScope_TREE_PROJECTION_SCOPE_DOCUMENT,
	}
}

func toItemFromGetRow(row sqlcgen.GetItemRow) *domain.Item {
	return &domain.Item{
		ItemID:          row.ID,
		WorkspaceID:     row.WorkspaceID,
		ParentID:        row.ParentID.String,
		Label:           row.Label,
		Level:           int(row.Level),
		Description:     row.Description,
		SummaryHTML:     row.SummaryHtml,
		CreatedBy:       row.CreatedBy,
		GovernanceState: treev1.ItemGovernanceState_ITEM_GOVERNANCE_STATE_SYSTEM_GENERATED,
		CreatedAt:       row.CreatedAt.UTC().Format(time.RFC3339),
		Scope:           treev1.TreeProjectionScope_TREE_PROJECTION_SCOPE_DOCUMENT,
	}
}

func parseJobStatus(s string) treev1.JobLifecycleState {
	switch s {
	case "queued":
		return treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_QUEUED
	case "running":
		return treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_RUNNING
	case "completed", "succeeded":
		return treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_SUCCEEDED
	case "failed":
		return treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_FAILED
	default:
		return treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_QUEUED
	}
}

func parseJobType(s string) treev1.JobType {
	switch s {
	case "process_document":
		return treev1.JobType_JOB_TYPE_PROCESS_DOCUMENT
	case "reprocess_document":
		return treev1.JobType_JOB_TYPE_REPROCESS_DOCUMENT
	default:
		return treev1.JobType_JOB_TYPE_PROCESS_DOCUMENT
	}
}

func parseGovernanceState(s string) treev1.ItemGovernanceState {
	switch s {
	case "system_generated":
		return treev1.ItemGovernanceState_ITEM_GOVERNANCE_STATE_SYSTEM_GENERATED
	case "pending_review":
		return treev1.ItemGovernanceState_ITEM_GOVERNANCE_STATE_PENDING_REVIEW
	case "human_curated":
		return treev1.ItemGovernanceState_ITEM_GOVERNANCE_STATE_HUMAN_CURATED
	case "locked":
		return treev1.ItemGovernanceState_ITEM_GOVERNANCE_STATE_LOCKED
	default:
		return treev1.ItemGovernanceState_ITEM_GOVERNANCE_STATE_SYSTEM_GENERATED
	}
}
