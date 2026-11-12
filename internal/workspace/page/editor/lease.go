package editor

import (
	"context"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/storage"
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
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return LeaseResult{}, wrap("canvas_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := a.repository.OwnedPageWithin(ctx, transaction, resourceID, caller.ID)
	if err != nil {
		return LeaseResult{}, err
	}
	_, err = a.repository.Within(transaction).EditorEnsureLease(ctx, page.ID)
	if err != nil {
		return LeaseResult{}, err
	}
	var session string
	var client string
	var epoch, expires int64
	pageLeaseRow, err := a.repository.LeaseWithin(ctx, transaction, page.ID)
	session = pageLeaseRow.SessionID
	client = pageLeaseRow.ClientID
	epoch = pageLeaseRow.Epoch
	expires = pageLeaseRow.ExpiresAt
	if err != nil {
		return LeaseResult{}, err
	}
	state, changed, err := (LeaseState{SessionID: session, ClientID: client, Epoch: epoch, ExpiresAt: expires}).Advance(caller.SessionID, command, time.Now().Unix())
	if err != nil {
		return LeaseResult{}, err
	}
	epoch, expires, client = state.Epoch, state.ExpiresAt, state.ClientID
	session = state.SessionID
	if changed {
		err = a.repository.SaveLeaseWithin(ctx, transaction, page.ID, session, client, epoch, expires)
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

	return LeaseResult{Editable: session == caller.SessionID && client == command.ClientID && expires > time.Now().Unix(), Epoch: epoch, ExpiresAt: expires, Revision: page.Revision}, nil
}

// LeaseStatus reports editor control and revision for an owned page.
func (a *Service) LeaseStatus(ctx context.Context, actor session.Principal, resourceID string, clientID string) (_ LeaseResult, err error) {
	defer func() { err = repositoryError(err) }()

	var rev int
	ownedPageRevisionRow, err := a.repository.EditorOwnedPageRevision(ctx, storage.EditorOwnedPageRevisionParams{ID: resourceID, OwnerID: actor.ID})
	rev = ownedPageRevisionRow
	if err != nil {
		return LeaseResult{}, err
	}
	var session string
	var client string
	var epoch, expires int64
	pageLeaseRow, err := a.repository.Lease(ctx, resourceID)
	session = pageLeaseRow.SessionID
	client = pageLeaseRow.ClientID
	epoch = pageLeaseRow.Epoch
	expires = pageLeaseRow.ExpiresAt

	if err != nil && !storage.IsNotFound(err) {
		return LeaseResult{}, err
	}

	return LeaseResult{Editable: session == actor.SessionID && client == clientID && expires > time.Now().Unix(), Epoch: epoch, ExpiresAt: expires, Revision: rev}, nil
}
