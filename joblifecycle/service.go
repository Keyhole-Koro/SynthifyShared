package joblifecycle

import (
	"context"

	"github.com/synthify/backend/packages/shared/jobstatus"
)

type Repository interface {
	MarkProcessingJobRunning(ctx context.Context, jobID string) error
	FailProcessingJob(ctx context.Context, jobID, errorMessage string) error
	CompleteProcessingJob(ctx context.Context, jobID string) error
}

type Logger interface {
	Printf(format string, v ...any)
}

type Service struct {
	repo     Repository
	notifier jobstatus.Notifier
	logger   Logger
}

func New(repo Repository, notifier jobstatus.Notifier, logger Logger) *Service {
	return &Service{repo: repo, notifier: notifier, logger: logger}
}

func (s *Service) NotifyQueued(ctx context.Context, payload jobstatus.Payload) {
	if s.notifier == nil {
		return
	}
	if err := s.notifier.Queued(ctx, payload); err != nil {
		s.logf("jobstatus: failed to notify queued: %v", err)
	}
}

func (s *Service) MarkRunning(ctx context.Context, payload jobstatus.Payload) error {
	if err := s.repo.MarkProcessingJobRunning(ctx, payload.JobID); err != nil {
		return err
	}
	if s.notifier == nil {
		return nil
	}
	if err := s.notifier.Running(ctx, payload); err != nil {
		s.logf("jobstatus: failed to notify running: %v", err)
	}
	return nil
}

func (s *Service) TryFail(ctx context.Context, payload jobstatus.Payload, errorMessage string) {
	if err := s.repo.FailProcessingJob(ctx, payload.JobID, errorMessage); err != nil {
		s.logf("repository: failed to mark job failed: %v", err)
	}
	if s.notifier == nil {
		return
	}
	if err := s.notifier.Failed(ctx, payload, errorMessage); err != nil {
		s.logf("jobstatus: failed to notify failure: %v", err)
	}
}

func (s *Service) Complete(ctx context.Context, payload jobstatus.Payload) error {
	if err := s.repo.CompleteProcessingJob(ctx, payload.JobID); err != nil {
		return err
	}
	if s.notifier == nil {
		return nil
	}
	if err := s.notifier.Completed(ctx, payload); err != nil {
		s.logf("jobstatus: failed to notify completion: %v", err)
	}
	return nil
}

func (s *Service) logf(format string, v ...any) {
	if s.logger != nil {
		s.logger.Printf(format, v...)
	}
}
