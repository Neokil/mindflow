package storage

import (
	"context"
	"errors"
	"fmt"
)

// Check verifies SQLite integrity and foreign keys.
func (a *Database) Check(ctx context.Context) error {
	rows, err := a.connection.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return fmt.Errorf("check database: %w", err)
	}
	for rows.Next() {
		var result string
		err = rows.Scan(&result)
		if err != nil {
			_ = rows.Close()

			return fmt.Errorf("check database: %w", err)
		}
		if result != "ok" {
			_ = rows.Close()

			return fmt.Errorf("%w: %s", errIntegrity, result)
		}
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return fmt.Errorf("check database: %w", err)
	}
	rows, err = a.connection.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("check database: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		return errForeignKey
	}

	return checkRowsError(rows.Err())
}

var (
	errIntegrity  = errors.New("database integrity check failed")
	errForeignKey = errors.New("foreign key check found an inconsistency")
)

func checkRowsError(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("check database rows: %w", err)
}
