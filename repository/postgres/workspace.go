package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Keyhole-Koro/SynthifyShared/domain"
	"github.com/Keyhole-Koro/SynthifyShared/repository/postgres/sqlcgen"
)

// defaultRegisteredPlan defines the default plan limits for newly created users.
var defaultRegisteredPlan = struct {
	StorageQuotaBytes int64
	MaxFileSizeBytes  int64
	MaxUploadsPer5h   int64
	MaxUploadsPerWeek int64
}{
	StorageQuotaBytes: 5 * 1 << 30, // 5GB
	MaxFileSizeBytes:  100 << 20,   // 100MB
	MaxUploadsPer5h:   20,
	MaxUploadsPerWeek: 100,
}

func (s *Store) GetOrCreateAccount(ctx context.Context, userID string) (*domain.Account, error) {
	// Return the existing account if present.
	existing, err := s.q().GetAccountByUser(ctx, userID)
	if err == nil {
		acct := toAccount(sqlcgen.Account{
			AccountID:          existing.AccountID,
			Name:               existing.Name,
			Plan:               existing.Plan,
			StorageQuotaBytes:  existing.StorageQuotaBytes,
			StorageUsedBytes:   existing.StorageUsedBytes,
			MaxFileSizeBytes:   existing.MaxFileSizeBytes,
			MaxUploadsPer5h:    existing.MaxUploadsPer5h,
			MaxUploadsPer1week: existing.MaxUploadsPer1week,
			CreatedAt:          existing.CreatedAt,
		})
		return acct, nil
	}

	// Otherwise create a new account.
	accountID := newID()
	createdAt := nowTime()
	row, err := s.q().GetOrCreateAccount(ctx, sqlcgen.GetOrCreateAccountParams{
		AccountID:          accountID,
		Name:               fmt.Sprintf("account-%s", userID),
		Plan:               "registered",
		StorageQuotaBytes:  defaultRegisteredPlan.StorageQuotaBytes,
		MaxFileSizeBytes:   defaultRegisteredPlan.MaxFileSizeBytes,
		MaxUploadsPer5h:    int32(defaultRegisteredPlan.MaxUploadsPer5h),
		MaxUploadsPer1week: int32(defaultRegisteredPlan.MaxUploadsPerWeek),
		CreatedAt:          createdAt,
	})
	if err != nil {
		return nil, err
	}

	_ = s.q().CreateAccountUser(ctx, sqlcgen.CreateAccountUserParams{
		AccountID: row.AccountID,
		UserID:    userID,
		Role:      "owner",
		JoinedAt:  createdAt,
	})

	return toAccount(sqlcgen.Account{
		AccountID:          row.AccountID,
		Name:               row.Name,
		Plan:               row.Plan,
		StorageQuotaBytes:  row.StorageQuotaBytes,
		StorageUsedBytes:   row.StorageUsedBytes,
		MaxFileSizeBytes:   row.MaxFileSizeBytes,
		MaxUploadsPer5h:    row.MaxUploadsPer5h,
		MaxUploadsPer1week: row.MaxUploadsPer1week,
		CreatedAt:          row.CreatedAt,
	}), nil
}

func (s *Store) GetAccount(ctx context.Context, accountID string) (*domain.Account, error) {
	row, err := s.q().GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return toAccount(sqlcgen.Account{
		AccountID:          row.AccountID,
		Name:               row.Name,
		Plan:               row.Plan,
		StorageQuotaBytes:  row.StorageQuotaBytes,
		StorageUsedBytes:   row.StorageUsedBytes,
		MaxFileSizeBytes:   row.MaxFileSizeBytes,
		MaxUploadsPer5h:    row.MaxUploadsPer5h,
		MaxUploadsPer1week: row.MaxUploadsPer1week,
		CreatedAt:          row.CreatedAt,
	}), nil
}

func (s *Store) ListWorkspacesByUser(ctx context.Context, userID string) []*domain.Workspace {
	rows, err := s.q().ListWorkspacesByUser(ctx, userID)
	if err != nil {
		return nil
	}
	var workspaces []*domain.Workspace
	for _, row := range rows {
		rootItemID, _ := s.GetWorkspaceRootItemIDByWorkspace(ctx, row.WorkspaceID)
		workspaces = append(workspaces, &domain.Workspace{
			WorkspaceID: row.WorkspaceID,
			AccountID:   row.AccountID,
			Name:        row.Name,
			RootItemID:  rootItemID,
			CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return workspaces
}

func (s *Store) GetWorkspace(ctx context.Context, id string) (*domain.Workspace, bool) {
	row, err := s.q().GetWorkspace(ctx, id)
	if err != nil {
		return nil, false
	}
	ws := toWorkspace(sqlcgen.Workspace{
		WorkspaceID: row.WorkspaceID,
		AccountID:   row.AccountID,
		Name:        row.Name,
		CreatedAt:   row.CreatedAt,
	})
	ws.RootItemID, _ = s.GetWorkspaceRootItemIDByWorkspace(ctx, id)
	return ws, true
}

func (s *Store) IsWorkspaceAccessible(ctx context.Context, wsID, userID string) bool {
	accessible, err := s.q().IsWorkspaceAccessible(ctx, sqlcgen.IsWorkspaceAccessibleParams{
		WorkspaceID: wsID,
		UserID:      userID,
	})
	return err == nil && accessible
}

func (s *Store) CreateWorkspace(ctx context.Context, accountID, name string) *domain.Workspace {
	createdAt := nowTime()
	wsID := newID()
	rootItemID := newID()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil
	}
	defer tx.Rollback()
	qtx := s.q().WithTx(tx)

	if err := qtx.CreateWorkspace(ctx, sqlcgen.CreateWorkspaceParams{
		WorkspaceID: wsID,
		AccountID:   accountID,
		Name:        name,
		CreatedAt:   createdAt,
	}); err != nil {
		return nil
	}
	if err := qtx.CreateItem(ctx, sqlcgen.CreateItemParams{
		ID:          rootItemID,
		WorkspaceID: wsID,
		ParentID:    sql.NullString{},
		Label:       name,
		Level:       0,
		Description: "Workspace root",
		SummaryHtml: "",
		CreatedBy:   "system",
		CreatedAt:   createdAt,
	}); err != nil {
		return nil
	}
	if err := tx.Commit(); err != nil {
		return nil
	}
	return &domain.Workspace{
		WorkspaceID: wsID,
		AccountID:   accountID,
		Name:        name,
		RootItemID:  rootItemID,
		CreatedAt:   createdAt.Format(time.RFC3339),
	}
}

func (s *Store) GetWorkspaceRootItemIDByWorkspace(ctx context.Context, workspaceID string) (string, bool) {
	row, err := s.q().GetTreeRoot(ctx, workspaceID)
	if err != nil {
		return "", false
	}
	return row.ID, true
}

func toAccount(row sqlcgen.Account) *domain.Account {
	return &domain.Account{
		AccountID:          row.AccountID,
		Name:               row.Name,
		Plan:               row.Plan,
		StorageQuotaBytes:  row.StorageQuotaBytes,
		StorageUsedBytes:   row.StorageUsedBytes,
		MaxFileSizeBytes:   row.MaxFileSizeBytes,
		MaxUploadsPerFiveH: int64(row.MaxUploadsPer5h),
		MaxUploadsPerWeek:  int64(row.MaxUploadsPer1week),
		CreatedAt:          row.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toWorkspace(row sqlcgen.Workspace) *domain.Workspace {
	return &domain.Workspace{
		WorkspaceID: row.WorkspaceID,
		AccountID:   row.AccountID,
		Name:        row.Name,
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
	}
}
