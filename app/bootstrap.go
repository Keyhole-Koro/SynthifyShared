package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/synthify/backend/packages/shared/config"
	"github.com/synthify/backend/packages/shared/domain"
	treev1 "github.com/synthify/backend/packages/shared/gen/synthify/tree/v1"
	"github.com/synthify/backend/packages/shared/joblog"
	"github.com/synthify/backend/packages/shared/jobstatus"
	"github.com/synthify/backend/packages/shared/repository"
	"github.com/synthify/backend/packages/shared/repository/mock"
	"github.com/synthify/backend/packages/shared/repository/postgres"
)

type AppContext struct {
	Store    Store
	Notifier jobstatus.Notifier
}

func Bootstrap(ctx context.Context, gcsURLBase, firebaseProjectID string) *AppContext {
	store := InitStore(ctx, PublicUploadURLGenerator(gcsURLBase))
	notifier := jobstatus.NewNotifier(ctx, firebaseProjectID)
	return &AppContext{
		Store:    store,
		Notifier: notifier,
	}
}

type Store interface {
	GetOrCreateAccount(ctx context.Context, userID string) (*domain.Account, error)
	GetAccount(ctx context.Context, accountID string) (*domain.Account, error)
	ListWorkspacesByUser(ctx context.Context, userID string) []*domain.Workspace
	GetWorkspace(ctx context.Context, id string) (*domain.Workspace, bool)
	IsWorkspaceAccessible(ctx context.Context, wsID, userID string) bool
	CreateWorkspace(ctx context.Context, accountID, name string) *domain.Workspace
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
	ListJobMutationLogs(ctx context.Context, jobID string) ([]*domain.JobMutationLog, bool)
	ListAllJobs(ctx context.Context) ([]*domain.DocumentProcessingJob, bool)
	LogJobEvent(ctx context.Context, e joblog.Event) error
	ListJobLogs(ctx context.Context, jobID string, pageToken string, limit int) ([]*domain.JobLog, string, bool)
	SearchJobLogs(ctx context.Context, filter domain.JobLogSearchFilter) ([]*domain.JobLog, string, error)
	ListRelatedJobLogs(ctx context.Context, scope domain.RelatedLogScope, workspaceID, documentID, jobID string, pageToken string, limit int) ([]*domain.JobLogGroup, string, error)
	RequestJobApproval(ctx context.Context, jobID, requestedBy, reason string) (*domain.JobApprovalRequest, bool)
	ApproveJobApproval(ctx context.Context, jobID, approvalID, reviewedBy string) bool
	RejectJobApproval(ctx context.Context, jobID, approvalID, reviewedBy, reason string) bool
	CreateProcessingJob(ctx context.Context, docID, workspaceID string, jobType treev1.JobType) *domain.DocumentProcessingJob
	MarkProcessingJobRunning(ctx context.Context, jobID string) bool
	UpdateProcessingJobStage(ctx context.Context, jobID, stage string) bool
	FailProcessingJob(ctx context.Context, jobID, errorMessage string) bool
	CompleteProcessingJob(ctx context.Context, jobID string) bool
	SaveDocumentChunks(ctx context.Context, documentID string, chunks []*domain.DocumentChunk) error
	LogToolCall(ctx context.Context, jobID, toolName, inputJSON, outputJSON string, durationMs int64) error
	SearchRelatedChunks(ctx context.Context, workspaceID, query string, limit int) ([]*domain.DocumentChunk, error)
	SearchRelatedChunksByVector(ctx context.Context, workspaceID string, embedding []float32, limit int) ([]*domain.DocumentChunk, error)
	GetOrCreateTree(ctx context.Context, wsID string) (*domain.Tree, error)
	GetTreeByWorkspace(ctx context.Context, wsID string) ([]*domain.Item, bool)
	GetWorkspaceRootItemID(ctx context.Context, wsID string) (string, bool)
	FindPaths(ctx context.Context, wsID, sourceItemID, targetItemID string, maxDepth, limit int) ([]*domain.Item, []domain.TreePath, bool)
	GetSubtree(ctx context.Context, rootItemID string, maxDepth int) ([]*domain.SubtreeItem, error)
	GetItem(ctx context.Context, itemID string) (*domain.Item, bool)
	CreateItem(ctx context.Context, workspaceID, label, description, parentID, createdBy string) *domain.Item
	CreateStructuredItemWithCapability(ctx context.Context, capability *domain.JobCapability, jobID, documentID, workspaceID, label string, level int, description, summaryHTML, overrideCSS, createdBy, parentID string, sourceChunkIDs []string) *domain.Item
	UpsertItemSource(ctx context.Context, itemID, documentID, chunkID, sourceText string, confidence float64) error
	UpdateItemSummaryHTMLWithCapability(ctx context.Context, capability *domain.JobCapability, jobID, itemID, summaryHTML string) bool
	ApproveAlias(ctx context.Context, wsID, canonicalItemID, aliasItemID string) bool
	RejectAlias(ctx context.Context, wsID, canonicalItemID, aliasItemID string) bool
}

func PublicUploadURLGenerator(base string) repository.UploadURLGenerator {
	return func(workspaceID, documentID string) string {
		// alt=media makes GCS return the file contents directly instead of a download page
		return fmt.Sprintf("%s/%s%%2F%s?alt=media", base, workspaceID, documentID)
	}
}

func InitStore(ctx context.Context, urlGenerator repository.UploadURLGenerator) Store {
	if dsn := config.LoadStore().DatabaseURL; dsn != "" {
		var lastErr error
		for attempt := 1; attempt <= 10; attempt++ {
			store, err := postgres.NewStore(ctx, dsn, urlGenerator)
			if err == nil {
				log.Printf("using postgres store")
				return store
			}
			lastErr = err
			log.Printf("failed to connect postgres (attempt %d/10): %v", attempt, err)
			time.Sleep(2 * time.Second)
		}
		log.Fatalf("failed to connect postgres after retries: %v", lastErr)
	}
	log.Printf("DATABASE_URL is empty, falling back to mock store")
	return mock.NewStore()
}
