package account

import (
	"context"
	"mindflow/internal/account/storage"
	"strings"
	"sync"
	"time"
)

// CredentialRevoker removes sessions, account links and their queued messages atomically.
type CredentialRevoker interface {
	RevokeUser(ctx context.Context, transaction *storage.Transaction, userID string) error
}

// Service owns account state and password policy.
type Service struct {
	repository *storage.Repository
	hashMu     sync.Mutex
	dummyHash  string
	revoker    CredentialRevoker
}

// NewService supplies the collaborator used after account and credential changes.
func NewService(repository *storage.Repository, revoker CredentialRevoker) (*Service, error) {
	hash, err := hashPassword("dummy password never used")
	if err != nil {
		return nil, err
	}

	return &Service{repository: repository, revoker: revoker, dummyHash: hash}, nil
}

// HashPassword validates and hashes a new password before a transaction starts.
func (a *Service) HashPassword(password string) (string, error) {
	a.hashMu.Lock()
	defer a.hashMu.Unlock()

	return hashPassword(password)
}

// Find returns the public metadata for an account.
func (a *Service) Find(ctx context.Context, id string) (User, error) {
	row, err := a.repository.FindUser(ctx, id)

	return User(row), repositoryError(err)
}

// FindWithin reads account state inside a caller's transaction.
func (a *Service) FindWithin(ctx context.Context, transaction *storage.Transaction, id string) (User, error) {
	row, err := a.repository.Within(transaction).FindUser(ctx, id)

	return User(row), repositoryError(err)
}

// ActiveIDByEmail supports recovery without revealing nonexistent accounts to callers.
func (a *Service) ActiveIDByEmail(ctx context.Context, address string) (string, bool, error) {
	id, err := a.repository.ActiveUserByEmail(ctx, strings.ToLower(strings.TrimSpace(address)))
	if storage.IsNotFound(err) {
		return "", false, nil
	}

	return id, err == nil, err
}

// CreateAdministratorWithin creates the first active administrator during installation.
func (a *Service) CreateAdministratorWithin(ctx context.Context, transaction *storage.Transaction, address, hash string) error {
	_, err := a.repository.Within(transaction).CreateAdministrator(ctx, storage.CreateAdministratorParams{ID: newID(), Email: address, PasswordHash: hash, CreatedAt: time.Now().Unix()})

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
	count, err := a.repository.Within(transaction).CountEmail(ctx, normalized)
	if err != nil {
		return User{}, err
	}
	if count > 0 {
		return User{}, failure("email_exists", "That email already has an account.")
	}
	user := User{ID: newID(), Email: normalized, Role: role, Status: StatusInvited, Appearance: "system"}
	_, err = a.repository.Within(transaction).CreateInvitedUser(ctx, storage.CreateInvitedUserParams{ID: user.ID, Email: user.Email, Role: user.Role, CreatedAt: time.Now().Unix()})

	return user, err
}

// SetPasswordWithin changes the password, activates the account and revokes its old credentials.
func (a *Service) SetPasswordWithin(ctx context.Context, transaction *storage.Transaction, resourceID, hash string) error {
	_, err := a.repository.Within(transaction).ActivateAccount(ctx, storage.ActivateAccountParams{PasswordHash: hash, ID: resourceID})
	if err != nil {
		return err
	}

	return a.revoker.RevokeUser(ctx, transaction, resourceID)
}

func repositoryError(err error) error {
	if storage.IsNotFound(err) {
		return notFound()
	}

	return err
}
