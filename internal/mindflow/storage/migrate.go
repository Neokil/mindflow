package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"

	"github.com/pressly/goose/v3"
)

//go:generate go run ../../../cmd/genmigrations

type domainMigrations struct {
	name         string
	files        fs.FS
	dependencies []string
}
type domainProvider struct {
	name     string
	provider *goose.Provider
}

var (
	errNewerDomain         = errors.New("database domain version is newer than this application")
	errNewerLegacy         = errors.New("legacy database schema is newer than this application")
	errMigrationDependency = errors.New("invalid domain migration dependency")
)

// Migrate applies every discovered domain's pending migrations through Goose.
func (a *Database) Migrate(ctx context.Context) error { return a.migrate(ctx, discoveredMigrations()) }

func (a *Database) migrate(ctx context.Context, catalog []domainMigrations) error {
	err := a.checkLegacy(ctx)
	if err != nil {
		return err
	}
	ordered, err := orderMigrations(catalog)
	if err != nil {
		return err
	}
	providers, err := a.providers(ctx, ordered)
	if err != nil {
		return err
	}
	for _, domain := range providers {
		_, err = domain.provider.Up(ctx)
		if err != nil {
			return fmt.Errorf("migrate %s: %w", domain.name, err)
		}
	}

	return nil
}

func (a *Database) providers(ctx context.Context, catalog []domainMigrations) ([]domainProvider, error) {
	providers := make([]domainProvider, 0, len(catalog))
	for _, domain := range catalog {
		provider, err := goose.NewProvider(goose.DialectSQLite3, a.connection, domain.files, goose.WithTableName("goose_"+domain.name+"_version"), goose.WithDisableGlobalRegistry(true), goose.WithSlog(slog.Default().With("domain", domain.name)))
		if err != nil {
			return nil, fmt.Errorf("prepare %s migrations: %w", domain.name, err)
		}
		current, target, err := provider.GetVersions(ctx)
		if err != nil {
			return nil, fmt.Errorf("read %s migration versions: %w", domain.name, err)
		}
		if current > target {
			return nil, fmt.Errorf("%w: %s is %d; this binary supports %d", errNewerDomain, domain.name, current, target)
		}
		providers = append(providers, domainProvider{name: domain.name, provider: provider})
	}

	return providers, nil
}

func (a *Database) checkLegacy(ctx context.Context) error {
	var exists bool
	err := a.connection.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='schema_migrations')").Scan(&exists)
	if err != nil {
		return fmt.Errorf("check legacy migration history: %w", err)
	}
	if !exists {
		return nil
	}
	var version int
	err = a.connection.QueryRowContext(ctx, "SELECT COALESCE(MAX(version),0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return fmt.Errorf("read legacy migration version: %w", err)
	}
	if version > legacySchemaVersion {
		return fmt.Errorf("%w: %d", errNewerLegacy, version)
	}

	return nil
}

func orderMigrations(catalog []domainMigrations) ([]domainMigrations, error) {
	pending := map[string]domainMigrations{}
	for _, domain := range catalog {
		if _, exists := pending[domain.name]; exists {
			return nil, fmt.Errorf("%w: duplicate domain %s", errMigrationDependency, domain.name)
		}
		pending[domain.name] = domain
	}
	for _, domain := range catalog {
		for _, dependency := range domain.dependencies {
			if _, exists := pending[dependency]; !exists {
				return nil, fmt.Errorf("%w: %s requires missing %s", errMigrationDependency, domain.name, dependency)
			}
		}
	}
	ordered := make([]domainMigrations, 0, len(catalog))
	applied := map[string]bool{}
	for len(pending) > 0 {
		ready := readyMigrations(pending, applied)
		if len(ready) == 0 {
			return nil, fmt.Errorf("%w: dependency cycle", errMigrationDependency)
		}
		for _, name := range ready {
			ordered = append(ordered, pending[name])
			applied[name] = true
			delete(pending, name)
		}
	}

	return ordered, nil
}

func readyMigrations(pending map[string]domainMigrations, applied map[string]bool) []string {
	ready := []string{}
	for name, domain := range pending {
		available := true
		for _, dependency := range domain.dependencies {
			if !applied[dependency] {
				available = false

				break
			}
		}
		if available {
			ready = append(ready, name)
		}
	}
	sort.Strings(ready)

	return ready
}

const legacySchemaVersion = 2
