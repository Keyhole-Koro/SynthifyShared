package domain

import "time"

// DocumentLifecycleState represents the document processing lifecycle state.
type DocumentLifecycleState string

const (
	DocumentLifecycleUploaded             DocumentLifecycleState = "uploaded"
	DocumentLifecyclePendingNormalization DocumentLifecycleState = "pending_normalization"
	DocumentLifecycleProcessing           DocumentLifecycleState = "processing"
	DocumentLifecycleCompleted            DocumentLifecycleState = "completed"
	DocumentLifecycleFailed               DocumentLifecycleState = "failed"
)

type Account struct {
	AccountID          string `json:"account_id"`
	Name               string `json:"name"`
	Plan               string `json:"plan"` // "anonymous" | "registered" | "pro"
	StorageQuotaBytes  int64  `json:"storage_quota_bytes"`
	StorageUsedBytes   int64  `json:"storage_used_bytes"`
	MaxFileSizeBytes   int64  `json:"max_file_size_bytes"`
	MaxUploadsPerFiveH int64  `json:"max_uploads_per_5h"`
	MaxUploadsPerWeek  int64  `json:"max_uploads_per_1week"`
	CreatedAt          string `json:"created_at"`
}

type AccountUser struct {
	AccountID string `json:"account_id"`
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	JoinedAt  string `json:"joined_at"`
}

