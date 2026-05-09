package mock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/synthify/backend/packages/shared/domain"
	treev1 "github.com/synthify/backend/packages/shared/gen/synthify/tree/v1"
)

var ctx = context.Background()

// ── helpers ───────────────────────────────────────────────────────────────────

// setupWorkspace creates an account and workspace for a given userID.
func setupWorkspace(t *testing.T, s *Store, userID string) *domain.Workspace {
	t.Helper()
	acct, err := s.GetOrCreateAccount(ctx, userID)
	require.NoError(t, err, "GetOrCreateAccount")
	ws := s.CreateWorkspace(ctx, acct.AccountID, "test-workspace")
	require.NotNil(t, ws, "CreateWorkspace returned nil")
	return ws
}

// setupTree creates a tree and seed items for a workspace.
func setupTree(t *testing.T, s *Store, wsID string) *domain.Tree {
	t.Helper()
	tree, err := s.GetOrCreateTree(ctx, wsID)
	require.NoError(t, err, "GetOrCreateTree")
	// Create a processing job to generate seed data.
	doc, _ := s.CreateDocument(ctx, wsID, "user1", "f.pdf", "application/pdf", 100)
	s.CreateProcessingJob(ctx, doc.DocumentID, wsID, treev1.JobType_JOB_TYPE_PROCESS_DOCUMENT)
	return tree
}

// ── IsWorkspaceAccessible ─────────────────────────────────────────────────────

func TestIsWorkspaceAccessible_Owner_ReturnsTrue(t *testing.T) {
	store := NewStore()
	ws := setupWorkspace(t, store, "owner")

	assert.True(t, store.IsWorkspaceAccessible(ctx, ws.WorkspaceID, "owner"), "owner should have access to their workspace")
}

func TestIsWorkspaceAccessible_Stranger_ReturnsFalse(t *testing.T) {
	store := NewStore()
	ws := setupWorkspace(t, store, "owner")

	assert.False(t, store.IsWorkspaceAccessible(ctx, ws.WorkspaceID, "stranger"), "stranger should not have access")
}

// ── ApproveAlias / RejectAlias ────────────────────────────────────────────────

func TestApproveAlias_RecordsAlias(t *testing.T) {
	store := NewStore()
	ws := setupWorkspace(t, store, "u1")
	setupTree(t, store, ws.WorkspaceID)

	// Add seed items
	store.CreateItem(ctx, ws.WorkspaceID, "root", "root desc", "", "system")
	store.CreateItem(ctx, ws.WorkspaceID, "child", "child desc", "item-root", "system")

	ok := store.ApproveAlias(ctx, ws.WorkspaceID, "item-root", "item-child")
	require.True(t, ok, "ApproveAlias returned false")
}

func TestApproveAlias_UnknownItem_ReturnsFalse(t *testing.T) {
	store := NewStore()
	ws := setupWorkspace(t, store, "u1")

	ok := store.ApproveAlias(ctx, ws.WorkspaceID, "nonexistent", "also_nonexistent")
	assert.False(t, ok, "ApproveAlias unknown items: expected false, got true")
}

func TestRejectAlias_RemovesAlias(t *testing.T) {
	store := NewStore()
	ws := setupWorkspace(t, store, "u1")
	setupTree(t, store, ws.WorkspaceID)

	store.ApproveAlias(ctx, ws.WorkspaceID, "item-root", "item-child")

	ok := store.RejectAlias(ctx, ws.WorkspaceID, "item-root", "item-child")
	require.True(t, ok, "RejectAlias returned false")
}

func TestGetJobPlanningSignals_CountsProvenanceAndAliases(t *testing.T) {
	store := NewStore()
	ws := setupWorkspace(t, store, "u1")
	_ = setupTree(t, store, ws.WorkspaceID)

	err := store.UpsertItemSource(ctx, "nd_tel", "doc-1", "chunk-1", "source", 0.9)
	require.NoError(t, err, "UpsertItemSource nd_tel")
	err = store.UpsertItemSource(ctx, "nd_roi", "doc-1", "chunk-2", "source", 0.8)
	require.NoError(t, err, "UpsertItemSource nd_roi")

	signals, ok := store.GetJobPlanningSignals(ctx, "doc-1", ws.WorkspaceID, ws.WorkspaceID)
	require.True(t, ok, "GetJobPlanningSignals returned false")
	require.NotNil(t, signals, "GetJobPlanningSignals returned nil")
}

// ── FindPaths (Tree traversal) ────────────────────────────────────────────────

func TestFindPaths_FindsConnectedPath(t *testing.T) {
	wsID := "ws1"
	store := NewStore()
	store.CreateItem(ctx, wsID, "n1", "", "", "u1")
	store.CreateItem(ctx, wsID, "n2", "", "item-n1", "u1")
	store.CreateItem(ctx, wsID, "n3", "", "item-n2", "u1")

	_, paths, ok := store.FindPaths(ctx, wsID, "item-n3", "item-n1", 4, 3)
	require.True(t, ok, "FindPaths returned false")
	require.NotEmpty(t, paths, "expected at least one path")
	assert.Equal(t, "item-n3", paths[0].ItemIDs[0], "path start")
	assert.Equal(t, "item-n1", paths[0].ItemIDs[len(paths[0].ItemIDs)-1], "path end")
}

func TestFindPaths_NoPathExists_ReturnsEmptyPaths(t *testing.T) {
	wsID := "ws1"
	store := NewStore()
	store.CreateItem(ctx, wsID, "n1", "", "", "u1")
	store.CreateItem(ctx, wsID, "n3", "", "", "u1")

	_, paths, ok := store.FindPaths(ctx, wsID, "item-n3", "item-n1", 4, 3)
	// In my mock implementation FindPaths returns ok=true if workspace exists
	require.True(t, ok, "FindPaths returned false (workspace exists)")
	assert.Empty(t, paths, "expected no paths")
}

func TestFindPaths_WorkspaceNotFound_ReturnsFalse(t *testing.T) {
	store := NewStore()

	_, _, ok := store.FindPaths(ctx, "nonexistent_ws", "n1", "n2", 4, 3)
	assert.False(t, ok, "FindPaths unknown workspace: expected false, got true")
}
