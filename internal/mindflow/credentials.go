package mindflow

import (
	"context"
	"errors"
	"mindflow/internal/account"
	accountstorage "mindflow/internal/account/storage"
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

// CanDeliverInactivity rejects notices after renewed activity or account changes.
func (a linkAuthorization) CanDeliverInactivity(ctx context.Context, userID, purpose string) (bool, error) {
	row, err := a.data.Accounts.ActivityFind(ctx, userID)
	if err != nil {
		if accountstorage.IsNotFound(err) {
			return false, nil
		}

		return false, err
	}

	return row.Status == account.StatusActive && row.Role == account.RoleUser && row.Cycle != "" && (purpose == "inactivity-warning-"+row.Cycle || purpose == "inactivity-final-"+row.Cycle), nil
}

// CanDeliverUpdate checks current consent again immediately before delivery.
func (a linkAuthorization) CanDeliverUpdate(ctx context.Context, userID string) (bool, error) {
	active, err := a.CanDeliverNotice(ctx, userID)
	if err != nil || !active {
		return false, err
	}
	enabled, err := a.data.Accounts.UpdatesPreference(ctx, userID)

	return enabled != 0, err
}

type accountContent struct{ data *storage.Database }

// DeleteOwnerWithin composes independent content cleanup without cross-domain SQL.
func (a accountContent) DeleteOwnerWithin(ctx context.Context, transaction *accountstorage.Transaction, userID string) error {
	err := a.data.Workspaces.DeleteOwnerWithin(ctx, transaction, userID)
	if err != nil {
		return err
	}

	return a.data.Feedback.DeleteOwnerWithin(ctx, transaction, userID)
}