type Workspace struct {
	WorkspaceID string `json:"workspace_id"`
	AccountID   string `json:"account_id"`
	Name        string `json:"name"`
	RootNodeID  string `json:"root_node_id,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type Document struct {
	DocumentID  string `json:"document_id"`
	WorkspaceID string `json:"workspace_id"`
	UploadedBy  string `json:"uploaded_by"`
	Filename    string `json:"filename"`
	MimeType    string `json:"mime_type"`
	FileSize    int64  `json:"file_size"`
	CreatedAt   string `json:"created_at"`
}

type DocumentProcessingJob struct {
	JobID            string `json:"job_id"`
	DocumentID       string `json:"document_id"`
	GraphID          string `json:"graph_id,omitempty"`
	JobType          string `json:"job_type"`
	Status           string `json:"status"`
	CurrentStage     string `json:"current_stage,omitempty"`
	ErrorMessage     string `json:"error_message,omitempty"`
	ParamsJSON       string `json:"params_json,omitempty"`
	RequestedBy      string `json:"requested_by,omitempty"`
	CapabilityID     string `json:"capability_id,omitempty"`
	ExecutionPlanID  string `json:"execution_plan_id,omitempty"`
	PlanStatus       string `json:"plan_status,omitempty"`
	EvaluationStatus string `json:"evaluation_status,omitempty"`
	RetryCount       int    `json:"retry_count,omitempty"`
	BudgetJSON       string `json:"budget_json,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type Graph struct {
	GraphID     string `json:"graph_id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Node struct {
	NodeID            string `json:"node_id"`
	GraphID           string `json:"graph_id"`
	Label             string `json:"label"`
	Level             int    `json:"level,omitempty"`
	Description       string `json:"description"`
	SummaryHTML       string `json:"summary_html,omitempty"`
	CreatedBy         string `json:"created_by,omitempty"`
	GovernanceState   string `json:"governance_state,omitempty"`
	LockedBy          string `json:"locked_by,omitempty"`
	LockedAt          string `json:"locked_at,omitempty"`
	LastMutationJobID string `json:"last_mutation_job_id,omitempty"`
	CreatedAt         string `json:"created_at"`
}

type Edge struct {
	EdgeID       string `json:"edge_id"`
	GraphID      string `json:"graph_id"`
	SourceNodeID string `json:"source_node_id"`
	TargetNodeID string `json:"target_node_id"`
	EdgeType     string `json:"edge_type"`
	Description  string `json:"description,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

type NodeSource struct {
	NodeID     string  `json:"node_id"`
	DocumentID string  `json:"document_id"`
	ChunkID    string  `json:"chunk_id,omitempty"`
	SourceText string  `json:"source_text,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

type NodeEvidence struct {
	Sources []*NodeSource `json:"sources,omitempty"`
}

type EdgeSource struct {
	EdgeID     string  `json:"edge_id"`
	DocumentID string  `json:"document_id"`
	ChunkID    string  `json:"chunk_id,omitempty"`
	SourceText string  `json:"source_text,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

// GraphNode is the node representation returned by the API.
type GraphNode struct {
	ID          string `json:"id"`
	Scope       string `json:"scope"` // "document" | "canonical"
	Label       string `json:"label"`
	Description string `json:"description"`
	SummaryHTML string `json:"summary_html,omitempty"`
}

// GraphEdge is the edge representation returned by the API.
type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
	Scope  string `json:"scope"` // "document" | "canonical"
}

type DocumentChunk struct {
	ChunkID    string `json:"chunk_id"`
	DocumentID string `json:"document_id"`
	Heading    string `json:"heading"`
	Text       string `json:"text"`
	SourcePage int    `json:"source_page,omitempty"`
}

type Job struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type JobOperation string

const (
	JobOperationReadGraph                  JobOperation = "read_graph"
	JobOperationReadDocument               JobOperation = "read_document"
	JobOperationCreateNode                 JobOperation = "create_node"
	JobOperationUpdateNode                 JobOperation = "update_node"
	JobOperationCreateEdge                 JobOperation = "create_edge"
	JobOperationDeleteEdge                 JobOperation = "delete_edge"
	JobOperationRunNormalizationToolDryRun JobOperation = "run_normalization_tool_dry_run"
	JobOperationRunNormalizationToolApply  JobOperation = "run_normalization_tool_apply"
	JobOperationInvokeLLM                  JobOperation = "invoke_llm"
	JobOperationEmitPlan                   JobOperation = "emit_plan"
	JobOperationEmitEval                   JobOperation = "emit_eval"
)

type NodeGovernanceState string

const (
	NodeGovernanceStateSystemGenerated NodeGovernanceState = "system_generated"
	NodeGovernanceStatePendingReview   NodeGovernanceState = "pending_review"
	NodeGovernanceStateHumanCurated    NodeGovernanceState = "human_curated"
	NodeGovernanceStateLocked          NodeGovernanceState = "locked"
)

type JobCapability struct {
	CapabilityID       string         `json:"capability_id"`
	JobID              string         `json:"job_id"`
	WorkspaceID        string         `json:"workspace_id"`
	GraphID            string         `json:"graph_id"`
	AllowedDocumentIDs []string       `json:"allowed_document_ids,omitempty"`
	AllowedNodeIDs     []string       `json:"allowed_node_ids,omitempty"`
	AllowedOperations  []JobOperation `json:"allowed_operations,omitempty"`
	MaxLLMCalls        int            `json:"max_llm_calls,omitempty"`
	MaxToolRuns        int            `json:"max_tool_runs,omitempty"`
	MaxNodeCreations   int            `json:"max_node_creations,omitempty"`
	MaxEdgeMutations   int            `json:"max_edge_mutations,omitempty"`
	ExpiresAt          string         `json:"expires_at,omitempty"`
	CreatedAt          string         `json:"created_at,omitempty"`
}

func (c *JobCapability) Allows(op JobOperation) bool {
	if c == nil {
		return false
	}
	for _, allowed := range c.AllowedOperations {
		if allowed == op {
			return true
		}
	}
	return false
}

func (c *JobCapability) AllowsDocument(documentID string) bool {
	if c == nil || documentID == "" {
		return false
	}
	if len(c.AllowedDocumentIDs) == 0 {
		return true
	}
	for _, allowed := range c.AllowedDocumentIDs {
		if allowed == documentID {
			return true
		}
	}
	return false
}

func (c *JobCapability) AllowsNode(nodeID string) bool {
	if c == nil || nodeID == "" {
		return false
	}
	if len(c.AllowedNodeIDs) == 0 {
		return true
	}
	for _, allowed := range c.AllowedNodeIDs {
		if allowed == nodeID {
			return true
		}
	}
	return false
}

func (c *JobCapability) IsExpired(now time.Time) bool {
	if c == nil || c.ExpiresAt == "" {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, c.ExpiresAt)
	if err != nil {
		return true
	}
	return now.After(expiresAt)
}

type JobMutationLog struct {
	MutationID     string `json:"mutation_id"`
	JobID          string `json:"job_id"`
	PlanID         string `json:"plan_id,omitempty"`
	CapabilityID   string `json:"capability_id,omitempty"`
	GraphID        string `json:"graph_id"`
	TargetType     string `json:"target_type"`
	TargetID       string `json:"target_id"`
	MutationType   string `json:"mutation_type"`
	RiskTier       string `json:"risk_tier,omitempty"`
	BeforeJSON     string `json:"before_json,omitempty"`
	AfterJSON      string `json:"after_json,omitempty"`
	ProvenanceJSON string `json:"provenance_json,omitempty"`
	CreatedAt      string `json:"created_at"`
}

func DefaultJobCapability(jobID, workspaceID, graphID, documentID string, createdAt time.Time) *JobCapability {
	return &JobCapability{
		CapabilityID:       "cap_" + jobID,
		JobID:              jobID,
		WorkspaceID:        workspaceID,
		GraphID:            graphID,
		AllowedDocumentIDs: []string{documentID},
		AllowedOperations: []JobOperation{
			JobOperationReadGraph,
			JobOperationReadDocument,
			JobOperationCreateNode,
			JobOperationUpdateNode,
			JobOperationCreateEdge,
			JobOperationInvokeLLM,
		},
		MaxLLMCalls:      128,
		MaxToolRuns:      0,
		MaxNodeCreations: 4096,
		MaxEdgeMutations: 4096,
		ExpiresAt:        createdAt.Add(24 * time.Hour).UTC().Format(time.RFC3339),
		CreatedAt:        createdAt.UTC().Format(time.RFC3339),
	}
}

type SubtreeNode struct {
	Node
	HasChildren bool `json:"has_children"`
}

type GraphPath struct {
	NodeIDs  []string `json:"node_ids"`
	HopCount int      `json:"hop_count"`
	Evidence struct {
		SourceDocumentIDs []string `json:"source_document_ids"`
	} `json:"evidence_ref"`
}
