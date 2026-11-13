package account

import (
	"context"
	"mindflow/internal/account/storage"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxDisplayNameRunes = 80

// NormalizeDisplayName validates a bounded, single-line public name.
func NormalizeDisplayName(value string) (string, error) {
	if !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", invalid("Use a display name without control characters.")
	}
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > maxDisplayNameRunes {
		return "", invalid("Use a display name between 1 and 80 characters.")
	}

	return value, nil
}

func defaultDisplayName(address string) string {
	local, _, _ := strings.Cut(address, "@")
	value := []rune(strings.TrimSpace(local))
	if len(value) > maxDisplayNameRunes {
		value = value[:maxDisplayNameRunes]
	}
	if len(value) == 0 {
		return "User"
	}

	return string(value)
}

// ProfileCommand updates only the caller's public display name.
type ProfileCommand struct {
	DisplayName string `json:"displayName"`
}

// DirectoryUser is the public identity exposed to other registered users.
type DirectoryUser struct {
	ID            string `json:"id"`
	DisplayName   string `json:"displayName"`
	AvatarVersion string `json:"avatarVersion"`
}

// UpdateProfile updates the authenticated caller without accepting an account ID.
func (a *Service) UpdateProfile(ctx context.Context, actor User, command ProfileCommand) (User, error) {
	name, err := NormalizeDisplayName(command.DisplayName)
	if err != nil {
		return User{}, err
	}
	err = a.repository.AccountSetDisplayName(ctx, storage.AccountSetDisplayNameParams{DisplayName: name, ID: actor.ID})
	if err != nil {
		return User{}, repositoryError(err)
	}

	return a.Find(ctx, actor.ID)
}

// SetDisplayNameWithin updates a profile as part of account activation.
func (a *Service) SetDisplayNameWithin(ctx context.Context, transaction *storage.Transaction, resourceID, value string) error {
	name, err := NormalizeDisplayName(value)
	if err != nil {
		return err
	}

	return a.repository.Within(transaction).AccountSetDisplayName(ctx, storage.AccountSetDisplayNameParams{DisplayName: name, ID: resourceID})
}

// Directory lists active profiles without email addresses, account roles or credentials.
func (a *Service) Directory(ctx context.Context) ([]DirectoryUser, error) {
	rows, err := a.repository.AccountDirectory(ctx)
	if err != nil {
		return nil, repositoryError(err)
	}
	users := make([]DirectoryUser, 0, len(rows))
	for _, row := range rows {
		users = append(users, DirectoryUser(row))
	}

	return users, nil
}
