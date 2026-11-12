package authentication

import (
	"context"
	"mindflow/internal/authentication/dbquery"
)

func (a *Service) cleanup(ctx context.Context) {
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer func() { _ = transaction.Rollback() }()
	queries := dbquery.New(transaction)
	err = queries.DeleteOrphanMail(ctx)
	if err != nil {
		return
	}
	err = queries.DeleteExpiredTokens(ctx)
	if err != nil {
		return
	}
	err = queries.DeleteExpiredSessions(ctx)
	if err != nil {
		return
	}
	_ = transaction.Commit()
}
