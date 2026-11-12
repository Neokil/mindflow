package mindflow

import (
	"context"
	"database/sql"
	"mindflow/internal/editor"
	"mindflow/internal/email"
	"mindflow/internal/invitation"
	"mindflow/internal/passwordreset"
	"mindflow/internal/session"
)

// credentialRevoker coordinates domain-owned credential changes in one transaction.
type credentialRevoker struct{}

// RevokeUser revokes sessions, account links and queued credential mail in one transaction.
func (credentialRevoker) RevokeUser(ctx context.Context, transaction *sql.Tx, userID string) error {
	err := session.RevokeUserWithin(ctx, transaction, userID)
	if err != nil {
		return err
	}
	err = invitation.RevokeUserWithin(ctx, transaction, userID)
	if err != nil {
		return err
	}
	err = passwordreset.RevokeUserWithin(ctx, transaction, userID)
	if err != nil {
		return err
	}

	return email.CancelUserWithin(ctx, transaction, userID)
}

type linkAuthorization struct{ connection *sql.DB }

// CanDeliver checks whether an invitation or reset link may still be delivered.
func (a linkAuthorization) CanDeliver(ctx context.Context, hash string) (bool, error) {
	valid, err := invitation.CanDeliver(ctx, a.connection, hash)
	if err != nil || valid {
		return valid, err
	}

	return passwordreset.CanDeliver(ctx, a.connection, hash)
}

// RevokeCredentials coordinates credential revocation on a stopped restored instance.
func RevokeCredentials(ctx context.Context, connection *sql.DB) error {
	transaction, err := connection.BeginTx(ctx, nil)
	if err != nil {
		return wrap("begin restore revocation", err)
	}
	defer func() { _ = transaction.Rollback() }()
	err = session.RevokeAllWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = invitation.RevokeAllWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = passwordreset.RevokeAllWithin(ctx, transaction)
	if err != nil {
		return err
	}
	err = email.CancelAllWithin(ctx, transaction)
	if err != nil {
		return err
	}

	err = editor.RevokeWithin(ctx, transaction)
	if err != nil {
		return err
	}

	return wrap("commit restore revocation", transaction.Commit())
}
