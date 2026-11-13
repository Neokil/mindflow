package editor

import (
	"context"
	"mindflow/internal/workspace/storage"
	"time"
)

// NewService invalidates pre-restart leases and constructs editor control.
// CheckSaveWithin validates a persisted lease in the caller's save transaction.

// Sessions checks requester revocation through the session domain.
type Sessions interface {
	ActiveWithin(ctx context.Context, transaction *storage.Transaction, sessionID, userID string) (bool, error)
}

// Service owns this domain's behavior and persistence.
type Service struct {
	sessions   Sessions
	now        func() time.Time
	repository *storage.Repository
	publish    func(string, string, string)
}

// NewService constructs the domain service with explicit collaborators.
func NewService(ctx context.Context, repository *storage.Repository, publish func(string, string, string), sessions Sessions, now func() time.Time) (*Service, error) {
	_, err := repository.EditorExpireLeases(ctx)
	if err != nil {
		return nil, err
	}
	if now == nil {
		now = time.Now
	}
	err = repository.EditorCancelRequests(ctx)

	return &Service{repository: repository, publish: publish, sessions: sessions, now: now}, err
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

	request, readErr := a.repository.Within(transaction).EditorRequest(ctx, pageID)
	if readErr != nil && !storage.IsNotFound(readErr) {
		return readErr
	}
	if readErr == nil && request.Status == requestPending && request.Deadline <= a.now().UnixMilli() {
		return failure("lease_lost", "Control transfer is pending. This editor is now read-only.")
	}

	return (LeaseState{SessionID: lease.SessionID, ClientID: lease.ClientID, Epoch: lease.Epoch, ExpiresAt: lease.ExpiresAt}).CheckSave(sessionID, clientID, epoch, baseRevision, revision, a.now().Unix())
}
