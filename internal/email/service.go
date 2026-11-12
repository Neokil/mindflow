package email

import (
	"context"
	"encoding/json"
	"mindflow/internal/account"
	"mindflow/internal/email/storage"
	"time"
)

// Configuration supplies the operator's persisted SMTP settings.
type Configuration interface {
	DeliveryConfig(ctx context.Context) (Config, error)
}

// Authorization prevents delivery of revoked or consumed credential links.
type Authorization interface {
	CanDeliver(ctx context.Context, tokenHash string) (bool, error)
}

// Service owns SMTP delivery, queue retries and cancellation.
type Service struct {
	repository    *storage.Repository
	configuration Configuration
	authorization Authorization
}

// NewService supplies explicit configuration and credential-link authorization.
func NewService(repository *storage.Repository, configuration Configuration, authorization Authorization) *Service {
	return &Service{repository: repository, configuration: configuration, authorization: authorization}
}

// Process delivers one due message.
func (a *Service) Process(ctx context.Context) { a.runMail(ctx) }

// QueueWithin replaces a superseded message and queues the new credential link atomically.
func (a *Service) QueueWithin(ctx context.Context, transaction *storage.Transaction, userID, purpose, to, subject, body, tokenHash string) error {
	_, err := a.repository.Within(transaction).EmailDeletePurposeMail(ctx, storage.EmailDeletePurposeMailParams{UserID: userID, DedupKey: userID + ":" + purpose})
	if err != nil {
		return err
	}
	payload, err := json.Marshal(mailJob{To: to, Subject: subject, Body: body, TokenHash: tokenHash})
	if err != nil {
		return wrap("encode credential email", err)
	}
	_, err = a.repository.Within(transaction).EmailQueueMail(ctx, storage.EmailQueueMailParams{ID: newID(), UserID: userID, DedupKey: userID + ":" + purpose, Payload: string(payload), RunAt: time.Now().Unix()})

	return err
}

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
