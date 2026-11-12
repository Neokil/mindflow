package workspace

import (
	"context"
	"database/sql"
	"errors"
	"mindflow/internal/fault"
	"mindflow/internal/workspace/dbquery"
	"sync"
	"time"
)

// Service coordinates workspace operations and their SQLite transactions.
type Service struct {
	db      *sql.DB
	mediaMu sync.Mutex
	publish func(string, string, string)
}

// New constructs the workspace service with explicit dependencies.
func New(ctx context.Context, db *sql.DB, publish func(string, string, string)) (*Service, error) {
	s := &Service{db: db, publish: publish}
	_, err := dbquery.New(db).ExpireLeases(ctx)

	return s, err
}

func repositoryError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fault.NotFound()
	}

	return err
}

// RunWorkers processes due jobs until the context is canceled.
func (a *Service) RunWorkers(ctx context.Context) {
	tick := time.NewTicker(workerIntervalSeconds * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			a.RunVariant(ctx)
			a.cleanup(ctx)
		}
	}
}

// RevokeEditorControl invalidates all leases on a stopped instance after restore.
func RevokeEditorControl(ctx context.Context, connection *sql.DB) error {
	_, err := dbquery.New(connection).ExpireLeases(ctx)

	return fault.Wrap("revoke editor control", err)
}
