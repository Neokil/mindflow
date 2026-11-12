package account

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// VerifyCredentials checks a password without creating a session.
func (a *Service) VerifyCredentials(ctx context.Context, address, password string) (_ User, err error) {
	defer func() { err = repositoryError(err) }()

	email := strings.ToLower(strings.TrimSpace(address))
	var user User
	var hash string
	userByEmailRow, err := New(a.db).UserByEmail(ctx, email)
	user.ID = userByEmailRow.ID
	user.Email = userByEmailRow.Email
	user.Role = userByEmailRow.Role
	user.Status = userByEmailRow.Status
	user.Appearance = userByEmailRow.Appearance
	hash = userByEmailRow.PasswordHash

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return User{}, err
	}
	if err != nil || hash == "" {
		hash = a.dummyHash
	}
	a.hashMu.Lock()
	valid := verifyPassword(hash, password)
	a.hashMu.Unlock()
	if err != nil || !valid || user.Status != StatusActive {
		return User{}, failure("invalid_login", "Email or password is incorrect, or the account is unavailable.")
	}

	return user, nil
}
