package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/pressly/goose/v3"
)

//go:generate go run ../../../cmd/genmigrations

type domainMigrations struct {
	name  string
	files fs.FS
}
type domainProvider struct {
	name     string
	provider *goose.Provider
}

var errNewerDomain = errors.New("database domain version is newer than this application")

// Migrate applies every discovered domain's pending migrations through Goose.
func (a *Database) Migrate(ctx context.Context) error { return a.migrate(ctx, discoveredMigrations()) }

func (a *Database) migrate(ctx context.Context, catalog []domainMigrations) error {
	providers, err := a.providers(ctx, catalog)
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
