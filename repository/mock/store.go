package mock

import (
	"fmt"
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
	d, ok := s.documents[id]
	return d, ok
}

func (s *Store) GetDocumentChunks(documentID string) ([]*domain.DocumentChunk, bool) {
	return nil, false
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
	j, ok := s.jobs[jobID]
	return j, ok
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
func (s *Store) SaveDocumentChunks(documentID string, chunks []*domain.DocumentChunk) error {
	return nil
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
