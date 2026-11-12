package storage_test

import (
	"io/fs"
	"mindflow/internal/mindflow/storage"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
)

func fresh(t *testing.T) *storage.Database {
	t.Helper()
	database, err := storage.OpenData(t.Context(), filepath.Join(t.TempDir(), "mindflow.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	return database
}

func version(t *testing.T, database *storage.Database, name string) int64 {
	t.Helper()
	for _, domain := range storage.MigrationCatalogForTest() {
		if domain.Name != name {
			continue
		}
		provider, err := goose.NewProvider(goose.DialectSQLite3, database.SQLForTest(), domain.Files, goose.WithTableName("goose_"+name+"_version"), goose.WithDisableGlobalRegistry(true))
		if err != nil {
			t.Fatal(err)
		}
		current, err := provider.GetDBVersion(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		return current
	}
	t.Fatalf("unknown domain %s", name)

	return 0
}

func extended(t *testing.T, name, sql string) []storage.MigrationSetForTest {
	t.Helper()
	catalog := storage.MigrationCatalogForTest()
	for index, domain := range catalog {
		if domain.Name != name {
			continue
		}
		files := fstest.MapFS{}
		entries, err := fs.ReadDir(domain.Files, ".")
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			data, readErr := fs.ReadFile(domain.Files, entry.Name())
			if readErr != nil {
				t.Fatal(readErr)
			}
			files[entry.Name()] = &fstest.MapFile{Data: data}
		}
		files["00002_test.sql"] = &fstest.MapFile{Data: []byte(sql)}
		catalog[index].Files = files

		return catalog
	}
	t.Fatalf("unknown domain %s", name)

	return nil
}

func TestIndependentVersionsAndIdempotency(t *testing.T) {
	t.Parallel()
	database := fresh(t)
	catalog := extended(t, "canvas", "-- +goose Up\nCREATE TABLE canvas_upgrade_probe(id INTEGER PRIMARY KEY);\n-- +goose Down\nDROP TABLE canvas_upgrade_probe;\n")
	err := database.MigrateForTest(t.Context(), catalog)
	if err != nil {
		t.Fatal(err)
	}
	err = database.MigrateForTest(t.Context(), catalog)
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range storage.MigrationCatalogForTest() {
		expected := int64(1)
		if domain.Name == "canvas" || domain.Name == "installation" {
			expected = 2
		}
		actual := version(t, database, domain.Name)
		if actual != expected {
			t.Fatalf("%s version: got %d, want %d", domain.Name, actual, expected)
		}
	}
}

func TestFailedMigrationRollsBackAndCanRetry(t *testing.T) {
	t.Parallel()
	database := fresh(t)
	bad := extended(t, "canvas", "-- +goose Up\nCREATE TABLE failed_upgrade_probe(id INTEGER);\nINSERT INTO nonexistent_table VALUES(1);\n-- +goose Down\nDROP TABLE failed_upgrade_probe;\n")
	if database.MigrateForTest(t.Context(), bad) == nil {
		t.Fatal("invalid migration succeeded")
	}
	if version(t, database, "canvas") != 1 {
		t.Fatal("failed migration advanced history")
	}
	assertTable(t, database, "failed_upgrade_probe", false)
	good := extended(t, "canvas", "-- +goose Up\nCREATE TABLE failed_upgrade_probe(id INTEGER);\n-- +goose Down\nDROP TABLE failed_upgrade_probe;\n")
	err := database.MigrateForTest(t.Context(), good)
	if err != nil {
		t.Fatal(err)
	}
	assertTable(t, database, "failed_upgrade_probe", true)
}

func TestNewerDomainRefusesOtherPendingChanges(t *testing.T) {
	t.Parallel()
	database := fresh(t)
	_, err := database.SQLForTest().ExecContext(t.Context(), "INSERT INTO goose_canvas_version(version_id,is_applied) VALUES(99,1)")
	if err != nil {
		t.Fatal(err)
	}
	catalog := extended(t, "account", "-- +goose Up\nCREATE TABLE must_not_apply(id INTEGER);\n-- +goose Down\nDROP TABLE must_not_apply;\n")
	if database.MigrateForTest(t.Context(), catalog) == nil {
		t.Fatal("older binary accepted newer canvas history")
	}
	assertTable(t, database, "must_not_apply", false)
}

func TestLegacyUpgradePreservesContent(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "legacy.sqlite")
	database, err := storage.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	baseline, err := os.ReadFile("testdata/legacy_v2.sql")
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.SQLForTest().ExecContext(t.Context(), string(baseline))
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.SQLForTest().ExecContext(t.Context(), `INSERT INTO app_settings(id,initialized,config) VALUES(1,1,'{"remembered":true}');
 INSERT INTO users(id,email,role,status,created_at) VALUES('owner','owner@example.invalid','admin','active',1);
 INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES('workspace','owner','Preserve',1,1);
 INSERT INTO pages(id,workspace_id,name,position,document,updated_at) VALUES('page','workspace','Original',0,'{"schemaVersion":1}',1);
 INSERT INTO assets(id,owner_id,original,mime,width,height,digest,created_at) VALUES('asset','owner',x'000102','image/png',1,1,'hash',1);`)
	if err != nil {
		t.Fatal(err)
	}
	err = database.Migrate(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	err = database.Migrate(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var name, document, config string
	var image []byte
	err = database.SQLForTest().QueryRowContext(t.Context(), "SELECT w.name,p.document,s.config,a.original FROM workspaces w JOIN pages p ON p.workspace_id=w.id CROSS JOIN app_settings s CROSS JOIN assets a WHERE w.id='workspace'").Scan(&name, &document, &config, &image)
	if err != nil {
		t.Fatal(err)
	}
	if name != "Preserve" || document != "{\"schemaVersion\":1}" || config != "{\"remembered\":true}" || string(image) != "\x00\x01\x02" {
		t.Fatal("legacy upgrade changed stored content")
	}
	assertTable(t, database, "schema_migrations", false)
	for _, domain := range storage.MigrationCatalogForTest() {
		if version(t, database, domain.Name) == 0 {
			t.Fatalf("legacy domain %s was not adopted by Goose", domain.Name)
		}
	}
}

func TestUnknownLegacyAndBadDependenciesAreRejected(t *testing.T) {
	t.Parallel()
	database, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "future.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	_, err = database.SQLForTest().ExecContext(t.Context(), "CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY); INSERT INTO schema_migrations VALUES(99)")
	if err != nil {
		t.Fatal(err)
	}
	if database.Migrate(t.Context()) == nil {
		t.Fatal("newer legacy schema accepted")
	}
	assertTable(t, database, "goose_account_version", false)
	clean := fresh(t)
	for _, catalog := range [][]storage.MigrationSetForTest{{{Name: firstDomain, Dependencies: []string{"missing"}}}, {{Name: firstDomain, Dependencies: []string{"two"}}, {Name: "two", Dependencies: []string{firstDomain}}}} {
		err = clean.MigrateForTest(t.Context(), catalog)
		if err == nil || !strings.Contains(err.Error(), "invalid domain migration dependency") {
			t.Fatalf("wrong catalog validation result: %v", err)
		}
	}
}

func assertTable(t *testing.T, database *storage.Database, name string, expected bool) {
	t.Helper()
	var exists bool
	err := database.SQLForTest().QueryRowContext(t.Context(), "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)", name).Scan(&exists)
	if err != nil {
		t.Fatal(err)
	}
	if exists != expected {
		t.Fatalf("table %s existence: got %t, want %t", name, exists, expected)
	}
}

const firstDomain = "one"
