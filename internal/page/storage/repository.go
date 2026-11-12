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
		return nil, fmt.Errorf("begin page transaction: %w", err)
	}

	return transaction, nil
}

// Within binds queries to a transaction shared by cooperating domains.
func (a *Repository) Within(transaction *Transaction) *Queries { return New(transaction) }

// IsNotFound classifies a missing row without exposing driver errors to policies.
func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }

// PositionWithin updates a page position and reports the number of matching pages.
func (a *Repository) PositionWithin(ctx context.Context, transaction *Transaction, index int, id, workspaceID string) (int64, error) {
	result, err := a.Within(transaction).PositionPage(ctx, PositionPageParams{Position: index, ID: id, WorkspaceID: workspaceID})

	return affectedRows(result, err)
}

func affectedRows(result sql.Result, err error) (int64, error) {
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read affected row count: %w", err)
	}

	return count, nil
}
