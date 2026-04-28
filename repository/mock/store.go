package mock

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Keyhole-Koro/SynthifyShared/domain"
	treev1 "github.com/Keyhole-Koro/SynthifyShared/gen/synthify/tree/v1"
	"github.com/Keyhole-Koro/SynthifyShared/repository"
)

type Store struct {
	mu           sync.RWMutex
	accounts     map[string]*domain.Account
	workspaces   map[string]*domain.Workspace
	documents    map[string]*domain.Document
	jobs         map[string]*domain.DocumentProcessingJob
	capabilities map[string]*domain.JobCapability
	plans        map[string]*domain.JobExecutionPlan
	approvals    map[string][]*domain.JobApprovalRequest
	items        map[string]map[string]*domain.Item // workspaceID -> itemID -> Item
	sources      map[string][]*domain.ItemSource
	chunks       map[string][]*domain.DocumentChunk
}

func NewStore() *Store {
	return &Store{
		accounts:     make(map[string]*domain.Account),
		workspaces:   make(map[string]*domain.Workspace),
		documents:    make(map[string]*domain.Document),
		jobs:         make(map[string]*domain.DocumentProcessingJob),
		capabilities: make(map[string]*domain.JobCapability),
		plans:        make(map[string]*domain.JobExecutionPlan),
		approvals:    make(map[string][]*domain.JobApprovalRequest),
		items:        make(map[string]map[string]*domain.Item),
		sources:      make(map[string][]*domain.ItemSource),
		chunks:       make(map[string][]*domain.DocumentChunk),
	}
}

// AccountRepository
func (s *Store) GetOrCreateAccount(userID string) (*domain.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.accounts[userID]; ok {
		return a, nil
	}
	a := &domain.Account{
		AccountID: userID,
		Name:      "User " + userID,
		Plan:      "anonymous",
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	s.accounts[userID] = a
	return a, nil
}

func (s *Store) GetAccount(id string) (*domain.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if a, ok := s.accounts[id]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("account not found")
}

// WorkspaceRepository
func (s *Store) ListWorkspacesByUser(userID string) []*domain.Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []*domain.Workspace
	for _, w := range s.workspaces {
		res = append(res, w)
	}
	return res
}

func (s *Store) GetWorkspace(id string) (*domain.Workspace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.workspaces[id]
	return w, ok
}

func (s *Store) IsWorkspaceAccessible(wsID, userID string) bool { return true }

func (s *Store) CreateWorkspace(accountID, name string) *domain.Workspace {
	s.mu.Lock()
	defer s.mu.Unlock()
	w := &domain.Workspace{
		WorkspaceID: "ws-" + name,
		AccountID:   accountID,
		Name:        name,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	s.workspaces[w.WorkspaceID] = w
	return w
}

// DocumentRepository
func (s *Store) ListDocuments(wsID string) []*domain.Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []*domain.Document
	for _, d := range s.documents {
		if d.WorkspaceID == wsID {
			res = append(res, d)
		}
	}
	return res
}

func (s *Store) GetDocument(id string) (*domain.Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if d, ok := s.documents[id]; ok {
		return d, true
	}
	return &domain.Document{
		DocumentID:  id,
		WorkspaceID: "mock-ws",
		Filename:    "mock-document.pdf",
	}, true
}

func (s *Store) GetDocumentChunks(documentID string) ([]*domain.DocumentChunk, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	chunks, ok := s.chunks[documentID]
	if !ok {
		return nil, false
	}
	copied := make([]*domain.DocumentChunk, len(chunks))
	copy(copied, chunks)
	return copied, true
}

func (s *Store) GetJobPlanningSignals(documentID, workspaceID, treeID string) (*domain.JobPlanningSignals, bool) {
	return &domain.JobPlanningSignals{DocumentID: documentID, WorkspaceID: workspaceID}, true
}

