package account

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"mindflow/internal/account/storage"
	"net/mail"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// User contains public account metadata without credentials.
type User struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	Appearance    string `json:"appearance"`
	AvatarVersion string `json:"avatarVersion"`
	DisplayName   string `json:"displayName"`
}

// NormalizeEmail validates and normalizes a plain email address.
func NormalizeEmail(settings string) (string, error) {
	settings = strings.ToLower(strings.TrimSpace(settings))
	addr, err := mail.ParseAddress(settings)
	if err != nil || addr.Address != settings || len(settings) > 254 {
		return "", invalid("Enter a valid email address.")
	}

	return settings, nil
}

// ValidatePassword checks the supported password length.
func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < 8 || utf8.RuneCountInString(password) > 128 {
		return invalid("Use a password between 8 and 128 characters.")
	}

	return nil
}

// CheckLastAdministrator prevents removing the last active administrator.
func CheckLastAdministrator(role, status, newRole, newStatus string, activeAdministrators int) error {
	if role == RoleAdmin && status == StatusActive && (newRole != RoleAdmin || newStatus != StatusActive) && activeAdministrators <= 1 {
		return invalid("Keep at least one active administrator.")
	}

	return nil
}

// ChangeAccount validates role and status transitions.
func ChangeAccount(role, status, newRole, newStatus string) (string, string, error) {
	if newRole == "" {
		newRole = role
	}
	if newStatus == "" {
		newStatus = status
	}
	if newRole != RoleUser && newRole != RoleAdmin {
		return "", "", invalid("Invalid role.")
	}
	if newStatus != StatusActive && newStatus != StatusInactive && newStatus != StatusInvited {
		return "", "", invalid("Invalid status.")
	}
	if status == StatusInvited && newStatus != status || status != StatusInvited && newStatus == StatusInvited {
		return "", "", invalid("Invitation activation requires the email link.")
	}

	return newRole, newStatus, nil
}

// UpdateUserCommand requests a role or activation-status change.
type UpdateUserCommand struct {
	Role   string `json:"role"`
	Status string `json:"status"`
}

// DeleteUserCommand confirms deletion of an account and its cascading content.
type DeleteUserCommand struct {
	Confirm bool `json:"confirm"`
}

// UserDelivery pairs account metadata with delivery status.
type UserDelivery struct {
	User      User   `json:"user"`
	MailError string `json:"mailError"`
}

func hashPassword(password string) (string, error) {
	err := ValidatePassword(password)
	if err != nil {
		return "", err
	}
	salt := make([]byte, passwordSaltBytes)
	_, readErr := rand.Read(salt)
	if readErr != nil {
		return "", wrap("account.hashPassword", readErr)
	}
	h := argon2.IDKey([]byte(password), salt, passwordIterations, passwordMemoryKiB, 1, passwordKeyBytes)

	return fmt.Sprintf("$argon2id$v=19$m=19456,t=2,p=1$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(h)), nil
}

func verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[3] != "m=19456,t=2,p=1" || len(password) > 1024 {
		return false
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[4])
	if e != nil {
		return false
	}
	expected, e := base64.RawStdEncoding.DecodeString(parts[5])
	if e != nil {
		return false
	}
	h := argon2.IDKey([]byte(password), salt, passwordIterations, passwordMemoryKiB, 1, passwordKeyBytes)

	return subtle.ConstantTimeCompare(h, expected) == 1
}

func (a *Service) guardLastAdmin(ctx context.Context, transaction *storage.Transaction, id, newRole, newStatus string) error {
	var role, status string
	userRoleStatusRow, err := a.repository.Within(transaction).AccountUserRoleStatus(ctx, id)
	role = userRoleStatusRow.Role
	status = userRoleStatusRow.Status
	if err != nil {
		return err
	}
	var count int
	countAdministratorsRow, err := a.repository.Within(transaction).AccountCountAdministrators(ctx)
	count = int(countAdministratorsRow)
	if err != nil {
		return err
	}

	return CheckLastAdministrator(role, status, newRole, newStatus, count)
}

