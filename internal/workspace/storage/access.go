package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Access identifies an authorized workspace relationship without account table access.
type Access struct{ WorkspaceID, OwnerID, Role string }

// AccessError is a safe authorization failure from the owning domain's storage policy.
type AccessError struct{}

func (*AccessError) Error() string { return "Your workspace role does not allow this action." }

// FailureCode supplies the HTTP authorization status.
func (*AccessError) FailureCode() string { return "forbidden" }

// UserMessage supplies a safe response.
func (a *AccessError) UserMessage() string { return a.Error() }

// CanEdit permits content changes and editor takeover.
func CanEdit(role string) bool { return role == "owner" || role == "full" }

// CanComment permits independent comment participation.
func CanComment(role string) bool { return CanEdit(role) || role == "comment" }

// RequireWorkspace checks current membership inside the caller's transaction.
func (queries *Queries) RequireWorkspace(ctx context.Context, resourceID, userID string, edit bool) (Access, error) {
	row, err := queries.AccessWorkspace(ctx, AccessWorkspaceParams{ID: resourceID, UserID: userID})
	if err != nil {
		return Access{}, err
	}
	if edit && !CanEdit(row.Role) {
		return Access{}, &AccessError{}
	}

	return Access(row), nil
}

// RequirePage checks page access through the parent workspace.
func (queries *Queries) RequirePage(ctx context.Context, resourceID, userID string, edit bool) (Access, error) {
	row, err := queries.AccessPage(ctx, AccessPageParams{ID: resourceID, UserID: userID})
	if err != nil {
		return Access{}, err
	}
	if edit && !CanEdit(row.Role) {
		return Access{}, &AccessError{}
	}

	return Access(row), nil
}

// CreateScopedAsset authorizes and assigns shared uploads to the workspace owner atomically.
func (a *Repository) CreateScopedAsset(ctx context.Context, userID, workspaceID string, params ImageCreateAssetParams) error {
	transaction, err := a.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.Within(transaction)
	if workspaceID != "" {
		access, accessErr := queries.RequireWorkspace(ctx, workspaceID, userID, true)
		if accessErr != nil {
			return accessErr
		}
		params.OwnerID = access.OwnerID
	}
	_, err = queries.ImageCreateAsset(ctx, params)
	if err != nil {
		return err
	}
	if workspaceID != "" {
		err = queries.SharingScopeAsset(ctx, SharingScopeAssetParams{WorkspaceID: sql.NullString{String: workspaceID, Valid: true}, ID: params.ID})
		if err != nil {
			return err
		}
	}

	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("commit scoped image: %w", err)
	}

	return nil
}

// EventAudience resolves recipients for workspace-owned resources after a commit.
func (a *Repository) EventAudience(ctx context.Context, kind, resourceID string) ([]string, error) {
	var workspaces []string
	switch kind {
	case "workspace":
		workspaces = []string{resourceID}
	case "page", "lease", "comment":
		// Internal event routing uses only workspace-owned page metadata.
		page, err := a.PageWorkspace(ctx, resourceID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		workspaces = []string{page}
	case "asset":
		ids, err := a.SharingAssetWorkspaces(ctx, SharingAssetWorkspacesParams{ID: resourceID, AssetID: resourceID, TemplateAssetID: resourceID})
		if err != nil {
			return nil, err
		}
		for _, value := range ids {
			if value.Valid {
				workspaces = append(workspaces, value.String)
			}
		}
	default:
		return nil, nil
	}
	recipients := map[string]bool{}
	for _, resourceID := range workspaces {
		users, err := a.SharingAudience(ctx, resourceID)
		if err != nil {
			return nil, err
		}
		for _, user := range users {
			recipients[user] = true
		}
	}
	out := make([]string, 0, len(recipients))
	for resourceID := range recipients {
		out = append(out, resourceID)
	}

	return out, nil
}