func (s *Store) CreateDocument(wsID, uploadedBy, filename, mimeType string, fileSize int64) (*domain.Document, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := &domain.Document{
		DocumentID:  "doc-" + filename,
		WorkspaceID: wsID,
		UploadedBy:  uploadedBy,
		Filename:    filename,
		MimeType:    mimeType,
		FileSize:    fileSize,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	s.documents[d.DocumentID] = d
	return d, "http://mock-upload-url/" + d.DocumentID
}

func (s *Store) GetLatestProcessingJob(docID string) (*domain.DocumentProcessingJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, j := range s.jobs {
		if j.DocumentID == docID {
			return j, true
		}
	}
	return nil, false
}

func (s *Store) GetProcessingJob(jobID string) (*domain.DocumentProcessingJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if j, ok := s.jobs[jobID]; ok {
		return j, true
	}
	return &domain.DocumentProcessingJob{
		JobID:       jobID,
		DocumentID:  "mock-doc-" + jobID,
		WorkspaceID: "mock-ws",
		Status:      1, // running
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}, true
}

func (s *Store) GetJobCapability(jobID string) (*domain.JobCapability, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.capabilities[jobID]
	return c, ok
}

func (s *Store) GetJobExecutionPlan(jobID string) (*domain.JobExecutionPlan, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.plans[jobID]
	return p, ok
}

func (s *Store) UpsertJobExecutionPlan(jobID string, plan *domain.JobExecutionPlan) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[jobID] = plan
	return true
}

func (s *Store) UpsertJobEvaluation(jobID string, result *domain.JobEvaluationResult) bool {
	return true
}

func (s *Store) EvaluateJob(jobID string) (*domain.JobEvaluationResult, bool) {
	return &domain.JobEvaluationResult{JobID: jobID, Passed: true, Summary: "mock eval passed"}, true
}

func (s *Store) ListJobApprovalRequests(jobID string) ([]*domain.JobApprovalRequest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.approvals[jobID]
	return a, ok
}

func (s *Store) RequestJobApproval(jobID, requestedBy, reason string) (*domain.JobApprovalRequest, bool) {
	return &domain.JobApprovalRequest{JobID: jobID, Status: "pending"}, true
}

func (s *Store) ApproveJobApproval(jobID, approvalID, reviewedBy string) bool        { return true }
func (s *Store) RejectJobApproval(jobID, approvalID, reviewedBy, reason string) bool { return true }

func (s *Store) CreateProcessingJob(docID, workspaceID string, jobType treev1.JobType) *domain.DocumentProcessingJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	j := &domain.DocumentProcessingJob{
		JobID:       "job-" + docID,
		DocumentID:  docID,
		WorkspaceID: workspaceID,
		JobType:     jobType,
		Status:      treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_QUEUED,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	s.jobs[j.JobID] = j
	return j
}

func (s *Store) MarkProcessingJobRunning(jobID string) bool        { return true }
func (s *Store) UpdateProcessingJobStage(jobID, stage string) bool { return true }
func (s *Store) FailProcessingJob(jobID, errorMessage string) bool { return true }
func (s *Store) CompleteProcessingJob(jobID string) bool           { return true }
func (s *Store) ListAllJobs() ([]*domain.DocumentProcessingJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If no jobs exist, return a set of realistic mock jobs for UI testing
	if len(s.jobs) == 0 {
		now := time.Now().UTC()
		return []*domain.DocumentProcessingJob{
			{
				JobID:      "job-audit-demo-1",
				DocumentID: "annual_report_2024.pdf",
				Status:     treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_SUCCEEDED,
				CreatedAt:  now.Add(-1 * time.Hour).Format(time.RFC3339),
			},
			{
				JobID:      "job-audit-demo-2",
				DocumentID: "technical_spec_v2.docx",
				Status:     treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_RUNNING,
				CreatedAt:  now.Add(-30 * time.Minute).Format(time.RFC3339),
			},
			{
				JobID:      "job-audit-demo-3",
				DocumentID: "contract_legal_v1.pdf",
				Status:     treev1.JobLifecycleState_JOB_LIFECYCLE_STATE_FAILED,
				CreatedAt:  now.Add(-2 * time.Hour).Format(time.RFC3339),
			},
		}, true
	}

	var res []*domain.DocumentProcessingJob
	for _, j := range s.jobs {
		res = append(res, j)
	}
	return res, true
}

