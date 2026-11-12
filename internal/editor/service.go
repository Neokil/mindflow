package editor

import (
	"context"
	"mindflow/internal/editor/storage"
	"time"
)

// NewService invalidates pre-restart leases and constructs editor control.
// CheckSaveWithin validates a persisted lease in the caller's save transaction.

// Service owns this domain's behavior and persistence.
type Service struct {
	repository *storage.Repository
	publish    func(string, string, string)
}

// NewService constructs the domain service with explicit collaborators.
func NewService(ctx context.Context, repository *storage.Repository, publish func(string, string, string)) (*Service, error) {
	_, err := repository.ExpireLeases(ctx)

	return &Service{repository: repository, publish: publish}, err
}

// CheckSaveWithin checks persisted editor control inside the caller's save transaction.
func (a *Service) CheckSaveWithin(ctx context.Context, transaction *storage.Transaction, sessionID, pageID, clientID string, epoch int64, baseRevision, revision int) error {
	lease, err := a.repository.LeaseWithin(ctx, transaction, pageID)
	if storage.IsNotFound(err) {
		return failure("lease_lost", "Editor control expired.")
	}
	if err != nil {
		return err
	}

	return (LeaseState{SessionID: lease.SessionID, ClientID: lease.ClientID, Epoch: lease.Epoch, ExpiresAt: lease.ExpiresAt}).CheckSave(sessionID, clientID, epoch, baseRevision, revision, time.Now().Unix())
}
