package installation

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	// Register the pure-Go SQLite database/sql driver.
	_ "modernc.org/sqlite"
)

// Open opens SQLite with the application durability settings and migrations.
func Open(ctx context.Context, path string, domains ...string) (*sql.DB, error) {
	mkdirAllErr := os.MkdirAll(filepath.Dir(path), privateDirectoryMode)
	if mkdirAllErr != nil {
		return nil, wrap("store.go", mkdirAllErr)
	}
	dsn := "file:" + filepath.ToSlash(path) + "?" + url.Values{"_pragma": {"foreign_keys(1)", "busy_timeout(5000)", "journal_mode(WAL)", "synchronous(FULL)"}}.Encode()
	dbValue, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, wrap("store.go", err)
	}
	dbValue.SetMaxOpenConns(1)
	err = dbValue.PingContext(ctx)
	if err != nil {
		_ = dbValue.Close()

		return nil, wrap("store.go", err)
	}
	err = migrate(ctx, dbValue, domains...)
	if err != nil {
		_ = dbValue.Close()

		return nil, fmt.Errorf("migrate: %w", err)
	}

	return dbValue, nil
}

//go:embed schema.sql
var schema string

func migrate(ctx context.Context, dbValue *sql.DB, domains ...string) error {
	_, err := dbValue.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY);
	CREATE TABLE IF NOT EXISTS app_settings(id INTEGER PRIMARY KEY CHECK(id=1), initialized INTEGER NOT NULL DEFAULT 0, config TEXT NOT NULL DEFAULT '{}');
	INSERT OR IGNORE INTO app_settings(id) VALUES(1);`)
	if err != nil {
		return wrap("store.go", err)
	}
	var version int
	err = dbValue.QueryRowContext(ctx, "SELECT COALESCE(MAX(version),0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return wrap("store.go", err)
	}
	if version > supportedSchemaVersion {
		return fmt.Errorf("%w: %d", errFutureSchema, version)
	}
	transaction, err := dbValue.BeginTx(ctx, nil)
	if err != nil {
		return wrap("store.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	_, err = transaction.ExecContext(ctx, schema+strings.Join(domains, "\n"))
	if err != nil {
		return wrap("store.go", err)
	}

	return wrap("store.go", transaction.Commit())
}

var errFutureSchema = errors.New("database schema is newer than this application supports")