func (s *Store) SaveDocumentChunks(documentID string, chunks []*domain.DocumentChunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([]*domain.DocumentChunk, len(chunks))
	copy(copied, chunks)
	s.chunks[documentID] = copied
	return nil
}

func (s *Store) LogToolCall(ctx context.Context, jobID, toolName, inputJSON, outputJSON string, durationMs int64) error {
	return nil
}

func (s *Store) SearchRelatedChunks(ctx context.Context, workspaceID, query string, limit int) ([]*domain.DocumentChunk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 8
	}
	query = strings.ToLower(query)
	var out []*domain.DocumentChunk
	for documentID, chunks := range s.chunks {
		doc, ok := s.documents[documentID]
		if !ok || doc.WorkspaceID != workspaceID {
			continue
		}
		for _, chunk := range chunks {
			if query == "" || strings.Contains(strings.ToLower(chunk.Heading+" "+chunk.Text), query) {
				out = append(out, chunk)
				if len(out) >= limit {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

func (s *Store) ListJobMutationLogs(jobID string) ([]*domain.JobMutationLog, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return real mock data for any requested jobId to facilitate UI testing
	now := time.Now().UTC()
	return []*domain.JobMutationLog{
		{
			MutationID:   "m1",
			JobID:        jobID,
			TargetType:   "agent",
			TargetID:     "Orchestrator",
			MutationType: "start",
			CreatedAt:    now.Add(-10 * time.Minute).Format(time.RFC3339),
		},
		{
			MutationID:     "m2",
			JobID:          jobID,
			TargetType:     "tool_call",
			TargetID:       "manage_job_checklist",
			MutationType:   "execute",
			BeforeJSON:     `{"action": "add", "description": "Analyze document structure"}`,
			AfterJSON:      `{"status": "ok", "task_id": "T1"}`,
			ProvenanceJSON: `{"duration_ms": 150}`,
			CreatedAt:      now.Add(-9 * time.Minute).Format(time.RFC3339),
		},
		{
			MutationID:     "m3",
			JobID:          jobID,
			TargetType:     "tool_call",
			TargetID:       "extract_text",
			MutationType:   "execute",
			BeforeJSON:     `{"file_uri": "gs://bucket/doc.pdf"}`,
			AfterJSON:      `{"raw_text": "Extracted content..."}`,
			ProvenanceJSON: `{"duration_ms": 1200}`,
			CreatedAt:      now.Add(-8 * time.Minute).Format(time.RFC3339),
		},
		{
			MutationID:     "m4",
			JobID:          jobID,
			TargetType:     "tool_call",
			TargetID:       "semantic_chunking",
			MutationType:   "execute",
			BeforeJSON:     `{"raw_text": "..."}`,
			AfterJSON:      `{"chunks": [{"index":0, "text": "..."}]}`,
			ProvenanceJSON: `{"duration_ms": 2500}`,
			CreatedAt:      now.Add(-7 * time.Minute).Format(time.RFC3339),
		},
		{
			MutationID:     "m5",
			JobID:          jobID,
			TargetType:     "tool_call",
			TargetID:       "goal_driven_synthesis",
			MutationType:   "execute",
			BeforeJSON:     `{"document_brief": "Master blueprint..."}`,
			AfterJSON:      `{"items": [{"label": "Concept A", "level": 0}]}`,
			ProvenanceJSON: `{"duration_ms": 5000}`,
			CreatedAt:      now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
		{
			MutationID:     "m6",
			JobID:          jobID,
			TargetType:     "tool_call",
			TargetID:       "quality_critique",
			MutationType:   "execute",
			BeforeJSON:     `{"target_data": "..."}`,
			AfterJSON:      `{"valid": true, "issues": []}`,
			ProvenanceJSON: `{"duration_ms": 3200}`,
			CreatedAt:      now.Add(-2 * time.Minute).Format(time.RFC3339),
		},
		{
			MutationID:     "m7",
			JobID:          jobID,
			TargetType:     "tool_call",
			TargetID:       "persist_knowledge_tree",
			MutationType:   "execute",
			BeforeJSON:     `{"items": [...]}`,
			AfterJSON:      `{"success": true}`,
			ProvenanceJSON: `{"duration_ms": 450}`,
			CreatedAt:      now.Add(-1 * time.Minute).Format(time.RFC3339),
		},
	}, true
}

// TreeRepository
func (s *Store) GetOrCreateTree(wsID string) (*domain.Tree, error) {
	return &domain.Tree{TreeID: wsID, WorkspaceID: wsID, Name: "default"}, nil
}

func (s *Store) GetTreeByWorkspace(wsID string) ([]*domain.Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wsItems, ok := s.items[wsID]
	if !ok {
		return nil, false
	}
	var res []*domain.Item
	for _, n := range wsItems {
		res = append(res, n)
	}
	return res, true
}

func (s *Store) GetWorkspaceRootItemID(wsID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wsItems, ok := s.items[wsID]
	if !ok {
		return "", false
	}
	for _, n := range wsItems {
		if n.ParentID == "" {
			return n.ItemID, true
		}
	}
	return "", false
}

func (s *Store) FindPaths(wsID, sourceItemID, targetItemID string, maxDepth, limit int) ([]*domain.Item, []domain.TreePath, bool) {
	items, ok := s.GetTreeByWorkspace(wsID)
	if !ok {
		return nil, nil, false
	}
	return items, nil, true
}

func (s *Store) GetSubtree(rootItemID string, maxDepth int) ([]*domain.SubtreeItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var res []*domain.SubtreeItem
	var root *domain.Item
	var workspaceID string

	// Find the root item and its workspace
	for wsID, wsItems := range s.items {
		if n, ok := wsItems[rootItemID]; ok {
			root = n
			workspaceID = wsID
			break
		}
	}

	if root == nil {
		return nil, fmt.Errorf("root item not found")
	}

	res = append(res, &domain.SubtreeItem{Item: *root})

	// Find direct children
	for _, n := range s.items[workspaceID] {
		if n.ParentID == rootItemID {
			res = append(res, &domain.SubtreeItem{Item: *n})
		}
	}

	return res, nil
}

// ItemRepository
func (s *Store) GetItem(itemID string) (*domain.Item, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, wsItems := range s.items {
		if n, ok := wsItems[itemID]; ok {
			return n, true
		}
	}
	return nil, false
}

func (s *Store) CreateItem(workspaceID, label, description, parentID, createdBy string) *domain.Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[workspaceID]; !ok {
		s.items[workspaceID] = make(map[string]*domain.Item)
	}
	n := &domain.Item{
		ItemID:      "item-" + label,
		WorkspaceID: workspaceID,
		ParentID:    parentID,
		Label:       label,
		Description: description,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	s.items[workspaceID][n.ItemID] = n
	return n
}

func (s *Store) CreateStructuredItemWithCapability(capability *domain.JobCapability, jobID, documentID, workspaceID, label string, level int, description, summaryHTML, createdBy, parentID string, sourceChunkIDs []string) *domain.Item {
	return s.CreateItem(workspaceID, label, description, parentID, createdBy)
}

func (s *Store) UpsertItemSource(itemID, documentID, chunkID, sourceText string, confidence float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sources[itemID] = append(s.sources[itemID], &domain.ItemSource{
		ItemID:     itemID,
		DocumentID: documentID,
		ChunkID:    chunkID,
		SourceText: sourceText,
		Confidence: confidence,
	})
	return nil
}

func (s *Store) UpdateItemSummaryHTMLWithCapability(capability *domain.JobCapability, jobID, itemID, summaryHTML string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, wsItems := range s.items {
		if n, ok := wsItems[itemID]; ok {
			n.SummaryHTML = summaryHTML
			return true
		}
	}
	return false
}

func (s *Store) ApproveAlias(wsID, canonicalItemID, aliasItemID string) bool { return true }
func (s *Store) RejectAlias(wsID, canonicalItemID, aliasItemID string) bool  { return true }

var _ repository.AccountRepository = (*Store)(nil)
var _ repository.WorkspaceRepository = (*Store)(nil)
var _ repository.DocumentRepository = (*Store)(nil)
var _ repository.TreeRepository = (*Store)(nil)
var _ repository.ItemRepository = (*Store)(nil)
