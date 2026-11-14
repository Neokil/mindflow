package release

import (
	"context"
	"fmt"
	"mindflow/internal/mindflow/storage/transaction"
	"mindflow/internal/release/storage"
)

// Subscribers supplies opted-in recipients without exposing account tables.
type Subscribers interface {
	NotifyWithin(ctx context.Context, transactionValue *transaction.Transaction, version string) error
}

// Service persists the highest observed release in the same transaction as announcements.
type Service struct {
	repository  *storage.Repository
	subscribers Subscribers
}

// NewService supplies release-owned storage and the subscriber collaborator.
func NewService(repository *storage.Repository, subscribers Subscribers) *Service {
	return &Service{repository: repository, subscribers: subscribers}
}

// Observe ignores development builds, first-install baselines, restarts and downgrades.
func (a *Service) Observe(ctx context.Context, version string) error {
	if !Valid(version) {
		return nil
	}
	transactionValue, err := a.repository.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transactionValue.Rollback() }()
	queries := a.repository.Within(transactionValue)
	current, err := queries.ReleaseCurrent(ctx)
	first := storage.IsNotFound(err)
	if err != nil && !first {
		return err
	}
	if !first && Compare(version, current) <= 0 {
		return nil
	}
	if !first {
		err = a.subscribers.NotifyWithin(ctx, transactionValue, version)
		if err != nil {
			return err
		}
	}
	err = queries.ReleaseObserve(ctx, version)
	if err != nil {
		return err
	}
	err = transactionValue.Commit()
	if err != nil {
		return fmt.Errorf("record observed release: %w", err)
	}

	return nil
}
