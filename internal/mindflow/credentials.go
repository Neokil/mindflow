package mindflow

import (
	"context"
	"errors"
	"mindflow/internal/account"
	"mindflow/internal/mindflow/storage"
)

type linkAuthorization struct {
	data     *storage.Database
	accounts *account.Service
}

// CanDeliver checks whether an invitation or reset link may still be delivered.
func (a linkAuthorization) CanDeliver(ctx context.Context, hash string) (bool, error) {
	valid, err := a.data.Accounts.InvitationCanDeliver(ctx, hash)
	if err != nil || valid {
		return valid, err
	}

	return a.data.Accounts.PasswordresetCanDeliver(ctx, hash)
}

// CanDeliverNotice avoids sending onboarding messages to removed or inactive accounts.
func (a linkAuthorization) CanDeliverNotice(ctx context.Context, userID string) (bool, error) {
	user, err := a.accounts.Find(ctx, userID)
	var problem *account.Error
	if errors.As(err, &problem) && problem.Code == "not_found" {
		return false, nil
	}

	return err == nil && user.Status == account.StatusActive, err
}
