package storage

import (
	"context"
	"database/sql"
	"fmt"
	accountstorage "mindflow/internal/account/storage"
	emailstorage "mindflow/internal/email/storage"
	feedbackstorage "mindflow/internal/feedback/storage"
	installationstorage "mindflow/internal/installation/storage"
	"mindflow/internal/mindflow/storage/transaction"
	releasestorage "mindflow/internal/release/storage"
	workspacestorage "mindflow/internal/workspace/storage"
	"net/url"
	"os"
	"path/filepath"

	// Register the pure-Go SQLite database/sql driver.
	_ "modernc.org/sqlite"
)

// Database owns a SQLite connection and its domain storage adapters.
type Database struct {
	Releases     *releasestorage.Repository
	Feedback     *feedbackstorage.Repository
	connection   *sql.DB
	Accounts     *accountstorage.Repository
	Email        *emailstorage.Repository
	Installation *installationstorage.Repository
	Workspaces   *workspacestorage.Repository
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

	return &Database{Feedback: feedbackstorage.NewRepository(connection), Releases: releasestorage.NewRepository(connection), connection: connection, Accounts: accountstorage.NewRepository(connection), Email: emailstorage.NewRepository(connection), Installation: installationstorage.NewRepository(connection), Workspaces: workspacestorage.NewRepository(connection)}, nil
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
	initialized, err := a.Installation.InstallationSetupStatus(ctx)
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
	err = a.Accounts.SessionRevokeAllWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = a.Accounts.InvitationRevokeAllWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = a.Accounts.PasswordresetRevokeAllWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = a.Email.CancelAllWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = a.Workspaces.RevokeWithin(ctx, transaction)
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
