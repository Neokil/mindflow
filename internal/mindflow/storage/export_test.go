package storage

import (
	"context"
	"database/sql"
	"io/fs"
)

// SQLForTest permits white-box storage assertions without exporting a production SQL handle.
func (a *Database) SQLForTest() *sql.DB { return a.connection }

// MigrationSetForTest supplies a test-local extension to one domain's embedded migrations.
type MigrationSetForTest struct {
	Name         string
	Files        fs.FS
	Dependencies []string
}

// MigrationCatalogForTest returns copies of the discovered domain descriptors.
func MigrationCatalogForTest() []MigrationSetForTest {
	catalog := discoveredMigrations()
	out := make([]MigrationSetForTest, 0, len(catalog))
	for _, domain := range catalog {
		out = append(out, MigrationSetForTest{Name: domain.name, Files: domain.files, Dependencies: domain.dependencies})
	}

	return out
}

// MigrateForTest runs the production coordinator against a test-local catalog.
func (a *Database) MigrateForTest(ctx context.Context, catalog []MigrationSetForTest) error {
	domains := make([]domainMigrations, 0, len(catalog))
	for _, domain := range catalog {
		domains = append(domains, domainMigrations{name: domain.Name, files: domain.Files, dependencies: domain.Dependencies})
	}

	return a.migrate(ctx, domains)
}
