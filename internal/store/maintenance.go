package store

import (
	"database/sql"
	"fmt"
)

// RevokeCredentials is for an operator working on a stopped instance after restore.
func RevokeCredentials(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{"DELETE FROM sessions", "DELETE FROM account_tokens", "DELETE FROM jobs WHERE kind='mail'", "UPDATE page_leases SET session_id=NULL,client_id='',expires_at=0,epoch=epoch+1"} {
		if _, err = tx.Exec(q); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func Check(db *sql.DB) error {
	rows, err := db.Query("PRAGMA integrity_check")
	if err != nil {
		return err
	}
	for rows.Next() {
		var result string
		if err = rows.Scan(&result); err != nil {
			rows.Close()
			return err
		}
		if result != "ok" {
			rows.Close()
			return fmt.Errorf("integrity check: %s", result)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	rows, err = db.Query("PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return fmt.Errorf("foreign key check found an inconsistency")
	}
	return rows.Err()
}
