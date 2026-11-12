package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mindflow/internal/fault"
)

// Check verifies SQLite integrity and foreign keys.
func Check(ctx context.Context, connection *sql.DB) error {
	rows, err := connection.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return fault.Wrap("maintenance.go", err)
	}
	for rows.Next() {
		var result string
		err = rows.Scan(&result)
		if err != nil {
			_ = rows.Close()

			return fault.Wrap("maintenance.go", err)
		}
		if result != "ok" {
			_ = rows.Close()

			return fmt.Errorf("%w: %s", errIntegrity, result)
		}
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return fault.Wrap("maintenance.go", err)
	}
	rows, err = connection.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return fault.Wrap("maintenance.go", err)
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		return errForeignKey
	}

	return fault.Wrap("maintenance.go", rows.Err())
}

var (
	errIntegrity  = errors.New("database integrity check failed")
	errForeignKey = errors.New("foreign key check found an inconsistency")
)
