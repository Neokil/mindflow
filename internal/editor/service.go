package editor

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// NewService invalidates pre-restart leases and constructs editor control.
// CheckSaveWithin validates a persisted lease in the caller's save transaction.

// Service owns this domain's behavior and persistence.
type Service struct {
	db      *sql.DB
	publish func(string, string, string)
}

// NewService constructs the domain service with explicit collaborators.
func NewService(ctx context.Context, connection *sql.DB, publish func(string, string, string)) (*Service, error) {
	_, err := New(connection).ExpireLeases(ctx)

	return &Service{db: connection, publish: publish}, err
}

func loadOwnedPage(ctx context.Context, transaction *sql.Tx, id, owner string) (OwnedPageRow, error) {
	return New(transaction).OwnedPage(ctx, OwnedPageParams{ID: id, OwnerID: owner})
}

// CheckSaveWithin checks persisted editor control inside the caller's save transaction.
func (a *Service) CheckSaveWithin(ctx context.Context, transaction *sql.Tx, sessionID, pageID, clientID string, epoch int64, baseRevision, revision int) error {
	lease, err := New(transaction).PageLease(ctx, pageID)
	if errors.Is(err, sql.ErrNoRows) {
		return failure("lease_lost", "Editor control expired.")
	}
	if err != nil {
		return err
	}

	return (LeaseState{SessionID: lease.SessionID.String, ClientID: lease.ClientID, Epoch: lease.Epoch, ExpiresAt: lease.ExpiresAt}).CheckSave(sessionID, clientID, epoch, baseRevision, revision, time.Now().Unix())
}

// RevokeWithin invalidates all editor control in a stopped-instance restore transaction.
func RevokeWithin(ctx context.Context, transaction *sql.Tx) error {
	_, err := New(transaction).ExpireLeases(ctx)

	return err
}
