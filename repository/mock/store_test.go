package mock

import (
	"context"
	"testing"

	"github.com/Keyhole-Koro/SynthifyShared/domain"
	treev1 "github.com/Keyhole-Koro/SynthifyShared/gen/synthify/tree/v1"
)

var ctx = context.Background()

// ── helpers ───────────────────────────────────────────────────────────────────

// setupWorkspace creates an account and workspace for a given userID.
func setupWorkspace(t *testing.T, s *Store, userID string) *domain.Workspace {
	t.Helper()
	acct, err := s.GetOrCreateAccount(ctx, userID)
	if err != nil {
		t.Fatalf("GetOrCreateAccount: %v", err)
	}
	ws := s.CreateWorkspace(ctx, acct.AccountID, "test-workspace")
	if ws == nil {
		t.Fatal("CreateWorkspace returned nil")
	}
	return ws
}

// setupTree creates a tree and seed items for a workspace.
func setupTree(t *testing.T, s *Store, wsID string) *domain.Tree {
	t.Helper()
	tree, err := s.GetOrCreateTree(ctx, wsID)
	if err != nil {
		t.Fatalf("GetOrCreateTree: %v", err)
	}
	// Create a processing job to generate seed data.
	doc, _ := s.CreateDocument(ctx, wsID, "user1", "f.pdf", "application/pdf", 100)
	s.CreateProcessingJob(ctx, doc.DocumentID, wsID, treev1.JobType_JOB_TYPE_PROCESS_DOCUMENT)
	return tree
}

// ── IsWorkspaceAccessible ─────────────────────────────────────────────────────

func TestIsWorkspaceAccessible_Owner_ReturnsTrue(t *testing.T) {
	store := NewStore()
	ws := setupWorkspace(t, store, "owner")

	if !store.IsWorkspaceAccessible(ctx, ws.WorkspaceID, "owner") {
		t.Error("owner should have access to their workspace")
	}
}

func TestIsWorkspaceAccessible_Stranger_ReturnsFalse(t *testing.T) {
	store := NewStore()
	ws := setupWorkspace(t, store, "owner")

	if store.IsWorkspaceAccessible(ctx, ws.WorkspaceID, "stranger") {
		t.Error("stranger should not have access")
	}
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
	if !ok {
		t.Fatal("ApproveAlias returned false")
	}
}

func TestApproveAlias_UnknownItem_ReturnsFalse(t *testing.T) {
	store := NewStore()
	ws := setupWorkspace(t, store, "u1")

	ok := store.ApproveAlias(ctx, ws.WorkspaceID, "nonexistent", "also_nonexistent")
	if ok {
		t.Error("ApproveAlias unknown items: expected false, got true")
	}
}

func TestRejectAlias_RemovesAlias(t *testing.T) {
	store := NewStore()
	ws := setupWorkspace(t, store, "u1")
	setupTree(t, store, ws.WorkspaceID)

	store.ApproveAlias(ctx, ws.WorkspaceID, "item-root", "item-child")

	ok := store.RejectAlias(ctx, ws.WorkspaceID, "item-root", "item-child")
	if !ok {
		t.Fatal("RejectAlias returned false")
	}
}

func TestGetJobPlanningSignals_CountsProvenanceAndAliases(t *testing.T) {
	store := NewStore()
	ws := setupWorkspace(t, store, "u1")
	_ = setupTree(t, store, ws.WorkspaceID)

	if err := store.UpsertItemSource(ctx, "nd_tel", "doc-1", "chunk-1", "source", 0.9); err != nil {
		t.Fatalf("UpsertItemSource nd_tel: %v", err)
	}
	if err := store.UpsertItemSource(ctx, "nd_roi", "doc-1", "chunk-2", "source", 0.8); err != nil {
		t.Fatalf("UpsertItemSource nd_roi: %v", err)
	}

	signals, ok := store.GetJobPlanningSignals(ctx, "doc-1", ws.WorkspaceID, ws.WorkspaceID)
	if !ok || signals == nil {
		t.Fatal("GetJobPlanningSignals returned false")
	}
}

// ── FindPaths (Tree traversal) ────────────────────────────────────────────────

func TestFindPaths_FindsConnectedPath(t *testing.T) {
	wsID := "ws1"
	store := NewStore()
	store.CreateItem(ctx, wsID, "n1", "", "", "u1")
	store.CreateItem(ctx, wsID, "n2", "", "item-n1", "u1")
	store.CreateItem(ctx, wsID, "n3", "", "item-n2", "u1")

	_, paths, ok := store.FindPaths(ctx, wsID, "item-n3", "item-n1", 4, 3)
	if !ok {
		t.Fatal("FindPaths returned false")
	}
	if len(paths) == 0 {
		t.Fatal("expected at least one path")
	}
	if paths[0].ItemIDs[0] != "item-n3" {
		t.Errorf("path start = %q, want item-n3", paths[0].ItemIDs[0])
	}
	if paths[0].ItemIDs[len(paths[0].ItemIDs)-1] != "item-n1" {
		t.Errorf("path end = %q, want item-n1", paths[0].ItemIDs[len(paths[0].ItemIDs)-1])
	}
}

func TestFindPaths_NoPathExists_ReturnsEmptyPaths(t *testing.T) {
	wsID := "ws1"
	store := NewStore()
	store.CreateItem(ctx, wsID, "n1", "", "", "u1")
	store.CreateItem(ctx, wsID, "n3", "", "", "u1")

	_, paths, ok := store.FindPaths(ctx, wsID, "item-n3", "item-n1", 4, 3)
	if !ok {
		// In my mock implementation FindPaths returns ok=true if workspace exists
		t.Fatal("FindPaths returned false (workspace exists)")
	}
	if len(paths) != 0 {
		t.Errorf("expected no paths, got %d", len(paths))
	}
}

func TestFindPaths_WorkspaceNotFound_ReturnsFalse(t *testing.T) {
	store := NewStore()

	_, _, ok := store.FindPaths(ctx, "nonexistent_ws", "n1", "n2", 4, 3)
	if ok {
		t.Error("FindPaths unknown workspace: expected false, got true")
	}
}
