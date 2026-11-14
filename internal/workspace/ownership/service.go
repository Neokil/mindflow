// Package ownership owns explicit workspace ownership transfers.
package ownership

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mindflow/internal/account"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/storage"
	"time"
)

// Accounts resolves active recipient identities through the account boundary.
type Accounts interface {
	FindWithin(ctx context.Context, transaction *storage.Transaction, resourceID string) (account.User, error)
}

// Service owns authorization, content independence and former-owner access.
type Service struct {
	repository *storage.Repository
	accounts   Accounts
	publish    func(string, string, string)
}

// NewService composes transfer collaborators.
func NewService(repository *storage.Repository, accounts Accounts, publish func(string, string, string)) *Service {
	return &Service{repository: repository, accounts: accounts, publish: publish}
}

// Command explicitly confirms the recipient and current workspace metadata revision.
type Command struct {
	UserID   string `json:"userId"`
	Revision int    `json:"revision"`
	Confirm  bool   `json:"confirm"`
}

// Completed acknowledges ownership transfer.
type Completed struct {
	OK bool `json:"ok"`
}

// Transfer preserves all saved content while demoting the prior owner to full access.
func (a *Service) Transfer(ctx context.Context, actor session.Principal, workspaceID string, command Command) (Completed, error) {
	if !command.Confirm || command.UserID == "" || command.UserID == actor.ID {
		return Completed{}, invalid("Choose another active user and confirm ownership transfer.")
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	access, err := queries.RequireWorkspace(ctx, workspaceID, actor.ID, false)
	if err != nil {
		return Completed{}, unavailable(err)
	}
	if access.Role != "owner" {
		return Completed{}, &storage.AccessError{}
	}
	revision, err := queries.WorkspaceWorkspaceRevision(ctx, storage.WorkspaceWorkspaceRevisionParams{ID: workspaceID, OwnerID: actor.ID})
	if err != nil {
		return Completed{}, err
	}
	if revision != command.Revision {
		return Completed{}, &workspace.Error{Code: "metadata_conflict", Message: "Workspace changed. Refresh before transferring ownership."}
	}
	recipient, err := a.accounts.FindWithin(ctx, transaction, command.UserID)
	if err != nil {
		return Completed{}, err
	}
	if recipient.Status != account.StatusActive {
		return Completed{}, invalid("The new owner must be an active account.")
	}
	previous, err := a.accounts.FindWithin(ctx, transaction, actor.ID)
	if err != nil {
		return Completed{}, err
	}
	mapping, err := a.repository.TransferAssetsWithin(ctx, transaction, workspaceID, recipient.ID, newID)
	if err != nil {
		return Completed{}, err
	}
	err = a.rewriteMediaWithin(ctx, transaction, workspaceID, mapping)
	if err != nil {
		return Completed{}, err
	}
	err = queries.SharingRemoveMember(ctx, storage.SharingRemoveMemberParams{WorkspaceID: workspaceID, UserID: recipient.ID})
	if err != nil {
		return Completed{}, err
	}
	err = queries.SharingSetMember(ctx, storage.SharingSetMemberParams{WorkspaceID: workspaceID, UserID: actor.ID, Email: previous.Email, Role: "full"})
	if err != nil {
		return Completed{}, err
	}
	updated, updateErr := queries.OwnershipWorkspace(ctx, storage.OwnershipWorkspaceParams{ID: workspaceID, OwnerID: recipient.ID, PreviousOwner: actor.ID, UpdatedAt: time.Now().Unix(), Revision: command.Revision})
	if updateErr != nil {
		return Completed{}, updateErr
	}
	if updated != 1 {
		return Completed{}, invalid("Workspace ownership changed. Refresh and retry.")
	}
	err = queries.OwnershipRevokeLeases(ctx, workspaceID)
	if err != nil {
		return Completed{}, err
	}
	err = queries.OwnershipCancelRequests(ctx, workspaceID)
	if err != nil {
		return Completed{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Completed{}, fmt.Errorf("commit ownership transfer: %w", err)
	}
	a.publish(actor.ID, "workspace", workspaceID)

	return Completed{OK: true}, nil
}

func (a *Service) rewriteMediaWithin(ctx context.Context, transaction *storage.Transaction, workspaceID string, mapping map[string]string) error {
	if len(mapping) == 0 {
		return nil
	}
	queries := a.repository.Within(transaction)
	pages, err := queries.OwnershipPages(ctx, workspaceID)
	if err != nil {
		return err
	}
	for _, row := range pages {
		var document canvas.Document
		err = json.Unmarshal([]byte(row.Document), &document)
		if err != nil {
			return fmt.Errorf("decode transferred page: %w", err)
		}
		remap(document.Elements, mapping)
		raw, encodeErr := json.Marshal(document)
		if encodeErr != nil {
			return fmt.Errorf("encode transferred page: %w", encodeErr)
		}
		err = queries.OwnershipPageDocument(ctx, storage.OwnershipPageDocumentParams{ID: row.ID, Document: string(raw)})
		if err != nil {
			return err
		}
	}
	templates, err := queries.OwnershipTemplates(ctx, workspaceID)
	if err != nil {
		return err
	}
	for _, row := range templates {
		var elements []canvas.Element
		err = json.Unmarshal([]byte(row.Elements), &elements)
		if err != nil {
			return fmt.Errorf("decode transferred template: %w", err)
		}
		remap(elements, mapping)
		raw, encodeErr := json.Marshal(elements)
		if encodeErr != nil {
			return fmt.Errorf("encode transferred template: %w", encodeErr)
		}
		err = queries.OwnershipTemplateElements(ctx, storage.OwnershipTemplateElementsParams{ID: row.ID, Elements: string(raw)})
		if err != nil {
			return err
		}
	}

	return nil
}

func remap(elements []canvas.Element, mapping map[string]string) {
	for i := range elements {
		if resourceID, ok := mapping[elements[i].AssetID]; ok {
			elements[i].AssetID = resourceID
		}
	}
}

func invalid(message string) error {
	return &workspace.Error{Code: "validation_failed", Message: message}
}

func unavailable(err error) error {
	if storage.IsNotFound(err) {
		return &workspace.Error{Code: "not_found", Message: "Workspace unavailable."}
	}

	return err
}

func newID() string {
	value := make([]byte, identifierBytes)
	_, err := rand.Read(value)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(value)
}

const identifierBytes = 32
