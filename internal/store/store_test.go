package store

import (
	"path/filepath"
	"testing"
)

func TestDatabasePersistsSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mindflow.sqlite")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE app_settings SET config='{"test":true}' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var config string
	if err = db.QueryRow("SELECT config FROM app_settings WHERE id=1").Scan(&config); err != nil {
		t.Fatal(err)
	}
	if config != `{"test":true}` {
		t.Fatalf("settings lost: %s", config)
	}
}
