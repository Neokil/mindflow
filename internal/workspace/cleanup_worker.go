package workspace

import (
	"context"
	"mindflow/internal/workspace/dbquery"
)

func (a *Service) cleanup(ctx context.Context) {
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer func() { _ = transaction.Rollback() }()
	queries := dbquery.New(transaction)
	err = queries.DeleteOrphanVariants(ctx)
	if err != nil {
		return
	}
	err = queries.DeleteExpiredReceipts(ctx)
	if err != nil {
		return
	}
	err = queries.DeleteUnusedAssets(ctx)
	if err != nil {
		return
	}
	_ = transaction.Commit()
}
