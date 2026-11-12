package account

import (
	"context"
	"mindflow/internal/account/storage"
	"strings"
	"sync"
	"time"
)

// CredentialMail supplies account-linked delivery status and cancellation without table access.
type CredentialMail interface {
	CancelUserWithin(ctx context.Context, transaction *storage.Transaction, userID string) error
	DeliveryError(ctx context.Context, userID string) (string, error)
}

// UserContent removes private content in the account deletion transaction.
type UserContent interface {
	DeleteOwnerWithin(ctx context.Context, transaction *storage.Transaction, userID string) error
}

// Service owns account state and password policy.
type Service struct {
	repository *storage.Repository
	hashMu     sync.Mutex
	dummyHash  string
	mail       CredentialMail
	content    UserContent
}

// NewService supplies account mail and content lifecycle collaborators.
func NewService(repository *storage.Repository, mail CredentialMail, content UserContent) (*Service, error) {
	hash, err := hashPassword("dummy password never used")
	if err != nil {
		return nil, err
	}

	return &Service{repository: repository, mail: mail, content: content, dummyHash: hash}, nil
}

// HashPassword validates and hashes a new password before a transaction starts.
func (a *Service) HashPassword(password string) (string, error) {
	a.hashMu.Lock()
	defer a.hashMu.Unlock()

	return hashPassword(password)
}

// Find returns the public metadata for an account.
func (a *Service) Find(ctx context.Context, id string) (User, error) {
	row, err := a.repository.AccountFindUser(ctx, id)

	return User(row), repositoryError(err)
}

// FindWithin reads account state inside a caller's transaction.
func (a *Service) FindWithin(ctx context.Context, transaction *storage.Transaction, id string) (User, error) {
	row, err := a.repository.Within(transaction).AccountFindUser(ctx, id)

	return User(row), repositoryError(err)
}

// ActiveIDByEmail supports recovery without revealing nonexistent accounts to callers.
func (a *Service) ActiveIDByEmail(ctx context.Context, address string) (string, bool, error) {
	id, err := a.repository.AccountActiveUserByEmail(ctx, strings.ToLower(strings.TrimSpace(address)))
	if storage.IsNotFound(err) {
		return "", false, nil
	}

	return id, err == nil, err
}

// CreateAdministratorWithin creates the first active administrator during installation.
func (a *Service) CreateAdministratorWithin(ctx context.Context, transaction *storage.Transaction, address, hash string) error {
	_, err := a.repository.Within(transaction).AccountCreateAdministrator(ctx, storage.AccountCreateAdministratorParams{ID: newID(), Email: address, PasswordHash: hash, CreatedAt: time.Now().Unix()})

	return err
}

// CreateInvitedWithin creates an unactivated account in an invitation transaction.
func (a *Service) CreateInvitedWithin(ctx context.Context, transaction *storage.Transaction, address, role string) (User, error) {
	normalized, err := NormalizeEmail(address)
	if err != nil {
		return User{}, err
	}
	if role == "" {
		role = RoleUser
	}
	if role != RoleAdmin && role != RoleUser {
		return User{}, invalid("Invalid role.")
	}
	count, err := a.repository.Within(transaction).AccountCountEmail(ctx, normalized)
	if err != nil {
		return User{}, err
	}
	if count > 0 {
		return User{}, failure("email_exists", "That email already has an account.")
	}
	user := User{ID: newID(), Email: normalized, Role: role, Status: StatusInvited, Appearance: "system"}
	_, err = a.repository.Within(transaction).AccountCreateInvitedUser(ctx, storage.AccountCreateInvitedUserParams{ID: user.ID, Email: user.Email, Role: user.Role, CreatedAt: time.Now().Unix()})

	return user, err
}

// SetPasswordWithin changes the password, activates the account and revokes its old credentials.
func (a *Service) SetPasswordWithin(ctx context.Context, transaction *storage.Transaction, resourceID, hash string) error {
	_, err := a.repository.Within(transaction).AccountActivateAccount(ctx, storage.AccountActivateAccountParams{PasswordHash: hash, ID: resourceID})
	if err != nil {
		return err
	}

	return a.revokeCredentials(ctx, transaction, resourceID)
}

func repositoryError(err error) error {
	if storage.IsNotFound(err) {
		return notFound()
	}

	return err
}

// revokeCredentials removes account credentials and their queued delivery atomically.
func (a *Service) revokeCredentials(ctx context.Context, transaction *storage.Transaction, userID string) error {
	err := a.repository.SessionRevokeUserWithin(ctx, transaction, userID)
	if err != nil {
		return err
	}
	err = a.repository.InvitationRevokeUserWithin(ctx, transaction, userID)
	if err != nil {
		return err
	}
	err = a.repository.PasswordresetRevokeUserWithin(ctx, transaction, userID)
	if err != nil {
		return err
	}

	return a.mail.CancelUserWithin(ctx, transaction, userID)
}
