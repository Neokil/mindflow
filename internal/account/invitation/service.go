package invitation

import (
	"context"
	"mindflow/internal/account"
	"mindflow/internal/account/storage"
	"mindflow/internal/email"
	"time"
)

// LinkConfiguration supplies the configured public URL for credential links.
type LinkConfiguration interface {
	PublicAddress(ctx context.Context) (string, error)
}

// Service owns invite links and their account/email workflow.
type Service struct {
	repository    *storage.Repository
	accounts      *account.Service
	email         *email.Service
	configuration LinkConfiguration
}

// RedeemCommand supplies a one-time credential link and the new password.
type RedeemCommand struct {
	Token       string `json:"token"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

const tokenLifetimeSeconds = 7 * 86400

// InviteCommand requests an invited account with an administrative role choice.
type InviteCommand struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// NewService supplies explicit account, delivery and link-configuration collaborators.
func NewService(repository *storage.Repository, accounts *account.Service, delivery *email.Service, configuration LinkConfiguration) *Service {
	return &Service{repository: repository, accounts: accounts, email: delivery, configuration: configuration}
}

// Redeem consumes a invite link and changes the account password atomically.
func (a *Service) Redeem(ctx context.Context, command RedeemCommand) (Completed, error) {
	hash, err := a.accounts.HashPassword(command.Password)
	if err != nil {
		return Completed{}, err
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, wrap("begin invite redemption", err)
	}
	defer func() { _ = transaction.Rollback() }()
	token, err := a.repository.Within(transaction).InvitationFindToken(ctx, storage.InvitationFindTokenParams{TokenHash: secretDigest(command.Token), ExpiresAt: time.Now().Unix()})
	if err != nil && !storage.IsNotFound(err) {
		return Completed{}, err
	}
	if err != nil {
		return Completed{}, invalid("This link is invalid or expired. Request a new reset link or ask your administrator to resend the invitation.")
	}
	user, err := a.accounts.FindWithin(ctx, transaction, token)
	if err != nil {
		return Completed{}, err
	}
	if user.Status != "invited" {
		return Completed{}, invalid("This link is invalid or expired. Request a new reset link or ask your administrator to resend the invitation.")
	}
	err = a.accounts.SetPasswordWithin(ctx, transaction, user.ID, hash)
	if err != nil {
		return Completed{}, err
	}

	if command.DisplayName != "" {
		err = a.accounts.SetDisplayNameWithin(ctx, transaction, user.ID, command.DisplayName)
		if err != nil {
			return Completed{}, err
		}
	}

	return Completed{OK: true}, wrap("commit invite redemption", transaction.Commit())
}

// Cleanup removes expired invite links.
func (a *Service) Cleanup(ctx context.Context) error {
	return a.repository.InvitationDeleteExpiredTokens(ctx)
}

// Invite creates an unactivated account and queues its invitation in one transaction.
func (a *Service) Invite(ctx context.Context, actor account.User, command InviteCommand) (Created, error) {
	err := account.RequireAdministrator(actor)
	if err != nil {
		return Created{}, err
	}
	publicURL, err := a.configuration.PublicAddress(ctx)
	if err != nil {
		return Created{}, err
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Created{}, wrap("begin invitation", err)
	}
	defer func() { _ = transaction.Rollback() }()
	user, err := a.accounts.CreateInvitedWithin(ctx, transaction, command.Email, command.Role)
	if err != nil {
		return Created{}, err
	}
	err = a.issueWithin(ctx, transaction, user, publicURL)
	if err != nil {
		return Created{}, err
	}

	return Created{ID: user.ID}, wrap("commit invitation", transaction.Commit())
}

// ResendInvite replaces the previous unconsumed invitation.
func (a *Service) ResendInvite(ctx context.Context, actor account.User, resourceID string) (Completed, error) {
	err := account.RequireAdministrator(actor)
	if err != nil {
		return Completed{}, err
	}

	return Completed{OK: true}, a.requestForUser(ctx, resourceID)
}

func (a *Service) issueWithin(ctx context.Context, transaction *storage.Transaction, user account.User, publicURL string) error {
	_, err := a.repository.Within(transaction).InvitationDeleteUserTokens(ctx, user.ID)
	if err != nil {
		return err
	}
	token := newID()
	_, err = a.repository.Within(transaction).InvitationCreateToken(ctx, storage.InvitationCreateTokenParams{ID: newID(), UserID: user.ID, TokenHash: secretDigest(token), ExpiresAt: time.Now().Unix() + tokenLifetimeSeconds})
	if err != nil {
		return err
	}
	subject := "You're invited to mindflow"
	body := subject + "\n\n" + publicURL + "/#accept-invite/" + token + "\n\nIf you weren't expecting this email, you can ignore it."

	return a.email.QueueWithin(ctx, transaction, user.ID, "invite", user.Email, subject, body, secretDigest(token))
}

func (a *Service) requestForUser(ctx context.Context, resourceID string) error {
	publicURL, err := a.configuration.PublicAddress(ctx)
	if err != nil {
		return err
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return wrap("begin invite issuance", err)
	}
	defer func() { _ = transaction.Rollback() }()
	user, err := a.accounts.FindWithin(ctx, transaction, resourceID)
	if err != nil {
		return err
	}
	if user.Status != "invited" {
		return invalid("Account status does not allow this link.")
	}
	err = a.issueWithin(ctx, transaction, user, publicURL)
	if err != nil {
		return err
	}

	return wrap("commit invite issuance", transaction.Commit())
}
