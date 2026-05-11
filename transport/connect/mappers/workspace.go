package mappers

import (
	"github.com/synthify/backend/packages/shared/domain"
	treev1 "github.com/synthify/backend/packages/shared/gen/synthify/tree/v1"
)

func ToProtoWorkspace(ws *domain.Workspace) *treev1.Workspace {
	if ws == nil {
		return nil
	}
	return &treev1.Workspace{
		WorkspaceId: ws.WorkspaceID,
		Name:        ws.Name,
		OwnerId:     ws.AccountID,
		CreatedAt:   ws.CreatedAt,
	}
}
