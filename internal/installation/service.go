package installation

import (
	"context"
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
	repository *storage.Repository
	Config     Config
	accounts   *account.Service
}

// NewService supplies the account creator used by first-run setup.
func NewService(repository *storage.Repository, accounts *account.Service, config Config) *Service {
	return &Service{repository: repository, accounts: accounts, Config: config}
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
