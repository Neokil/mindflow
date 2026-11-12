package authentication

import (
	"context"
	"database/sql"
	"mindflow/internal/authentication/dbquery"
	"mindflow/internal/fault"
)

// RevokeCredentials is for an operator working on a stopped instance after restore.
func RevokeCredentials(ctx context.Context, db *sql.DB) error {
	transaction, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fault.Wrap("maintenance.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	queries := dbquery.New(transaction)
	err = queries.RevokeSessions(ctx)
	if err != nil {
		return err
	}
	err = queries.RevokeTokens(ctx)
	if err != nil {
		return err
	}
	err = queries.RevokeMail(ctx)
	if err != nil {
		return err
	}

	return fault.Wrap("maintenance.go", transaction.Commit())
}
