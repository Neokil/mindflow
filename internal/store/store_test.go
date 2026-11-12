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

func TestRestoreRevokesCredentialsWithoutRemovingContent(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "restore.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`INSERT INTO users(id,email,role,status,created_at) VALUES('u','test@example.com','admin','active',1);
 INSERT INTO sessions(id,user_id,verifier_hash,csrf,expires_at) VALUES('s','u','hash','csrf',9999999999);
 INSERT INTO account_tokens(id,user_id,purpose,token_hash,expires_at) VALUES('t','u','reset','hash',9999999999);
 INSERT INTO jobs(id,user_id,kind,payload,run_at) VALUES('j','u','mail','secret',1);
 INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES('w','u','Keep me',1,1);`); err != nil {
		t.Fatal(err)
	}
	if err = RevokeCredentials(db); err != nil {
		t.Fatal(err)
	}
	if err = Check(db); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"sessions", "account_tokens", "jobs"} {
		var n int
		if err = db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("%s retained credentials", table)
		}
	}
	var n int
	db.QueryRow("SELECT count(*) FROM workspaces").Scan(&n)
	if n != 1 {
		t.Fatal("restore maintenance removed content")
	}
}
