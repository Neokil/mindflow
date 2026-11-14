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
		return nil, fmt.Errorf("begin feedback transaction: %w", err)
	}

	return transaction, nil
}

// Within binds queries to a transaction shared by cooperating domains.
func (a *Repository) Within(transaction *Transaction) *Queries { return New(transaction) }

// IsNotFound classifies a missing row without exposing driver errors to policies.
func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }

// DeleteOwnerWithin removes reports and authored comments when an account is deleted.
func (a *Repository) DeleteOwnerWithin(ctx context.Context, transactionValue *Transaction, userID string) error {
	err := a.Within(transactionValue).FeedbackDeleteAuthorComments(ctx, userID)
	if err != nil {
		return err
	}

	return a.Within(transactionValue).FeedbackDeleteAuthorReports(ctx, userID)
}
