package mindflow

import (
	"context"
	"mindflow/internal/mindflow/storage"
)

type linkAuthorization struct{ data *storage.Database }

// CanDeliver checks whether an invitation or reset link may still be delivered.
func (a linkAuthorization) CanDeliver(ctx context.Context, hash string) (bool, error) {
	valid, err := a.data.Accounts.InvitationCanDeliver(ctx, hash)
	if err != nil || valid {
		return valid, err
	}

	return a.data.Accounts.PasswordresetCanDeliver(ctx, hash)
}
