package workspace

import (
	"context"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/storage"
	"strings"
	"time"
)

// ListWorkspaces lists only workspaces owned by the caller.
func (a *Service) ListWorkspaces(ctx context.Context, actor session.Principal) (_ []Workspace, err error) {
	defer func() { err = repositoryError(err) }()
	rows, err := a.repository.WorkspaceListWorkspaces(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	out := make([]Workspace, 0, len(rows))
	for _, row := range rows {
		out = append(out, Workspace{ID: row.ID, Name: row.Name, Revision: row.Revision, UpdatedAt: row.UpdatedAt, PageCount: int(row.PageCount)})
	}

	return out, nil
}

// CreateWorkspace creates a private workspace with its first page.
func (a *Service) CreateWorkspace(ctx context.Context, actor session.Principal, command CreateWorkspaceCommand) (_ Created, err error) {
	defer func() { err = repositoryError(err) }()

	command.Name = strings.TrimSpace(command.Name)
	validateNameErr := ValidateName(command.Name)
	if validateNameErr != nil {
		return Created{}, validateNameErr
	}
	resourceID := newID()
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Created{}, wrap("workspace.CreateWorkspace", err)
	}
	defer func() { _ = transaction.Rollback() }()
	_, err = a.repository.Within(transaction).WorkspaceCreateWorkspace(ctx, storage.WorkspaceCreateWorkspaceParams{ID: resourceID, OwnerID: actor.ID, Name: command.Name, CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix()})
	if err != nil {
		return Created{}, err
	}
	pid, err := a.pages.CreateBlankWithin(ctx, transaction, resourceID, "Untitled")
	if err != nil {
		return Created{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Created{}, wrap("workspace.CreateWorkspace", err)
	}
	a.publish(actor.ID, "workspace", resourceID)

	return Created{ID: resourceID, PageID: pid}, nil
}

// RenameWorkspace renames an owned workspace using its metadata revision.
func (a *Service) RenameWorkspace(ctx context.Context, actor session.Principal, resourceID string, command RenameWorkspaceCommand) (_ Completed, err error) {
	defer func() { err = repositoryError(err) }()

	validateNameErr2 := ValidateName(command.Name)
	if validateNameErr2 != nil {
		return Completed{}, validateNameErr2
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, wrap("workspace.RenameWorkspace", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var revision int
	workspaceRevisionRow, err := a.repository.Within(transaction).WorkspaceWorkspaceRevision(ctx, storage.WorkspaceWorkspaceRevisionParams{ID: resourceID, OwnerID: actor.ID})
	revision = workspaceRevisionRow
	if err != nil {
		return Completed{}, err
	}
	if revision != command.Revision {
		return Completed{}, failure("metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	_, err = a.repository.Within(transaction).WorkspaceRenameWorkspace(ctx, storage.WorkspaceRenameWorkspaceParams{Name: strings.TrimSpace(command.Name), UpdatedAt: time.Now().Unix(), ID: resourceID})
	if err != nil {
		return Completed{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Completed{}, wrap("workspace.RenameWorkspace", err)
	}
	a.publish(actor.ID, "workspace", resourceID)

	return Completed{OK: true}, nil
}

// DeleteWorkspace deletes an owned workspace and its unshared assets.
func (a *Service) DeleteWorkspace(ctx context.Context, actor session.Principal, resourceID string, command DeleteWorkspaceCommand) (_ Completed, err error) {
	defer func() { err = repositoryError(err) }()

	if !command.Confirm {
		return Completed{}, invalid("Confirm workspace deletion.")
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, wrap("workspace.DeleteWorkspace", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var rev int
	workspaceRevisionRow, err := a.repository.Within(transaction).WorkspaceWorkspaceRevision(ctx, storage.WorkspaceWorkspaceRevisionParams{ID: resourceID, OwnerID: actor.ID})
	rev = workspaceRevisionRow
	if err != nil {
		return Completed{}, err
	}
	if rev != command.Revision {
		return Completed{}, failure("metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	_, err = a.images.DeleteWorkspaceAssetsWithin(ctx, transaction, actor.ID, resourceID)
	if err != nil {
		return Completed{}, err
	}
	_, err = a.repository.Within(transaction).WorkspaceDeleteWorkspace(ctx, storage.WorkspaceDeleteWorkspaceParams{ID: resourceID, OwnerID: actor.ID})
	if err != nil {
		return Completed{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Completed{}, wrap("workspace.DeleteWorkspace", err)
	}
	a.publish(actor.ID, "workspace", resourceID)

	return Completed{OK: true}, nil
}

// DuplicateWorkspace copies owned pages and assigns new element identifiers.
func (a *Service) DuplicateWorkspace(ctx context.Context, actor session.Principal, resourceID string) (_ Duplicated, err error) {
	defer func() { err = repositoryError(err) }()

	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Duplicated{}, wrap("workspace.DuplicateWorkspace", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var name string
	source := resourceID
	workspaceNameRow, err := a.repository.Within(transaction).WorkspaceWorkspaceName(ctx, storage.WorkspaceWorkspaceNameParams{ID: source, OwnerID: actor.ID})
	name = workspaceNameRow
	if err != nil {
		return Duplicated{}, err
	}
	resourceID = newID()
	name = copyName(name)
	_, err = a.repository.Within(transaction).WorkspaceCreateWorkspace(ctx, storage.WorkspaceCreateWorkspaceParams{ID: resourceID, OwnerID: actor.ID, Name: name, CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix()})
	if err != nil {
		return Duplicated{}, err
	}
	err = a.pages.CopyWorkspaceWithin(ctx, transaction, source, resourceID)
	if err != nil {
		return Duplicated{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Duplicated{}, wrap("workspace.DuplicateWorkspace", err)
	}
	a.publish(actor.ID, "workspace", resourceID)

	return Duplicated{ID: resourceID}, nil
}

// ReorderPages atomically validates and updates the full page order.
func (a *Service) ReorderPages(ctx context.Context, actor session.Principal, resourceID string, command ReorderPagesCommand) (_ Completed, err error) {
	defer func() { err = repositoryError(err) }()

	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, wrap("workspace.ReorderPages", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var rev int
	wid := resourceID
	workspaceRevisionRow, err := a.repository.Within(transaction).WorkspaceWorkspaceRevision(ctx, storage.WorkspaceWorkspaceRevisionParams{ID: wid, OwnerID: actor.ID})
	rev = workspaceRevisionRow
	if err != nil {
		return Completed{}, err
	}
	if rev != command.Revision {
		return Completed{}, failure("metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	var count int
	countPagesRow, err := a.pages.CountWithin(ctx, transaction, wid)
	count = int(countPagesRow)
	if err != nil {
		return Completed{}, err
	}
	if count != len(command.IDs) {
		return Completed{}, invalid("Include every page exactly once.")
	}
	seen := map[string]bool{}
	for i, id := range command.IDs {
		if seen[id] {
			return Completed{}, invalid("Duplicate page in order.")
		}
		seen[id] = true
		updated, e := a.pages.PositionWithin(ctx, transaction, i, id, wid)
		if e != nil {
			return Completed{}, e
		}
		if updated != 1 {
			return Completed{}, notFound()
		}
	}
	_, err = a.repository.Within(transaction).WorkspaceIncrementWorkspaceRevision(ctx, wid)
	if err != nil {
		return Completed{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Completed{}, wrap("workspace.ReorderPages", err)
	}
	a.publish(actor.ID, "workspace", wid)

	return Completed{OK: true}, nil
}

func copyName(name string) string {
	r := []rune(name)
	if len(r) > copyNamePrefixLength {
		r = r[:copyNamePrefixLength]
	}

	return string(r) + " copy"
}
