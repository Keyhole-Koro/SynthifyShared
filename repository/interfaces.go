package repository

import (
	"github.com/Keyhole-Koro/SynthifyShared/domain"
	treev1 "github.com/Keyhole-Koro/SynthifyShared/gen/synthify/tree/v1"
)

type UploadURLGenerator func(workspaceID, documentID string) string

type AccountRepository interface {
	GetOrCreateAccount(userID string) (*domain.Account, error)
	GetAccount(accountID string) (*domain.Account, error)
}

type WorkspaceRepository interface {
	ListWorkspacesByUser(userID string) []*domain.Workspace
	GetWorkspace(id string) (*domain.Workspace, bool)
	IsWorkspaceAccessible(wsID, userID string) bool
	CreateWorkspace(accountID, name string) *domain.Workspace
}

type DocumentRepository interface {
	ListDocuments(wsID string) []*domain.Document
	GetDocument(id string) (*domain.Document, bool)
	GetDocumentChunks(documentID string) ([]*domain.DocumentChunk, bool)
	GetJobPlanningSignals(documentID, workspaceID, treeID string) (*domain.JobPlanningSignals, bool)
	CreateDocument(wsID, uploadedBy, filename, mimeType string, fileSize int64) (*domain.Document, string)
	GetLatestProcessingJob(docID string) (*domain.DocumentProcessingJob, bool)
	GetProcessingJob(jobID string) (*domain.DocumentProcessingJob, bool)
	GetJobCapability(jobID string) (*domain.JobCapability, bool)
	GetJobExecutionPlan(jobID string) (*domain.JobExecutionPlan, bool)
	UpsertJobExecutionPlan(jobID string, plan *domain.JobExecutionPlan) bool
	EvaluateJob(jobID string) (*domain.JobEvaluationResult, bool)
	ListJobApprovalRequests(jobID string) ([]*domain.JobApprovalRequest, bool)
	RequestJobApproval(jobID, requestedBy, reason string) (*domain.JobApprovalRequest, bool)
	ApproveJobApproval(jobID, approvalID, reviewedBy string) bool
	RejectJobApproval(jobID, approvalID, reviewedBy, reason string) bool
	CreateProcessingJob(docID, workspaceID string, jobType treev1.JobType) *domain.DocumentProcessingJob
	MarkProcessingJobRunning(jobID string) bool
	UpdateProcessingJobStage(jobID, stage string) bool
	FailProcessingJob(jobID, errorMessage string) bool
	CompleteProcessingJob(jobID string) bool
	SaveDocumentChunks(documentID string, chunks []*domain.DocumentChunk) error
}

type TreeRepository interface {
	GetOrCreateTree(wsID string) (*domain.Tree, error)
	GetTreeByWorkspace(wsID string) ([]*domain.Item, bool)
	GetWorkspaceRootItemID(wsID string) (string, bool)
	FindPaths(wsID, sourceItemID, targetItemID string, maxDepth, limit int) ([]*domain.Item, []domain.TreePath, bool)
	GetSubtree(rootItemID string, maxDepth int) ([]*domain.SubtreeItem, error)
}

type ItemRepository interface {
	GetItem(itemID string) (*domain.Item, bool)
	CreateItem(workspaceID, label, description, parentID, createdBy string) *domain.Item
	CreateStructuredItemWithCapability(capability *domain.JobCapability, jobID, documentID, workspaceID, label string, level int, description, summaryHTML, createdBy, parentID string, sourceChunkIDs []string) *domain.Item
	UpsertItemSource(itemID, documentID, chunkID, sourceText string, confidence float64) error
	UpdateItemSummaryHTMLWithCapability(capability *domain.JobCapability, jobID, itemID, summaryHTML string) bool
	ApproveAlias(wsID, canonicalItemID, aliasItemID string) bool
	RejectAlias(wsID, canonicalItemID, aliasItemID string) bool
}
