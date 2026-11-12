package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mindflow/internal/mindflow/storage/transaction"
)

// Transaction is the handle shared by collaborating domain storage adapters.
type Transaction = transaction.Transaction

// Repository owns this domain's database queries and transaction binding.
type Repository struct {
	*Queries

	connection *sql.DB
}

// NewRepository binds queries to the supplied SQLite connection.
func NewRepository(connection *sql.DB) *Repository {
	return &Repository{Queries: New(connection), connection: connection}
}

// Begin starts a caller-owned transaction for an atomic domain operation.
func (a *Repository) Begin(ctx context.Context) (*Transaction, error) {
	transaction, err := a.connection.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin editor transaction: %w", err)
	}

	return transaction, nil
}

// Within binds queries to a transaction shared by cooperating domains.
func (a *Repository) Within(transaction *Transaction) *Queries { return New(transaction) }

// IsNotFound classifies a missing row without exposing driver errors to policies.
func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }

// LeaseRecord represents persisted editor control without SQL null values.
type LeaseRecord struct {
	SessionID, ClientID string
	Epoch, ExpiresAt    int64
}

// Lease reads the current persisted editor control.
func (a *Repository) Lease(ctx context.Context, pageID string) (LeaseRecord, error) {
	row, err := a.PageLease(ctx, pageID)

	return LeaseRecord{SessionID: row.SessionID.String, ClientID: row.ClientID, Epoch: row.Epoch, ExpiresAt: row.ExpiresAt}, err
}

// LeaseWithin reads editor control inside the document transaction.
func (a *Repository) LeaseWithin(ctx context.Context, transaction *Transaction, pageID string) (LeaseRecord, error) {
	row, err := a.Within(transaction).PageLease(ctx, pageID)

	return LeaseRecord{SessionID: row.SessionID.String, ClientID: row.ClientID, Epoch: row.Epoch, ExpiresAt: row.ExpiresAt}, err
}

// SaveLeaseWithin persists the policy result using SQL null for unowned control.
func (a *Repository) SaveLeaseWithin(ctx context.Context, transaction *Transaction, pageID, sessionID, clientID string, epoch, expires int64) error {
	_, err := a.Within(transaction).UpdateLease(ctx, UpdateLeaseParams{PageID: pageID, SessionID: sql.NullString{String: sessionID, Valid: sessionID != ""}, ClientID: clientID, Epoch: epoch, ExpiresAt: expires})

	return err
}

// RevokeWithin invalidates all editor control in a stopped-instance restore transaction.
func (a *Repository) RevokeWithin(ctx context.Context, transaction *Transaction) error {
	_, err := a.Within(transaction).ExpireLeases(ctx)

	return err
}

// OwnedPageWithin reads page ownership in the editor transaction.
func (a *Repository) OwnedPageWithin(ctx context.Context, transaction *Transaction, id, owner string) (OwnedPageRow, error) {
	return a.Within(transaction).OwnedPage(ctx, OwnedPageParams{ID: id, OwnerID: owner})
}
