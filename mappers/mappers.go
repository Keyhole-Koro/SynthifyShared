package mappers

import (
	"github.com/Keyhole-Koro/SynthifyShared/domain"
	treev1 "github.com/Keyhole-Koro/SynthifyShared/gen/synthify/tree/v1"
)

func ToProtoJob(job *domain.DocumentProcessingJob) *treev1.Job {
	if job == nil {
		return nil
	}
	return &treev1.Job{
		JobId:        job.JobID,
		DocumentId:   job.DocumentID,
		Type:         job.JobType,
		Status:       job.Status,
		CreatedAt:    job.CreatedAt,
		CompletedAt:  job.UpdatedAt,
		ErrorMessage: job.ErrorMessage,
	}
}

func ToProtoApprovalRequest(req *domain.JobApprovalRequest) *treev1.JobApprovalRequest {
	if req == nil {
		return nil
	}
	return &treev1.JobApprovalRequest{
		ApprovalId:          req.ApprovalID,
		JobId:               req.JobID,
		PlanId:              req.PlanID,
		Status:              req.Status,
		RequestedOperations: req.RequestedOperations,
		Reason:              req.Reason,
		RiskTier:            req.RiskTier,
		RequestedBy:         req.RequestedBy,
		ReviewedBy:          req.ReviewedBy,
		RequestedAt:         req.RequestedAt,
		ReviewedAt:          req.ReviewedAt,
	}
}

func ToProtoMutationLog(log *domain.JobMutationLog) *treev1.JobMutationLog {
	if log == nil {
		return nil
	}
	return &treev1.JobMutationLog{
		MutationId:     log.MutationID,
		JobId:          log.JobID,
		TargetType:     log.TargetType,
		TargetId:       log.TargetID,
		MutationType:   log.MutationType,
		RiskTier:       log.RiskTier,
		BeforeJson:     log.BeforeJSON,
		AfterJson:      log.AfterJSON,
		ProvenanceJson: log.ProvenanceJSON,
		CreatedAt:      log.CreatedAt,
	}
}

func ToProtoExecutionPlan(plan *domain.JobExecutionPlan) *treev1.JobExecutionPlan {
	if plan == nil {
		return nil
	}
	return &treev1.JobExecutionPlan{
		PlanId:    plan.PlanID,
		JobId:     plan.JobID,
		Status:    plan.Status,
		Summary:   plan.Summary,
		PlanJson:  plan.PlanJSON,
		CreatedBy: plan.CreatedBy,
		CreatedAt: plan.CreatedAt,
		UpdatedAt: plan.UpdatedAt,
	}
}

func ToProtoDocument(doc *domain.Document) *treev1.Document {
	if doc == nil {
		return nil
	}
	return &treev1.Document{
		DocumentId:  doc.DocumentID,
		WorkspaceId: doc.WorkspaceID,
		UploadedBy:  doc.UploadedBy,
		Filename:    doc.Filename,
		MimeType:    doc.MimeType,
		FileSize:    doc.FileSize,
		Status:      treev1.DocumentLifecycleState_DOCUMENT_LIFECYCLE_STATE_UPLOADED,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.CreatedAt,
	}
}

func ToProtoWorkspace(ws *domain.Workspace) *treev1.Workspace {
	if ws == nil {
		return nil
	}
	return &treev1.Workspace{
		WorkspaceId: ws.WorkspaceID,
		Name:        ws.Name,
		OwnerId:     ws.AccountID,
		CreatedAt:   ws.CreatedAt,
	}
}

func ToProtoItem(item *domain.Item) *treev1.Item {
	if item == nil {
		return nil
	}
	return &treev1.Item{
		Id:              item.ItemID,
		Label:           item.Label,
		Level:           int32(item.Level),
		Description:     item.Description,
		SummaryHtml:     item.SummaryHTML,
		CreatedAt:       item.CreatedAt,
		ParentId:        item.ParentID,
		ChildIds:        item.ChildIDs,
		Scope:           item.Scope,
		GovernanceState: item.GovernanceState,
	}
}
