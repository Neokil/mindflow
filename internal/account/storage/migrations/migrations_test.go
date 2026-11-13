package migrations_test

import (
	"database/sql"
	"mindflow/internal/account/storage/migrations"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // Register the SQLite test database driver.
)

func TestDisplayNamesUpgradeExistingAccounts(t *testing.T) {
	t.Parallel()
	database, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "account.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	provider, err := goose.NewProvider(goose.DialectSQLite3, database, migrations.Files(), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	_, err = provider.UpTo(t.Context(), 2)
	if err != nil {
		t.Fatal(err)
	}
	address := strings.Repeat("x", 90) + "@example.com"
	for _, entry := range []struct{ id, email string }{{"existing", "alice@example.com"}, {"long", address}} {
		_, err = database.ExecContext(t.Context(), "INSERT INTO users(id,email,role,status,created_at) VALUES(?,?,'user','active',0)", entry.id, entry.email)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = provider.Up(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct{ id, name string }{{"existing", "alice"}, {"long", strings.Repeat("x", 80)}} {
		var name string
		err = database.QueryRowContext(t.Context(), "SELECT display_name FROM users WHERE id=?", entry.id).Scan(&name)
		if err != nil {
			t.Fatal(err)
		}
		if name != entry.name {
			t.Fatalf("migration name %q, want %q", name, entry.name)
		}
	}
}
