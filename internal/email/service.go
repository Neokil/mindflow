package email

import (
	"context"
	"mindflow/internal/account"
	"mindflow/internal/email/storage"
)

// Configuration supplies the operator's persisted SMTP settings.
type Configuration interface {
	DeliveryConfig(ctx context.Context) (Config, error)
}

// Authorization prevents delivery of revoked or consumed credential links.
type Authorization interface {
	CanDeliverNotice(ctx context.Context, userID string) (bool, error)
	CanDeliverUpdate(ctx context.Context, userID string) (bool, error)
	CanDeliverInactivity(ctx context.Context, userID, purpose string) (bool, error)
	CanDeliver(ctx context.Context, tokenHash string) (bool, error)
}

// Service owns SMTP delivery, queue retries and cancellation.
type Service struct {
	*Outbox

	repository    *storage.Repository
	configuration Configuration
	authorization Authorization
}

// NewService supplies explicit configuration and credential-link authorization.
func NewService(repository *storage.Repository, configuration Configuration, authorization Authorization) *Service {
	return &Service{Outbox: NewOutbox(repository), repository: repository, configuration: configuration, authorization: authorization}
}

// Process delivers one due message.
func (a *Service) Process(ctx context.Context) { a.runMail(ctx) }

// Test sends a test message using saved operator settings.
func (a *Service) Test(ctx context.Context, actor account.User) error {
	err := account.RequireAdministrator(actor)
	if err != nil {
		return err
	}
	config, err := a.configuration.DeliveryConfig(ctx)
	if err != nil {
		return err
	}
	err = Send(ctx, config, actor.Email, "mindflow test email", "Your mindflow email settings are working.")
	if err != nil {
		return invalid("The SMTP test failed. Check the host, credentials, and TLS settings.")
	}

	return nil
}

const (
	smtpConnectSeconds = 10
	smtpWriteSeconds   = 20
	maxMailAttempts    = 8
	retryBaseSeconds   = 15
)
