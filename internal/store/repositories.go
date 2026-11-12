package store

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"mindflow/internal/domain"
)

type Options struct {
	Dev       bool
	Bootstrap string
}

// Repositories is the SQLite adapter. Domain decisions remain in domain models;
// state-dependent checks and writes run in the same database transaction.
type Repositories struct {
	DB              *sql.DB
	Config          Options
	hashMu, mediaMu sync.Mutex
	dummyHash       string
	publish         func(string, string, string)
}

func NewRepositories(db *sql.DB, options Options, publish func(string, string, string)) (*Repositories, error) {
	r := &Repositories{DB: db, Config: options, publish: publish}
	var err error
	r.dummyHash, err = hashPassword("dummy password never used")
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec("UPDATE page_leases SET epoch=epoch+1,expires_at=0,session_id=NULL,client_id=''"); err != nil {
		return nil, err
	}
	return r, nil
}
func mapRepositoryError(err *error) {
	if errors.Is(*err, sql.ErrNoRows) {
		*err = domain.NotFound()
	}
}
func (r *Repositories) Initialized(ctx context.Context) (bool, error) {
	var n int
	err := r.DB.QueryRowContext(ctx, "SELECT initialized FROM app_settings WHERE id=1").Scan(&n)
	return n == 1, err
}
func (r *Repositories) Settings(ctx context.Context) (Settings, error) { return r.settings(ctx) }
func (r *Repositories) Logout(ctx context.Context, actor domain.Principal) error {
	_, err := r.DB.ExecContext(ctx, "DELETE FROM sessions WHERE id=?", actor.SessionID)
	return err
}
func (r *Repositories) SetAppearance(ctx context.Context, actor domain.Principal, value string) error {
	if value != "system" && value != "light" && value != "dark" {
		return domain.Invalid("Invalid appearance.")
	}
	_, err := r.DB.ExecContext(ctx, "UPDATE users SET appearance=? WHERE id=?", value, actor.ID)
	return err
}
