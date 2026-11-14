package installation

import (
	"context"
	"encoding/json"
	"mindflow/internal/account"
	"mindflow/internal/email"
	"mindflow/internal/installation/storage"
)

// Config supplies local development mode and the one-time installation credential.
type Config struct {
	Dev       bool
	Bootstrap string
}

// Service owns installation state and operator configuration.
type Service struct {
	welcome    *email.Outbox
	repository *storage.Repository
	Config     Config
	accounts   *account.Service
}

// NewService supplies the account creator used by first-run setup.
func NewService(repository *storage.Repository, accounts *account.Service, config Config, welcome *email.Outbox) *Service {
	return &Service{repository: repository, accounts: accounts, Config: config, welcome: welcome}
}

// Settings returns the saved operator configuration.
func (a *Service) Settings(ctx context.Context) (Settings, error) { return a.settings(ctx) }

// DeliveryConfig supplies the email domain without exposing account operations.
func (a *Service) DeliveryConfig(ctx context.Context) (email.Config, error) {
	settings, err := a.settings(ctx)

	return settings.SMTP, err
}

// PublicAddress supplies credential-link issuers with the configured application URL.
func (a *Service) PublicAddress(ctx context.Context) (string, error) {
	settings, err := a.settings(ctx)

	return settings.PublicURL, err
}

func repositoryError(err error) error {
	if storage.IsNotFound(err) {
		return notFound()
	}

	return err
}

// PublicAddressWithin reads link configuration without opening a nested database connection.
func (a *Service) PublicAddressWithin(ctx context.Context, transaction *storage.Transaction) (string, error) {
	raw, err := a.repository.Within(transaction).InstallationSettings(ctx)
	if err != nil {
		return "", err
	}
	var settings Settings
	err = json.Unmarshal([]byte(raw), &settings)

	return settings.PublicURL, wrap("decode transactional settings", err)
}
