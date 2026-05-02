package repository

import (
	"context"

	"github.com/synthify/backend/packages/shared/domain"
	treev1 "github.com/synthify/backend/packages/shared/gen/synthify/tree/v1"
)

type UploadURLGenerator func(workspaceID, documentID string) string

type AccountRepository interface {
	GetOrCreateAccount(ctx context.Context, userID string) (*domain.Account, error)
	GetAccount(ctx context.Context, accountID string) (*domain.Account, error)
}

type WorkspaceRepository interface {
	ListWorkspacesByUser(ctx context.Context, userID string) []*domain.Workspace
	GetWorkspace(ctx context.Context, id string) (*domain.Workspace, bool)
	IsWorkspaceAccessible(ctx context.Context, wsID, userID string) bool
	CreateWorkspace(ctx context.Context, accountID, name string) *domain.Workspace
}

type DocumentRepository interface {
	ListDocuments(ctx context.Context, wsID string) []*domain.Document
	GetDocument(ctx context.Context, id string) (*domain.Document, bool)
	GetDocumentChunks(ctx context.Context, documentID string) ([]*domain.DocumentChunk, bool)
	GetJobPlanningSignals(ctx context.Context, documentID, workspaceID, treeID string) (*domain.JobPlanningSignals, bool)
	CreateDocument(ctx context.Context, wsID, uploadedBy, filename, mimeType string, fileSize int64) (*domain.Document, string)
	GetLatestProcessingJob(ctx context.Context, docID string) (*domain.DocumentProcessingJob, bool)
	GetProcessingJob(ctx context.Context, jobID string) (*domain.DocumentProcessingJob, bool)
	GetJobCapability(ctx context.Context, jobID string) (*domain.JobCapability, bool)
	GetJobExecutionPlan(ctx context.Context, jobID string) (*domain.JobExecutionPlan, bool)
	UpsertJobExecutionPlan(ctx context.Context, jobID string, plan *domain.JobExecutionPlan) bool
	UpsertJobEvaluation(ctx context.Context, jobID string, result *domain.JobEvaluationResult) bool
	EvaluateJob(ctx context.Context, jobID string) (*domain.JobEvaluationResult, bool)
	ListJobApprovalRequests(ctx context.Context, jobID string) ([]*domain.JobApprovalRequest, bool)
	RequestJobApproval(ctx context.Context, jobID, requestedBy, reason string) (*domain.JobApprovalRequest, bool)
	ApproveJobApproval(ctx context.Context, jobID, approvalID, reviewedBy string) bool
	RejectJobApproval(ctx context.Context, jobID, approvalID, reviewedBy, reason string) bool
	SearchRelatedChunks(ctx context.Context, workspaceID, query string, limit int) ([]*domain.DocumentChunk, error)
	SearchRelatedChunksByVector(ctx context.Context, workspaceID string, embedding []float32, limit int) ([]*domain.DocumentChunk, error)
	LogToolCall(ctx context.Context, jobID, toolName, inputJSON, outputJSON string, durationMs int64) error
	CreateProcessingJob(ctx context.Context, docID, workspaceID string, jobType treev1.JobType) *domain.DocumentProcessingJob
	MarkProcessingJobRunning(ctx context.Context, jobID string) bool
	UpdateProcessingJobStage(ctx context.Context, jobID, stage string) bool
	FailProcessingJob(ctx context.Context, jobID, errorMessage string) bool
	CompleteProcessingJob(ctx context.Context, jobID string) bool
	SaveDocumentChunks(ctx context.Context, documentID string, chunks []*domain.DocumentChunk) error
	ListJobMutationLogs(ctx context.Context, jobID string) ([]*domain.JobMutationLog, bool)
	ListAllJobs(ctx context.Context) ([]*domain.DocumentProcessingJob, bool)
	ListJobLogs(ctx context.Context, jobID string, pageToken string, limit int) ([]*domain.JobLog, string, bool)
	SearchJobLogs(ctx context.Context, filter domain.JobLogSearchFilter) ([]*domain.JobLog, string, error)
	ListRelatedJobLogs(ctx context.Context, scope domain.RelatedLogScope, workspaceID, documentID, jobID string, pageToken string, limit int) ([]*domain.JobLogGroup, string, error)
}

type TreeRepository interface {
	GetOrCreateTree(ctx context.Context, wsID string) (*domain.Tree, error)
	GetTreeByWorkspace(ctx context.Context, wsID string) ([]*domain.Item, bool)
	GetWorkspaceRootItemID(ctx context.Context, wsID string) (string, bool)
	FindPaths(ctx context.Context, wsID, sourceItemID, targetItemID string, maxDepth, limit int) ([]*domain.Item, []domain.TreePath, bool)
	GetSubtree(ctx context.Context, rootItemID string, maxDepth int) ([]*domain.SubtreeItem, error)
}

type ItemRepository interface {
	GetItem(ctx context.Context, itemID string) (*domain.Item, bool)
	CreateItem(ctx context.Context, workspaceID, label, description, parentID, createdBy string) *domain.Item
	CreateStructuredItemWithCapability(ctx context.Context, capability *domain.JobCapability, jobID, documentID, workspaceID, label string, level int, description, summaryHTML, overrideCSS, createdBy, parentID string, sourceChunkIDs []string) *domain.Item
	UpsertItemSource(ctx context.Context, itemID, documentID, chunkID, sourceText string, confidence float64) error
	UpdateItemSummaryHTMLWithCapability(ctx context.Context, capability *domain.JobCapability, jobID, itemID, summaryHTML string) bool
	ApproveAlias(ctx context.Context, wsID, canonicalItemID, aliasItemID string) bool
	RejectAlias(ctx context.Context, wsID, canonicalItemID, aliasItemID string) bool
}
