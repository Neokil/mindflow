package storage_test

import (
	"mindflow/internal/mindflow/storage"
	"path/filepath"
	"testing"
)

func TestDatabasePersistsSettings(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "mindflow.sqlite")
	dbValue, err := storage.OpenData(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dbValue.SQLForTest().ExecContext(t.Context(), `UPDATE app_settings SET config='{"test":true}' WHERE id=1`)
	if err != nil {
		t.Fatal(err)
	}
	checkErr20 := dbValue.Close()
	if checkErr20 != nil {
		t.Fatal(checkErr20)
	}
	dbValue, err = storage.OpenData(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dbValue.Close() }()
	var config string
	err = dbValue.SQLForTest().QueryRowContext(t.Context(), "SELECT config FROM app_settings WHERE id=1").Scan(&config)
	if err != nil {
		t.Fatal(err)
	}
	if config != `{"test":true}` {
		t.Fatalf("settings lost: %s", config)
	}
}

func TestRestoreRevokesCredentialsWithoutRemovingContent(t *testing.T) {
	t.Parallel()

	dbValue, err := storage.OpenData(t.Context(), filepath.Join(t.TempDir(), "restore.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dbValue.Close() }()
	_, err = dbValue.SQLForTest().ExecContext(t.Context(), `INSERT INTO users(id,email,role,status,created_at) VALUES('u','test@example.com','admin','active',1);
 INSERT INTO sessions(id,user_id,verifier_hash,csrf,expires_at) VALUES('s','u','hash','csrf',9999999999);
 INSERT INTO account_tokens(id,user_id,purpose,token_hash,expires_at) VALUES('t','u','reset','hash',9999999999);
 INSERT INTO jobs(id,user_id,kind,payload,run_at) VALUES('j','u','mail','secret',1);
 INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES('w','u','Keep me',1,1);`)
	if err != nil {
		t.Fatal(err)
	}
	err = dbValue.RevokeCredentials(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	err = dbValue.Check(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"sessions", "account_tokens", "jobs"} {
		var n int
		err = dbValue.SQLForTest().QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("%s retained credentials", table)
		}
	}
	var n int
	checkErr70 := dbValue.SQLForTest().QueryRowContext(t.Context(), "SELECT count(*) FROM workspaces").Scan(&n)
	if checkErr70 != nil {
		t.Fatal(checkErr70)
	}
	if n != 1 {
		t.Fatal("restore maintenance removed content")
	}
}
