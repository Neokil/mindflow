package passwordreset

import (
	"context"
	"database/sql"
	"errors"
	"mindflow/internal/account"
	"mindflow/internal/email"
	"time"
)

// LinkConfiguration supplies the configured public URL for credential links.
type LinkConfiguration interface {
	PublicAddress(ctx context.Context) (string, error)
}

// Service owns reset links and their account/email workflow.
type Service struct {
	db            *sql.DB
	accounts      *account.Service
	email         *email.Service
	configuration LinkConfiguration
}

// RedeemCommand supplies a one-time credential link and the new password.
type RedeemCommand struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

const tokenLifetimeSeconds = 3600

// ForgotCommand requests recovery without exposing whether an account exists.
type ForgotCommand struct {
	Email string `json:"email"`
}

// NewService supplies explicit account, delivery and link-configuration collaborators.
func NewService(connection *sql.DB, accounts *account.Service, delivery *email.Service, configuration LinkConfiguration) *Service {
	return &Service{db: connection, accounts: accounts, email: delivery, configuration: configuration}
}

// Redeem consumes a reset link and changes the account password atomically.
func (a *Service) Redeem(ctx context.Context, command RedeemCommand) (Completed, error) {
	hash, err := a.accounts.HashPassword(command.Password)
	if err != nil {
		return Completed{}, err
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Completed{}, wrap("begin reset redemption", err)
	}
	defer func() { _ = transaction.Rollback() }()
	token, err := New(transaction).FindToken(ctx, FindTokenParams{TokenHash: secretDigest(command.Token), ExpiresAt: time.Now().Unix()})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Completed{}, err
	}
	if err != nil {
		return Completed{}, invalid("This link is invalid or expired. Request a new reset link or ask your administrator to resend the invitation.")
	}
	user, err := a.accounts.FindWithin(ctx, transaction, token)
	if err != nil {
		return Completed{}, err
	}
	if user.Status != "active" {
		return Completed{}, invalid("This link is invalid or expired. Request a new reset link or ask your administrator to resend the invitation.")
	}
	err = a.accounts.SetPasswordWithin(ctx, transaction, user.ID, hash)
	if err != nil {
		return Completed{}, err
	}

	return Completed{OK: true}, wrap("commit reset redemption", transaction.Commit())
}

// RevokeUserWithin removes this account's reset links during a credential change.
func RevokeUserWithin(ctx context.Context, transaction *sql.Tx, userID string) error {
	_, err := New(transaction).DeleteUserTokens(ctx, userID)

	return err
}

// RevokeAllWithin removes reset links after a snapshot restore.
func RevokeAllWithin(ctx context.Context, transaction *sql.Tx) error {
	return New(transaction).RevokeTokens(ctx)
}

// CanDeliver authorizes delivery of an unexpired, unused reset link.
func CanDeliver(ctx context.Context, connection *sql.DB, hash string) (bool, error) {
	count, err := New(connection).CountValidTokens(ctx, CountValidTokensParams{TokenHash: hash, ExpiresAt: time.Now().Unix()})

	return count > 0, err
}

// Cleanup removes expired reset links.
func (a *Service) Cleanup(ctx context.Context) error { return New(a.db).DeleteExpiredTokens(ctx) }

// Forgot queues recovery without exposing whether an active account exists.
func (a *Service) Forgot(ctx context.Context, command ForgotCommand) (Completed, error) {
	resourceID, exists, err := a.accounts.ActiveIDByEmail(ctx, command.Email)
	if err != nil {
		return Completed{}, err
	}
	if !exists {
		return Completed{OK: true}, nil
	}
	err = a.requestForUser(ctx, resourceID)

	return Completed{OK: true}, err
}

// AdminReset sends a password reset after checking administrative privileges.
func (a *Service) AdminReset(ctx context.Context, actor account.User, resourceID string) (Completed, error) {
	err := account.RequireAdministrator(actor)
	if err != nil {
		return Completed{}, err
	}

	return Completed{OK: true}, a.requestForUser(ctx, resourceID)
}

func (a *Service) issueWithin(ctx context.Context, transaction *sql.Tx, user account.User, publicURL string) error {
	_, err := New(transaction).DeleteUserTokens(ctx, user.ID)
	if err != nil {
		return err
	}
	token := newID()
	_, err = New(transaction).CreateToken(ctx, CreateTokenParams{ID: newID(), UserID: user.ID, TokenHash: secretDigest(token), ExpiresAt: time.Now().Unix() + tokenLifetimeSeconds})
	if err != nil {
		return err
	}
	subject := "Reset your mindflow password"
	body := subject + "\n\n" + publicURL + "/#reset-password/" + token + "\n\nIf you weren't expecting this email, you can ignore it."

	return a.email.QueueWithin(ctx, transaction, user.ID, "reset", user.Email, subject, body, secretDigest(token))
}

func (a *Service) requestForUser(ctx context.Context, resourceID string) error {
	publicURL, err := a.configuration.PublicAddress(ctx)
	if err != nil {
		return err
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return wrap("begin reset issuance", err)
	}
	defer func() { _ = transaction.Rollback() }()
	user, err := a.accounts.FindWithin(ctx, transaction, resourceID)
	if err != nil {
		return err
	}
	if user.Status != "active" {
		return invalid("Account status does not allow this link.")
	}
	err = a.issueWithin(ctx, transaction, user, publicURL)
	if err != nil {
		return err
	}

	return wrap("commit reset issuance", transaction.Commit())
}
