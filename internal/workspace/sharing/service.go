package sharing

import (
	"context"
	"mindflow/internal/account"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/storage"
)

// Accounts resolves exact active account identities without reading account tables.
type Accounts interface {
	ActiveIDByEmail(ctx context.Context, address string) (string, bool, error)
	FindWithin(ctx context.Context, transaction *storage.Transaction, resourceID string) (account.User, error)
}

// Service owns workspace memberships and their editor-control consequences.
type Service struct {
	repository *storage.Repository
	accounts   Accounts
	publish    func(string, string, string)
}

// NewService supplies explicit account lookup and event delivery.
func NewService(repository *storage.Repository, accounts Accounts, publish func(string, string, string)) *Service {
	return &Service{repository: repository, accounts: accounts, publish: publish}
}

// Member describes one explicit workspace grant.
type (
	Member struct {
		UserID        string `json:"userId"`
		AvatarVersion string `json:"avatarVersion"`
		Email         string `json:"email"`
		Role          string `json:"role"`
	}
	// Command grants or changes a member's role.
	Command struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
)

func owned(ctx context.Context, queries *storage.Queries, resourceID, userID string) error {
	access, err := queries.RequireWorkspace(ctx, resourceID, userID, false)
	if storage.IsNotFound(err) {
		return notFound()
	}
	if err != nil {
		return err
	}
	if access.Role != "owner" {
		return &storage.AccessError{}
	}

	return nil
}

// List exposes membership details only to the workspace owner.
func (a *Service) List(ctx context.Context, actor session.Principal, resourceID string) ([]Member, error) {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	err = owned(ctx, queries, resourceID, actor.ID)
	if err != nil {
		return nil, err
	}
	rows, err := queries.SharingMembers(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	out := make([]Member, 0, len(rows))
	for _, row := range rows {
		user, readErr := a.accounts.FindWithin(ctx, transaction, row.UserID)
		if readErr != nil {
			return nil, readErr
		}
		out = append(out, Member{UserID: row.UserID, Email: row.Email, Role: row.Role, AvatarVersion: user.AvatarVersion})
	}

	return out, nil
}

// Set grants access to an existing active user and fences any former writer.
func (a *Service) Set(ctx context.Context, actor session.Principal, resourceID string, command Command) (Completed, error) {
	if command.Role != "read" && command.Role != "comment" && command.Role != "full" {
		return Completed{}, invalid("Choose read-only, comment or full access.")
	}
	address, err := account.NormalizeEmail(command.Email)
	if err != nil {
		return Completed{}, err
	}
	// Check ownership before resolving any account address.
	err = owned(ctx, a.repository.Queries, resourceID, actor.ID)
	if err != nil {
		return Completed{}, err
	}
	userID, exists, err := a.accounts.ActiveIDByEmail(ctx, address)
	if err != nil {
		return Completed{}, err
	}
	if !exists {
		return Completed{}, invalid("An active mindflow account with that email is required.")
	}
	if userID == actor.ID {
		return Completed{}, invalid("The workspace owner already has full access.")
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	err = owned(ctx, queries, resourceID, actor.ID)
	if err != nil {
		return Completed{}, err
	}
	user, err := a.accounts.FindWithin(ctx, transaction, userID)
	if err != nil {
		return Completed{}, err
	}
	if user.Status != account.StatusActive {
		return Completed{}, invalid("The account is no longer active.")
	}
	err = queries.SharingSetMember(ctx, storage.SharingSetMemberParams{WorkspaceID: resourceID, UserID: userID, Email: user.Email, Role: command.Role})
	if err != nil {
		return Completed{}, err
	}
	if command.Role != "full" {
		err = queries.SharingInvalidateLeases(ctx, storage.SharingInvalidateLeasesParams{WorkspaceID: resourceID, HolderUserID: userID})
		if err != nil {
			return Completed{}, err
		}
	}
	err = wrap("commit workspace sharing", transaction.Commit())
	if err != nil {
		return Completed{}, err
	}
	a.publish(actor.ID, "workspace", resourceID)

	return Completed{OK: true}, nil
}

// Remove revokes membership and any stale edit lease in the same transaction.
func (a *Service) Remove(ctx context.Context, actor session.Principal, resourceID, userID string) (Completed, error) {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	err = owned(ctx, queries, resourceID, actor.ID)
	if err != nil {
		return Completed{}, err
	}
	err = queries.SharingRemoveMember(ctx, storage.SharingRemoveMemberParams{WorkspaceID: resourceID, UserID: userID})
	if err != nil {
		return Completed{}, err
	}
	err = queries.SharingInvalidateLeases(ctx, storage.SharingInvalidateLeasesParams{WorkspaceID: resourceID, HolderUserID: userID})
	if err != nil {
		return Completed{}, err
	}
	err = wrap("commit workspace sharing", transaction.Commit())
	if err != nil {
		return Completed{}, err
	}
	a.publish(actor.ID, "workspace", resourceID)
	a.publish(userID, "workspace", resourceID)

	return Completed{OK: true}, nil
}

// Info returns owner identity to any currently authorized workspace member.
func (a *Service) Info(ctx context.Context, actor session.Principal, resourceID string) (Member, error) {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Member{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	access, err := a.repository.Within(transaction).RequireWorkspace(ctx, resourceID, actor.ID, false)
	if storage.IsNotFound(err) {
		return Member{}, notFound()
	}
	if err != nil {
		return Member{}, err
	}
	user, err := a.accounts.FindWithin(ctx, transaction, access.OwnerID)
	if err != nil {
		return Member{}, err
	}

	return Member{UserID: user.ID, Email: user.Email, AvatarVersion: user.AvatarVersion, Role: "owner"}, nil
}
