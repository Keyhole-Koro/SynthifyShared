package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Keyhole-Koro/SynthifyShared/domain"
	"github.com/Keyhole-Koro/SynthifyShared/repository/postgres/sqlcgen"
)

func (s *Store) GetOrCreateTree(wsID string) (*domain.Tree, error) {
	ctx := context.Background()
	// 1 ワークスペース = 1 ツリー。ルートアイテムがあればそれを返す。
	root, err := s.q().GetTreeRoot(ctx, wsID)
	if err == nil {
		return &domain.Tree{
			TreeID:      wsID,
			WorkspaceID: wsID,
			Name:        "default",
			CreatedAt:   root.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:   root.CreatedAt.UTC().Format(time.RFC3339),
		}, nil
	}

	// ルートがない場合は作成
	ws, err := s.q().GetWorkspace(ctx, wsID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	err = s.q().CreateItem(ctx, sqlcgen.CreateItemParams{
		ID:          newID(),
		WorkspaceID: wsID,
		ParentID:    sql.NullString{Valid: false},
		Label:       ws.Name,
		Level:       0,
		Description: "Workspace root",
		SummaryHtml: "",
		CreatedBy:   "system",
		CreatedAt:   nowTime(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create root item: %w", err)
	}

	return s.GetOrCreateTree(wsID)
}

func (s *Store) GetTreeByWorkspace(wsID string) ([]*domain.Item, bool) {
	ctx := context.Background()
	rows, err := s.q().ListItemsByWorkspace(ctx, wsID)
	if err != nil {
		return nil, false
	}
	var items []*domain.Item
	for _, r := range rows {
		items = append(items, toItemFromItemRow(r))
	}
	return items, true
}

func (s *Store) GetWorkspaceRootItemID(wsID string) (string, bool) {
	root, err := s.q().GetTreeRoot(context.Background(), wsID)
	if err != nil {
		return "", false
	}
	return root.ID, true
}

func (s *Store) FindPaths(wsID, sourceItemID, targetItemID string, maxDepth, limit int) ([]*domain.Item, []domain.TreePath, bool) {
	items, ok := s.GetTreeByWorkspace(wsID)
	if !ok || len(items) == 0 {
		return nil, nil, false
	}

	itemByID := make(map[string]*domain.Item, len(items))
	for _, item := range items {
		itemByID[item.ItemID] = item
	}

	if itemByID[sourceItemID] == nil || itemByID[targetItemID] == nil {
		return nil, nil, false
	}

	var paths []domain.TreePath
	curr := sourceItemID
	var sourcePath []string
	for curr != "" && itemByID[curr] != nil {
		sourcePath = append(sourcePath, curr)
		if curr == targetItemID {
			paths = append(paths, domain.TreePath{ItemIDs: sourcePath, HopCount: len(sourcePath) - 1})
			return items, paths, true
		}
		curr = itemByID[curr].ParentID
	}

	return items, paths, len(paths) > 0
}

func (s *Store) GetSubtree(rootItemID string, maxDepth int) ([]*domain.SubtreeItem, error) {
	ctx := context.Background()

	// ルートアイテム取得
	rootRow, err := s.q().GetItem(ctx, rootItemID)
	if err != nil {
		return nil, err
	}

	// 子要素取得
	rows, err := s.q().ListChildItems(ctx, sql.NullString{String: rootItemID, Valid: true})
	if err != nil {
		return nil, err
	}

	var items []*domain.SubtreeItem
	items = append(items, &domain.SubtreeItem{
		Item: *toItemFromGetRow(rootRow),
	})

	for _, r := range rows {
		items = append(items, &domain.SubtreeItem{
			Item:        *toItemFromChildRow(r),
			HasChildren: r.HasChildren,
		})
	}
	return items, nil
}
