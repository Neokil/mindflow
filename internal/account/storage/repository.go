package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mindflow/internal/mindflow/storage/transaction"
	"time"
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
		return nil, fmt.Errorf("begin account transaction: %w", err)
	}

	return transaction, nil
}

// Within binds queries to a transaction shared by cooperating domains.
func (a *Repository) Within(transaction *Transaction) *Queries { return New(transaction) }

// IsNotFound classifies a missing row without exposing driver errors to policies.
func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }

// SessionRevokeUserWithin removes all sessions during an account credential change.
func (a *Repository) SessionRevokeUserWithin(ctx context.Context, transaction *Transaction, userID string) error {
	_, err := a.Within(transaction).SessionDeleteUserSessions(ctx, userID)

	return err
}

// SessionRevokeAllWithin is used by stopped-instance restore maintenance.
func (a *Repository) SessionRevokeAllWithin(ctx context.Context, transaction *Transaction) error {
	return a.Within(transaction).SessionRevokeSessions(ctx)
}

// InvitationRevokeUserWithin removes this account's invite links during a credential change.
func (a *Repository) InvitationRevokeUserWithin(ctx context.Context, transaction *Transaction, userID string) error {
	_, err := a.Within(transaction).InvitationDeleteUserTokens(ctx, userID)

	return err
}

// InvitationRevokeAllWithin removes invite links after a snapshot restore.
func (a *Repository) InvitationRevokeAllWithin(ctx context.Context, transaction *Transaction) error {
	return a.Within(transaction).InvitationRevokeTokens(ctx)
}

// InvitationCanDeliver authorizes delivery of an unexpired, unused invite link.
func (a *Repository) InvitationCanDeliver(ctx context.Context, hash string) (bool, error) {
	count, err := a.InvitationCountValidTokens(ctx, InvitationCountValidTokensParams{TokenHash: hash, ExpiresAt: time.Now().Unix()})

	return count > 0, err
}

// PasswordresetRevokeUserWithin removes this account's reset links during a credential change.
func (a *Repository) PasswordresetRevokeUserWithin(ctx context.Context, transaction *Transaction, userID string) error {
	_, err := a.Within(transaction).PasswordresetDeleteUserTokens(ctx, userID)

	return err
}

// PasswordresetRevokeAllWithin removes reset links after a snapshot restore.
func (a *Repository) PasswordresetRevokeAllWithin(ctx context.Context, transaction *Transaction) error {
	return a.Within(transaction).PasswordresetRevokeTokens(ctx)
}

// PasswordresetCanDeliver authorizes delivery of an unexpired, unused reset link.
func (a *Repository) PasswordresetCanDeliver(ctx context.Context, hash string) (bool, error) {
	count, err := a.PasswordresetCountValidTokens(ctx, PasswordresetCountValidTokensParams{TokenHash: hash, ExpiresAt: time.Now().Unix()})

	return count > 0, err
}
