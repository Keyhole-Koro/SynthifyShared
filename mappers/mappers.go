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
