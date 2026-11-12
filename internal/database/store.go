package database

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"mindflow/internal/fault"
	"net/url"
	"os"
	"path/filepath"

	// Register the pure-Go SQLite database/sql driver.
	_ "modernc.org/sqlite"
)

// Open opens SQLite with the application durability settings and migrations.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	mkdirAllErr := os.MkdirAll(filepath.Dir(path), privateDirectoryMode)
	if mkdirAllErr != nil {
		return nil, fault.Wrap("store.go", mkdirAllErr)
	}
	dsn := "file:" + filepath.ToSlash(path) + "?" + url.Values{"_pragma": {"foreign_keys(1)", "busy_timeout(5000)", "journal_mode(WAL)", "synchronous(FULL)"}}.Encode()
	dbValue, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fault.Wrap("store.go", err)
	}
	dbValue.SetMaxOpenConns(1)
	err = dbValue.PingContext(ctx)
	if err != nil {
		_ = dbValue.Close()

		return nil, fault.Wrap("store.go", err)
	}
	err = migrate(ctx, dbValue)
	if err != nil {
		_ = dbValue.Close()

		return nil, fmt.Errorf("migrate: %w", err)
	}

	return dbValue, nil
}

//go:embed schema.sql
var schema string

func migrate(ctx context.Context, dbValue *sql.DB) error {
	_, err := dbValue.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY);
	CREATE TABLE IF NOT EXISTS app_settings(id INTEGER PRIMARY KEY CHECK(id=1), initialized INTEGER NOT NULL DEFAULT 0, config TEXT NOT NULL DEFAULT '{}');
	INSERT OR IGNORE INTO app_settings(id) VALUES(1);`)
	if err != nil {
		return fault.Wrap("store.go", err)
	}
	var version int
	err = dbValue.QueryRowContext(ctx, "SELECT COALESCE(MAX(version),0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return fault.Wrap("store.go", err)
	}
	if version > supportedSchemaVersion {
		return fmt.Errorf("%w: %d", errFutureSchema, version)
	}
	transaction, err := dbValue.BeginTx(ctx, nil)
	if err != nil {
		return fault.Wrap("store.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	_, err = transaction.ExecContext(ctx, schema)
	if err != nil {
		return fault.Wrap("store.go", err)
	}

	return fault.Wrap("store.go", transaction.Commit())
}

var errFutureSchema = errors.New("database schema is newer than this application supports")
