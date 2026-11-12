package storage

import (
	"context"
	"database/sql"
	"fmt"
	accountstorage "mindflow/internal/account/storage"
	canvasstorage "mindflow/internal/canvas/storage"
	editorstorage "mindflow/internal/editor/storage"
	emailstorage "mindflow/internal/email/storage"
	imagestorage "mindflow/internal/image/storage"
	installationstorage "mindflow/internal/installation/storage"
	invitationstorage "mindflow/internal/invitation/storage"
	"mindflow/internal/mindflow/storage/transaction"
	pagestorage "mindflow/internal/page/storage"
	resetstorage "mindflow/internal/passwordreset/storage"
	sessionstorage "mindflow/internal/session/storage"
	workspacestorage "mindflow/internal/workspace/storage"
	"net/url"
	"os"
	"path/filepath"

	// Register the pure-Go SQLite database/sql driver.
	_ "modernc.org/sqlite"
)

// Database owns a SQLite connection and its domain storage adapters.
type Database struct {
	connection     *sql.DB
	Accounts       *accountstorage.Repository
	Sessions       *sessionstorage.Repository
	Invitations    *invitationstorage.Repository
	PasswordResets *resetstorage.Repository
	Email          *emailstorage.Repository
	Installation   *installationstorage.Repository
	Workspaces     *workspacestorage.Repository
	Pages          *pagestorage.Repository
	Canvas         *canvasstorage.Repository
	Editor         *editorstorage.Repository
	Images         *imagestorage.Repository
}

// Open opens the database without modifying its schema.
func Open(ctx context.Context, path string) (*Database, error) {
	err := os.MkdirAll(filepath.Dir(path), privateDirectoryMode)
	if err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	dsn := "file:" + filepath.ToSlash(path) + "?" + url.Values{"_pragma": {"foreign_keys(1)", "busy_timeout(5000)", "journal_mode(WAL)", "synchronous(FULL)"}}.Encode()
	connection, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	connection.SetMaxOpenConns(1)
	err = connection.PingContext(ctx)
	if err != nil {
		_ = connection.Close()

		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Database{connection: connection, Accounts: accountstorage.NewRepository(connection), Sessions: sessionstorage.NewRepository(connection), Invitations: invitationstorage.NewRepository(connection), PasswordResets: resetstorage.NewRepository(connection), Email: emailstorage.NewRepository(connection), Installation: installationstorage.NewRepository(connection), Workspaces: workspacestorage.NewRepository(connection), Pages: pagestorage.NewRepository(connection), Canvas: canvasstorage.NewRepository(connection), Editor: editorstorage.NewRepository(connection), Images: imagestorage.NewRepository(connection)}, nil
}

// OpenData opens the database and applies every discovered domain's pending migrations.
func OpenData(ctx context.Context, path string) (*Database, error) {
	database, err := Open(ctx, path)
	if err != nil {
		return nil, err
	}
	err = database.Migrate(ctx)
	if err != nil {
		_ = database.Close()

		return nil, err
	}

	return database, nil
}

// Close releases the database connection.
func (a *Database) Close() error {
	err := a.connection.Close()
	if err != nil {
		return fmt.Errorf("close database: %w", err)
	}

	return nil
}

// Ping verifies database availability for readiness.
func (a *Database) Ping(ctx context.Context) error {
	err := a.connection.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("database readiness: %w", err)
	}

	return nil
}

// Begin starts the transaction used by cross-domain credential revocation.
func (a *Database) Begin(ctx context.Context) (*transaction.Transaction, error) {
	transaction, err := a.connection.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin database transaction: %w", err)
	}

	return transaction, nil
}

// SetupComplete reports whether the installation domain has been initialized.
func (a *Database) SetupComplete(ctx context.Context) (bool, error) {
	initialized, err := a.Installation.SetupStatus(ctx)
	if err != nil {
		return false, fmt.Errorf("read installation state: %w", err)
	}

	return initialized == 1, nil
}

// RevokeCredentials invalidates restored credentials and editor leases atomically.
func (a *Database) RevokeCredentials(ctx context.Context) error {
	transaction, err := a.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	err = a.Sessions.RevokeAllWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = a.Invitations.RevokeAllWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = a.PasswordResets.RevokeAllWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = a.Email.CancelAllWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = a.Editor.RevokeWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("commit credential revocation: %w", err)
	}

	return nil
}

const privateDirectoryMode = 0o700
