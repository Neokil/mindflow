package storage_test

import (
	"io/fs"
	"mindflow/internal/mindflow/storage"
	"path/filepath"
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
	catalog := extended(t, "workspace", "-- +goose Up\nCREATE TABLE canvas_upgrade_probe(id INTEGER PRIMARY KEY);\n-- +goose Down\nDROP TABLE canvas_upgrade_probe;\n")
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
		if domain.Name == "workspace" {
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
	bad := extended(t, "workspace", "-- +goose Up\nCREATE TABLE failed_upgrade_probe(id INTEGER);\nINSERT INTO nonexistent_table VALUES(1);\n-- +goose Down\nDROP TABLE failed_upgrade_probe;\n")
	if database.MigrateForTest(t.Context(), bad) == nil {
		t.Fatal("invalid migration succeeded")
	}
	if version(t, database, "workspace") != 1 {
		t.Fatal("failed migration advanced history")
	}
	assertTable(t, database, "failed_upgrade_probe", false)
	good := extended(t, "workspace", "-- +goose Up\nCREATE TABLE failed_upgrade_probe(id INTEGER);\n-- +goose Down\nDROP TABLE failed_upgrade_probe;\n")
	err := database.MigrateForTest(t.Context(), good)
	if err != nil {
		t.Fatal(err)
	}
	assertTable(t, database, "failed_upgrade_probe", true)
}

func TestNewerDomainRefusesOtherPendingChanges(t *testing.T) {
	t.Parallel()
	database := fresh(t)
	_, err := database.SQLForTest().ExecContext(t.Context(), "INSERT INTO goose_workspace_version(version_id,is_applied) VALUES(99,1)")
	if err != nil {
		t.Fatal(err)
	}
	catalog := extended(t, "account", "-- +goose Up\nCREATE TABLE must_not_apply(id INTEGER);\n-- +goose Down\nDROP TABLE must_not_apply;\n")
	if database.MigrateForTest(t.Context(), catalog) == nil {
		t.Fatal("older binary accepted newer canvas history")
	}
	assertTable(t, database, "must_not_apply", false)
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

func TestEachDomainMigratesWithoutOtherSchemas(t *testing.T) {
	t.Parallel()
	for _, domain := range storage.MigrationCatalogForTest() {
		t.Run(domain.Name, func(t *testing.T) {
			t.Parallel()
			database, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "isolated.sqlite"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			provider, err := goose.NewProvider(goose.DialectSQLite3, database.SQLForTest(), domain.Files, goose.WithTableName("goose_"+domain.Name+"_version"), goose.WithDisableGlobalRegistry(true))
			if err != nil {
				t.Fatal(err)
			}
			_, err = provider.Up(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			err = database.Check(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			_, err = provider.DownTo(t.Context(), 0)
			if err != nil {
				t.Fatal(err)
			}
			_, err = provider.Up(t.Context())
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
