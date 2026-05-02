package repository

import (
	"context"

	"github.com/synthify/backend/packages/shared/joblog"
)

type JobLogRepository interface {
	LogJobEvent(ctx context.Context, e joblog.Event) error
}
