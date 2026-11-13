package editor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/storage"
	"time"
)

// ControlRequest is visible only to its requester and the current holder.
type ControlRequest struct {
	ID                   string `json:"id"`
	RequesterID          string `json:"requesterId"`
	AvatarVersion        string `json:"avatarVersion"`
	RequesterEmail       string `json:"requesterEmail"`
	RequesterDisplayName string `json:"requesterDisplayName"`
	Deadline             int64  `json:"deadline"`
	Status               string `json:"status"`
	Incoming             bool   `json:"incoming"`
	ServerNow            int64  `json:"serverNow"`
}

func (a *Service) resolveRequest(ctx context.Context, actor session.Principal, pageID, clientID, action, requestID string) error {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	_, err = queries.RequirePage(ctx, pageID, actor.ID, false)
	if err != nil {
		return err
	}
	request, err := queries.EditorRequest(ctx, pageID)
	if storage.IsNotFound(err) {
		if action != "" {
			return failure("lease_lost", "This control request is no longer available.")
		}

		return nil
	}
	if err != nil {
		return err
	}
	if action != "" && (request.ID != requestID || request.Status != requestPending) {
		return failure("lease_lost", "This control request has ended.")
	}
	lease, err := a.repository.LeaseWithin(ctx, transaction, pageID)
	if err != nil {
		return err
	}
	holderMatches := lease.SessionID == request.HolderSession && lease.ClientID == request.HolderClient && lease.Epoch == request.HolderEpoch
	err = authorizeDecision(request, actor, clientID, action, holderMatches)
	if err != nil {
		return err
	}
	if request.Status != requestPending {
		return nil
	}
	status := requestPending
	switch {
	case !holderMatches:
		status = requestCancelled
	case a.now().UnixMilli() >= request.Deadline:
		status = requestAccepted
	case action == controlAccept:
		status = requestAccepted
	case action == controlDeny:
		status = "denied"
	case action == controlCancel:
		status = requestCancelled
	}
	if status == requestAccepted {
		status, err = a.grantRequest(ctx, transaction, request, lease.Epoch)
		if err != nil {
			return err
		}
	}
	if status == requestPending {
		return nil
	}
	err = queries.EditorRequestStatus(ctx, storage.EditorRequestStatusParams{PageID: pageID, ID: request.ID, Status: status})
	if err != nil {
		return err
	}
	err = wrap("commit control request", transaction.Commit())
	if err == nil {
		a.publish(actor.ID, "lease", pageID)
	}

	return err
}

func (a *Service) requestControl(ctx context.Context, transaction *storage.Transaction, actor session.Principal, pageID, clientID string) error {
	queries := a.repository.Within(transaction)
	request, err := queries.EditorRequest(ctx, pageID)
	if err != nil && !storage.IsNotFound(err) {
		return err
	}
	if err == nil && request.Status == requestPending {
		if request.RequesterSession == actor.SessionID && request.RequesterClient == clientID {
			return nil
		}

		return failure("lease_lost", "Another editor is already requesting control.")
	}
	lease, err := a.repository.LeaseWithin(ctx, transaction, pageID)
	if err != nil {
		return err
	}
	token := make([]byte, requestIdentifierBytes)
	_, _ = rand.Read(token)

	return queries.EditorCreateRequest(ctx, storage.EditorCreateRequestParams{PageID: pageID, ID: hex.EncodeToString(token), RequesterID: actor.ID, RequesterEmail: actor.Email, RequesterDisplayName: actor.DisplayName, RequesterAvatarVersion: actor.AvatarVersion, RequesterSession: actor.SessionID, RequesterClient: clientID, HolderSession: lease.SessionID, HolderClient: lease.ClientID, HolderEpoch: lease.Epoch, Deadline: a.now().Add(controlWait).UnixMilli()})
}

// Absence of a request is a valid optional result.
//
//nolint:nilnil // A nil request intentionally omits the optional JSON field.
func (a *Service) requestInfo(ctx context.Context, actor session.Principal, pageID, clientID string) (*ControlRequest, error) {
	request, err := a.repository.EditorRequest(ctx, pageID)
	if storage.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	incoming := request.HolderSession == actor.SessionID && request.HolderClient == clientID
	requester := request.RequesterSession == actor.SessionID && request.RequesterClient == clientID
	if !requester && (!incoming || request.Status != requestPending) {
		return nil, nil
	}

	return &ControlRequest{ID: request.ID, RequesterEmail: request.RequesterEmail, RequesterDisplayName: request.RequesterDisplayName, RequesterID: request.RequesterID, AvatarVersion: request.RequesterAvatarVersion, Deadline: request.Deadline, Status: request.Status, Incoming: incoming, ServerNow: a.now().UnixMilli()}, nil
}

// Decide accepts or denies a holder's pending request, or lets its requester cancel.
func (a *Service) Decide(ctx context.Context, actor session.Principal, pageID string, command LeaseCommand) (LeaseResult, error) {
	err := a.resolveRequest(ctx, actor, pageID, command.ClientID, command.Action, command.RequestID)
	if err != nil {
		return LeaseResult{}, repositoryError(err)
	}

	return a.LeaseStatus(ctx, actor, pageID, command.ClientID)
}

const (
	controlWait            = 10 * time.Second
	requestIdentifierBytes = 16
)

func (a *Service) grantRequest(ctx context.Context, transaction *storage.Transaction, request storage.EditorRequest, epoch int64) (string, error) {
	access, err := a.repository.Within(transaction).RequirePage(ctx, request.PageID, request.RequesterID, false)
	if storage.IsNotFound(err) {
		return requestCancelled, nil
	}
	if err != nil {
		return "", err
	}
	if !storage.CanEdit(access.Role) {
		return requestCancelled, nil
	}
	active, err := a.sessions.ActiveWithin(ctx, transaction, request.RequesterSession, request.RequesterID)
	if err != nil {
		return "", err
	}
	if !active {
		return requestCancelled, nil
	}
	err = a.repository.SaveLeaseWithin(ctx, transaction, request.PageID, request.RequesterSession, request.RequesterClient, request.RequesterID, epoch+1, a.now().Unix()+LeaseSeconds)

	return requestAccepted, err
}

const (
	controlAccept    = "accept"
	controlDeny      = "deny"
	controlCancel    = "cancel"
	requestPending   = "pending"
	requestAccepted  = "accepted"
	requestCancelled = "cancelled"
)

func authorizeDecision(request storage.EditorRequest, actor session.Principal, clientID, action string, holderMatches bool) error {
	if action == "" {
		return nil
	}
	holder := holderMatches && actor.SessionID == request.HolderSession && clientID == request.HolderClient
	requester := actor.SessionID == request.RequesterSession && clientID == request.RequesterClient
	if (action == controlCancel && !requester) || (action != controlCancel && !holder) {
		return &storage.AccessError{}
	}

	return nil
}
