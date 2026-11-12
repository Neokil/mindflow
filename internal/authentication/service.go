package authentication

import (
	"context"
	"database/sql"
	"errors"
	"mindflow/internal/authentication/dbquery"
	"mindflow/internal/fault"
	"sync"
)

// Config supplies development mode and the one-time setup credential.
type Config struct {
	Dev       bool
	Bootstrap string
}

// Service coordinates authentication operations and their SQLite transactions.
type Service struct {
	db        *sql.DB
	Config    Config
	hashMu    sync.Mutex
	dummyHash string
}

// New constructs the authentication service with explicit dependencies.
func New(connection *sql.DB, config Config) (*Service, error) {
	hash, err := hashPassword("dummy password never used")
	if err != nil {
		return nil, err
	}

	return &Service{db: connection, Config: config, dummyHash: hash}, nil
}

func repositoryError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fault.NotFound()
	}

	return err
}

// Initialized reports whether first-run setup is complete.
func (a *Service) Initialized(ctx context.Context) (bool, error) {
	var n int
	setupStatusRow, err := dbquery.New(a.db).SetupStatus(ctx)
	n = setupStatusRow

	return n == 1, err
}

// Settings loads operator-configured application settings.
func (a *Service) Settings(ctx context.Context) (Settings, error) { return a.settings(ctx) }

// Logout revokes the current session.
func (a *Service) Logout(ctx context.Context, actor Principal) error {
	_, err := dbquery.New(a.db).DeleteSession(ctx, actor.SessionID)

	return err
}

// SetAppearance stores a supported account appearance preference.
func (a *Service) SetAppearance(ctx context.Context, actor Principal, value string) error {
	if value != "system" && value != "light" && value != "dark" {
		return fault.Invalid("Invalid appearance.")
	}
	_, err := dbquery.New(a.db).SetAppearance(ctx, dbquery.SetAppearanceParams{Appearance: value, ID: actor.ID})

	return err
}

// SendTestEmail tests the saved SMTP settings for an administrator.
func (a *Service) SendTestEmail(ctx context.Context, actor Principal) error {
	requireAdministratorErr := requireAdministrator(actor)
	if requireAdministratorErr != nil {
		return requireAdministratorErr
	}
	settings, err := a.settings(ctx)
	if err != nil {
		return err
	}
	err = sendMail(ctx, settings.SMTP, actor.Email, "mindflow test email", "Your mindflow email settings are working.")
	if err != nil {
		return fault.Invalid("The SMTP test failed. Check the host, credentials, and TLS settings.")
	}

	return nil
}

func requireAdministrator(actor Principal) error {
	if actor.Role != RoleAdmin || actor.Status != StatusActive {
		return fault.Fault("forbidden", "Administrator access required.")
	}

	return nil
}

// SetupComplete reports whether first-run setup has been committed.
func SetupComplete(ctx context.Context, connection *sql.DB) (bool, error) {
	initialized, err := dbquery.New(connection).SetupStatus(ctx)

	return initialized == 1, fault.Wrap("read setup status", err)
}
