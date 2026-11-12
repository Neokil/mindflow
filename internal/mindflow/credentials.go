package mindflow

import (
	"context"
	accountstorage "mindflow/internal/account/storage"
	"mindflow/internal/mindflow/storage"
)

// credentialRevoker coordinates domain-owned credential changes in one transaction.
type credentialRevoker struct{ data *storage.Database }

// RevokeUser revokes sessions, account links and queued credential mail in one transaction.
func (a credentialRevoker) RevokeUser(ctx context.Context, transaction *accountstorage.Transaction, userID string) error {
	err := a.data.Sessions.RevokeUserWithin(ctx, transaction, userID)
	if err != nil {
		return err
	}
	err = a.data.Invitations.RevokeUserWithin(ctx, transaction, userID)
	if err != nil {
		return err
	}
	err = a.data.PasswordResets.RevokeUserWithin(ctx, transaction, userID)
	if err != nil {
		return err
	}

	return a.data.Email.CancelUserWithin(ctx, transaction, userID)
}

type linkAuthorization struct{ data *storage.Database }

// CanDeliver checks whether an invitation or reset link may still be delivered.
func (a linkAuthorization) CanDeliver(ctx context.Context, hash string) (bool, error) {
	valid, err := a.data.Invitations.CanDeliver(ctx, hash)
	if err != nil || valid {
		return valid, err
	}

	return a.data.PasswordResets.CanDeliver(ctx, hash)
}
