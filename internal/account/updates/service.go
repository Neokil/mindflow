// Package updates owns consent and capability links for product-update emails.
package updates

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"mindflow/internal/account"
	"mindflow/internal/account/storage"
	"mindflow/internal/mindflow/storage/transaction"
	"net/http"
	"strings"
)

// Mail queues notices and cancels release messages when consent is withdrawn.
type Mail interface {
	QueueWithin(ctx context.Context, transactionValue *transaction.Transaction, userID, purpose, to, subject, body, tokenHash string) error
	CancelUpdatesWithin(ctx context.Context, transactionValue *transaction.Transaction, userID string) error
}

// Configuration supplies the configured public application address.
type Configuration interface {
	PublicAddress(ctx context.Context) (string, error)
}

// Service manages explicit opt-in and upgrade notification delivery.
type Service struct {
	repository    *storage.Repository
	mail          Mail
	configuration Configuration
}

// NewService supplies account-owned storage and message delivery collaborators.
func NewService(repository *storage.Repository, mail Mail, configuration Configuration) *Service {
	return &Service{repository: repository, mail: mail, configuration: configuration}
}

// NotifyWithin queues release mail for active subscribers in the upgrade transaction.
func (a *Service) NotifyWithin(ctx context.Context, transactionValue *transaction.Transaction, version string) error {
	address, err := a.configuration.PublicAddress(ctx)
	if err != nil {
		return err
	}
	queries := a.repository.Within(transactionValue)
	users, err := queries.UpdatesSubscribers(ctx)
	if err != nil {
		return err
	}
	for _, user := range users {
		token := rand.Text()
		err = queries.UpdatesSetToken(ctx, storage.UpdatesSetTokenParams{UserID: user.ID, TokenHash: digest(token)})
		if err != nil {
			return err
		}
		body := "A new version of mindflow is ready: " + version + ".\n\nOpen mindflow\t" + address + "/\nView patchnotes on GitHub\thttps://github.com/Neokil/mindflow/releases/tag/" + version + "\nUnsubscribe from product updates\t" + address + "/api/v1/auth/product-updates/unsubscribe?token=" + token
		err = a.mail.QueueWithin(ctx, transactionValue, user.ID, "release-"+version, user.Email, "mindflow "+version+" is available", body, "")
		if err != nil {
			return err
		}
	}

	return nil
}

// Enabled returns the account's explicit consent state (off by default).
func (a *Service) Enabled(ctx context.Context, userID string) (bool, error) {
	value, err := a.repository.UpdatesPreference(ctx, userID)

	return value != 0, err
}

// Set changes consent and invalidates unsubscribe links from earlier consent cycles.
func (a *Service) Set(ctx context.Context, userID string, enabled bool) error {
	transactionValue, err := a.repository.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transactionValue.Rollback() }()
	err = SetWithin(ctx, a.repository, transactionValue, userID, enabled)
	if err != nil {
		return err
	}
	if !enabled {
		err = a.mail.CancelUpdatesWithin(ctx, transactionValue, userID)
		if err != nil {
			return err
		}
	}

	return wrap("save update preference", transactionValue.Commit())
}

// SetWithin stores consent as part of account activation or initial setup.
func SetWithin(ctx context.Context, repository *storage.Repository, transactionValue *transaction.Transaction, userID string, enabled bool) error {
	value := 0
	if enabled {
		value = 1
	}

	err := repository.Within(transactionValue).UpdatesSetPreference(ctx, storage.UpdatesSetPreferenceParams{UserID: userID, Enabled: value})
	if err != nil {
		return err
	}
	if !enabled {
		return repository.Within(transactionValue).UpdatesDeleteTokens(ctx, userID)
	}

	return nil
}

// Unsubscribe consumes a purpose-limited capability without requiring a session.
func (a *Service) Unsubscribe(ctx context.Context, token string) error {
	if len(token) != unsubscribeTokenLength {
		return nil
	}
	transactionValue, err := a.repository.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transactionValue.Rollback() }()
	userID, err := a.repository.Within(transactionValue).UpdatesFindToken(ctx, digest(token))
	if storage.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	err = SetWithin(ctx, a.repository, transactionValue, userID, false)
	if err != nil {
		return err
	}
	err = a.mail.CancelUpdatesWithin(ctx, transactionValue, userID)
	if err != nil {
		return err
	}

	return wrap("unsubscribe from product updates", transactionValue.Commit())
}

// RegisterHTTP exposes authenticated consent settings and an email capability endpoint.
func (a *Service) RegisterHTTP(routes account.HTTP) {
	routes.Endpoint("GET /api/v1/auth/product-updates", true, false, func(_ http.ResponseWriter, request *http.Request) (any, error) {
		enabled, err := a.Enabled(request.Context(), routes.CurrentUser(request).ID)

		return map[string]bool{"enabled": enabled}, err
	})
	routes.Endpoint("PATCH /api/v1/auth/product-updates", true, false, func(writer http.ResponseWriter, request *http.Request) (any, error) {
		var value struct {
			Enabled bool `json:"enabled"`
		}
		err := routes.Decode(writer, request, &value)
		if err != nil {
			return nil, err
		}
		err = a.Set(request.Context(), routes.CurrentUser(request).ID, value.Enabled)

		return map[string]bool{"enabled": value.Enabled}, err
	})
	routes.Endpoint("GET /api/v1/auth/product-updates/unsubscribe", false, false, func(writer http.ResponseWriter, request *http.Request) (any, error) {
		err := a.Unsubscribe(request.Context(), strings.TrimSpace(request.URL.Query().Get("token")))
		if err != nil {
			return nil, err
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err = writer.Write([]byte(`<!doctype html><html lang="en"><meta name="viewport" content="width=device-width"><title>mindflow — unsubscribed</title><body><h1>You're unsubscribed.</h1><p>You will no longer receive product-update emails. You can subscribe again in Your profile.</p><a href="/">Open mindflow</a></body></html>`))

		return nil, wrap("write unsubscribe result", err)
	})
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))

	return hex.EncodeToString(sum[:])
}

const unsubscribeTokenLength = 26
