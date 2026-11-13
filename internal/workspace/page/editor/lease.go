package editor

import (
	"context"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/storage"
)

// Lease applies editor-control policy in an ownership-checked transaction.
func (a *Service) Lease(ctx context.Context, actor session.Principal, resourceID string, command LeaseCommand) (_ LeaseResult, err error) {
	defer func() { err = repositoryError(err) }()
	if len(command.ClientID) < 8 || len(command.ClientID) > maxIdentifierLength {
		return LeaseResult{}, invalid("Invalid editor identity.")
	}
	if !validAction(command.Action) {
		return LeaseResult{}, invalid("Invalid editor action.")
	}
	if command.Action == controlAccept || command.Action == controlDeny || command.Action == controlCancel {
		return a.Decide(ctx, actor, resourceID, command)
	}
	err = a.resolveRequest(ctx, actor, resourceID, command.ClientID, "", "")
	if err != nil {
		return LeaseResult{}, err
	}
	caller := actor
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return LeaseResult{}, wrap("canvas_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	access, err := a.repository.Within(transaction).RequirePage(ctx, resourceID, caller.ID, false)
	if err != nil {
		return LeaseResult{}, err
	}
	page, err := a.repository.OwnedPageWithin(ctx, transaction, resourceID, caller.ID)
	if err != nil {
		return LeaseResult{}, err
	}
	if !storage.CanEdit(access.Role) {
		if command.Action == leaseTakeover || command.Action == leaseRenew {
			return LeaseResult{}, &storage.AccessError{}
		}

		return LeaseResult{Permission: access.Role, Revision: page.Revision}, nil
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
	if command.Action == leaseTakeover && expires > a.now().Unix() && (session != caller.SessionID || client != command.ClientID) {
		err = a.requestControl(ctx, transaction, actor, resourceID, command.ClientID)
		if err != nil {
			return LeaseResult{}, err
		}
		err = wrap("commit editor lease", transaction.Commit())
		if err != nil {
			return LeaseResult{}, err
		}
		a.publish(actor.ID, "lease", resourceID)

		return a.LeaseStatus(ctx, actor, resourceID, command.ClientID)
	}
	state, changed, err := (LeaseState{SessionID: session, ClientID: client, Epoch: epoch, ExpiresAt: expires}).Advance(caller.SessionID, command, a.now().Unix())
	if err != nil {
		return LeaseResult{}, err
	}
	epoch, expires, client = state.Epoch, state.ExpiresAt, state.ClientID
	session = state.SessionID
	if changed {
		err = a.repository.SaveLeaseWithin(ctx, transaction, page.ID, session, client, caller.ID, epoch, expires)
		if err != nil {
			return LeaseResult{}, err
		}
	}
	err = wrap("commit editor lease", transaction.Commit())
	if err != nil {
		return LeaseResult{}, wrap("canvas_repository.go", err)
	}
	if changed && command.Action != leaseRenew {
		a.publish(caller.ID, "lease", page.ID)
	}

	return LeaseResult{Permission: access.Role, Editable: session == caller.SessionID && client == command.ClientID && expires > a.now().Unix(), Epoch: epoch, ExpiresAt: expires, Revision: page.Revision}, nil
}

// LeaseStatus reports editor control and revision for an owned page.
func (a *Service) LeaseStatus(ctx context.Context, actor session.Principal, resourceID string, clientID string) (_ LeaseResult, err error) {
	defer func() { err = repositoryError(err) }()

	err = a.resolveRequest(ctx, actor, resourceID, clientID, "", "")
	if err != nil {
		return LeaseResult{}, err
	}

	access, err := a.repository.RequirePage(ctx, resourceID, actor.ID, false)
	if err != nil {
		return LeaseResult{}, err
	}
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

	request, err := a.requestInfo(ctx, actor, resourceID, clientID)
	if err != nil {
		return LeaseResult{}, err
	}

	return LeaseResult{Request: request, Permission: access.Role, Editable: storage.CanEdit(access.Role) && session == actor.SessionID && client == clientID && expires > a.now().Unix(), Epoch: epoch, ExpiresAt: expires, Revision: rev}, nil
}

func validAction(action string) bool {
	switch action {
	case "acquire", leaseTakeover, leaseRenew, "release", controlAccept, controlDeny, controlCancel:
		return true
	default:
		return false
	}
}