// ListUsers returns account and mail-delivery status to administrators.
func (a *Service) ListUsers(ctx context.Context, actor User) (_ []UserDelivery, err error) {
	requireAdministratorErr := RequireAdministrator(actor)
	if requireAdministratorErr != nil {
		return nil, requireAdministratorErr
	}
	defer func() { err = repositoryError(err) }()

	rows, err := a.repository.AccountListUsers(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]UserDelivery, 0, len(rows))
	for _, row := range rows {
		mailError, deliveryErr := a.mail.DeliveryError(ctx, row.ID)
		if deliveryErr != nil {
			return nil, deliveryErr
		}
		users = append(users, UserDelivery{User: User{ID: row.ID, Email: row.Email, Role: row.Role, Status: row.Status, Appearance: row.Appearance, AvatarVersion: row.AvatarVersion, DisplayName: row.DisplayName}, MailError: mailError})
	}

	return users, nil
}

// UpdateUser updates an account while preserving the last active administrator.
func (a *Service) UpdateUser(ctx context.Context, actor User, resourceID string, command UpdateUserCommand) (_ Completed, err error) {
	requireAdministratorErr5 := RequireAdministrator(actor)
	if requireAdministratorErr5 != nil {
		return Completed{}, requireAdministratorErr5
	}
	defer func() { err = repositoryError(err) }()

	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, wrap("account.UpdateUser", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var role, status string
	userRoleStatusRow, err := a.repository.Within(transaction).AccountUserRoleStatus(ctx, resourceID)
	role = userRoleStatusRow.Role
	status = userRoleStatusRow.Status
	if err != nil {
		return Completed{}, err
	}
	command.Role, command.Status, err = ChangeAccount(role, status, command.Role, command.Status)
	if err != nil {
		return Completed{}, err
	}
	err = a.guardLastAdmin(ctx, transaction, resourceID, command.Role, command.Status)
	if err != nil {
		return Completed{}, err
	}
	_, err = a.repository.Within(transaction).AccountUpdateUser(ctx, storage.AccountUpdateUserParams{Role: command.Role, Status: command.Status, ID: resourceID})
	if err != nil {
		return Completed{}, err
	}
	if command.Status != StatusActive {
		err = a.revokeCredentials(ctx, transaction, resourceID)
		if err != nil {
			return Completed{}, err
		}
	}

	return Completed{OK: true}, wrap("account.UpdateUser", transaction.Commit())
}

// DeleteUser permanently deletes an account and its cascading content.
func (a *Service) DeleteUser(ctx context.Context, actor User, resourceID string, command DeleteUserCommand) (_ Completed, err error) {
	requireAdministratorErr6 := RequireAdministrator(actor)
	if requireAdministratorErr6 != nil {
		return Completed{}, requireAdministratorErr6
	}
	defer func() { err = repositoryError(err) }()

	if !command.Confirm {
		return Completed{}, invalid("Confirm permanent user deletion.")
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, wrap("account.DeleteUser", err)
	}
	defer func() { _ = transaction.Rollback() }()
	err = a.guardLastAdmin(ctx, transaction, resourceID, "", "")
	if err != nil {
		return Completed{}, err
	}
	err = a.content.DeleteOwnerWithin(ctx, transaction, resourceID)
	if err != nil {
		return Completed{}, err
	}
	err = a.mail.CancelUserWithin(ctx, transaction, resourceID)
	if err != nil {
		return Completed{}, err
	}
	_, err = a.repository.Within(transaction).AccountDeleteUser(ctx, resourceID)
	if err != nil {
		return Completed{}, err
	}

	return Completed{OK: true}, wrap("account.DeleteUser", transaction.Commit())
}

// SetAppearance stores a supported account appearance preference.
func (a *Service) SetAppearance(ctx context.Context, actor User, value string) error {
	if value != "system" && value != "light" && value != "dark" {
		return invalid("Invalid appearance.")
	}
	_, err := a.repository.AccountSetAppearance(ctx, storage.AccountSetAppearanceParams{Appearance: value, ID: actor.ID})

	return err
}

// RequireAdministrator checks administrative account privileges.
func RequireAdministrator(actor User) error {
	if actor.Role != RoleAdmin || actor.Status != StatusActive {
		return failure("forbidden", "Administrator access required.")
	}

	return nil
}
