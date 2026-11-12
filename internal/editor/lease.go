package editor

import (
	"context"
	"database/sql"
	"errors"
	"mindflow/internal/session"
	"time"
)

// Lease applies editor-control policy in an ownership-checked transaction.
func (a *Service) Lease(ctx context.Context, actor session.Principal, resourceID string, command LeaseCommand) (_ LeaseResult, err error) {
	defer func() { err = repositoryError(err) }()
	if len(command.ClientID) < 8 || len(command.ClientID) > maxIdentifierLength {
		return LeaseResult{}, invalid("Invalid editor identity.")
	}
	if command.Action != "acquire" && command.Action != "takeover" && command.Action != leaseRenew && command.Action != "release" {
		return LeaseResult{}, invalid("Invalid editor action.")
	}
	caller := actor
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return LeaseResult{}, wrap("canvas_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := loadOwnedPage(ctx, transaction, resourceID, caller.ID)
	if err != nil {
		return LeaseResult{}, err
	}
	_, err = New(transaction).EnsureLease(ctx, page.ID)
	if err != nil {
		return LeaseResult{}, err
	}
	var session sql.NullString
	var client string
	var epoch, expires int64
	pageLeaseRow, err := New(transaction).PageLease(ctx, page.ID)
	session = pageLeaseRow.SessionID
	client = pageLeaseRow.ClientID
	epoch = pageLeaseRow.Epoch
	expires = pageLeaseRow.ExpiresAt
	if err != nil {
		return LeaseResult{}, err
	}
	state, changed, err := (LeaseState{SessionID: session.String, ClientID: client, Epoch: epoch, ExpiresAt: expires}).Advance(caller.SessionID, command, time.Now().Unix())
	if err != nil {
		return LeaseResult{}, err
	}
	epoch, expires, client = state.Epoch, state.ExpiresAt, state.ClientID
	session = sql.NullString{String: state.SessionID, Valid: state.SessionID != ""}
	if changed {
		_, err = New(transaction).UpdateLease(ctx, UpdateLeaseParams{SessionID: session, ClientID: client, Epoch: epoch, ExpiresAt: expires, PageID: page.ID})
		if err != nil {
			return LeaseResult{}, err
		}
	}
	err = transaction.Commit()
	if err != nil {
		return LeaseResult{}, wrap("canvas_repository.go", err)
	}
	if changed && command.Action != leaseRenew {
		a.publish(caller.ID, "lease", page.ID)
	}

	return LeaseResult{Editable: session.String == caller.SessionID && client == command.ClientID && expires > time.Now().Unix(), Epoch: epoch, ExpiresAt: expires, Revision: page.Revision}, nil
}

// LeaseStatus reports editor control and revision for an owned page.
func (a *Service) LeaseStatus(ctx context.Context, actor session.Principal, resourceID string, clientID string) (_ LeaseResult, err error) {
	defer func() { err = repositoryError(err) }()

	var rev int
	ownedPageRevisionRow, err := New(a.db).OwnedPageRevision(ctx, OwnedPageRevisionParams{ID: resourceID, OwnerID: actor.ID})
	rev = ownedPageRevisionRow
	if err != nil {
		return LeaseResult{}, err
	}
	var session sql.NullString
	var client string
	var epoch, expires int64
	pageLeaseRow, err := New(a.db).PageLease(ctx, resourceID)
	session = pageLeaseRow.SessionID
	client = pageLeaseRow.ClientID
	epoch = pageLeaseRow.Epoch
	expires = pageLeaseRow.ExpiresAt

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return LeaseResult{}, err
	}

	return LeaseResult{Editable: session.String == actor.SessionID && client == clientID && expires > time.Now().Unix(), Epoch: epoch, ExpiresAt: expires, Revision: rev}, nil
}
