package postgres

import (
	"context"
	"database/sql"

	"github.com/Keyhole-Koro/SynthifyShared/domain"
)

func (s *Store) GetOrCreateTree(wsID string) (*domain.Tree, error) {
	root, err := s.q().GetTreeRoot(context.Background(), wsID)
	if err != nil {
		return nil, err
	}
	return &domain.Tree{
		TreeID:      wsID, // ワークスペースIDをツリーIDとして扱う
		WorkspaceID: wsID,
		Name:        "default",
		CreatedAt:   root.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   root.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}, nil
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
			Item: domain.Item{
				ItemID:      r.ID,
				WorkspaceID: r.WorkspaceID,
				ParentID:    r.ParentID.String,
				Label:       r.Label,
				Level:       int(r.Level),
				Description: r.Description,
				SummaryHTML: r.SummaryHtml,
				CreatedBy:   r.CreatedBy,
				CreatedAt:   r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			},
			HasChildren: r.HasChildren,
		})
	}
	return items, nil
}

func (s *Store) listItemsByTree(wsID string) ([]*domain.Item, error) {
	rows, err := s.q().ListItemsByWorkspace(context.Background(), wsID)
	if err != nil {
		return nil, err
	}
	var items []*domain.Item
	for _, row := range rows {
		items = append(items, toItemFromItemRow(row))
	}
	return items, nil
}
