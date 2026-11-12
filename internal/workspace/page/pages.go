package page

import (
	"context"
	"encoding/json"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/storage"
	"strings"
	"time"
)

// ListPages lists metadata after checking parent workspace ownership.
func (a *Service) ListPages(ctx context.Context, actor session.Principal, resourceID string) (_ []Page, err error) {
	defer func() { err = repositoryError(err) }()
	ownedWorkspaceRow, err := a.repository.PageOwnedWorkspace(ctx, storage.PageOwnedWorkspaceParams{ID: resourceID, OwnerID: actor.ID})
	resourceID = ownedWorkspaceRow
	if err != nil {
		return nil, err
	}
	rows, err := a.repository.PageListPages(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	pages := make([]Page, 0, len(rows))
	for _, row := range rows {
		pages = append(pages, Page(row))
	}

	return pages, nil
}

// CreatePage appends a blank page to an owned workspace.
func (a *Service) CreatePage(ctx context.Context, actor session.Principal, resourceID string, command CreatePageCommand) (_ Created, err error) {
	defer func() { err = repositoryError(err) }()

	validateNameErr3 := ValidateName(command.Name)
	if validateNameErr3 != nil {
		return Created{}, validateNameErr3
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Created{}, wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	_, accessErr := a.repository.Within(transaction).RequireWorkspace(ctx, resourceID, actor.ID, true)
	if accessErr != nil {
		return Created{}, accessErr
	}
	var wid string
	ownedWorkspaceRow, err := a.repository.Within(transaction).PageOwnedWorkspace(ctx, storage.PageOwnedWorkspaceParams{ID: resourceID, OwnerID: actor.ID})
	wid = ownedWorkspaceRow
	if err != nil {
		return Created{}, err
	}
	raw, err := json.Marshal(canvas.NewDocument())
	if err != nil {
		return Created{}, wrap("encode empty document", err)
	}
	resourceID, err = a.CreateWithin(ctx, transaction, wid, command.Name, string(raw))
	if err != nil {
		return Created{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Created{}, wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", wid)

	return Created{ID: resourceID}, nil
}

// RenamePage renames an owned page using its metadata revision.
func (a *Service) RenamePage(ctx context.Context, actor session.Principal, resourceID string, command RenamePageCommand) (_ Completed, err error) {
	defer func() { err = repositoryError(err) }()

	validateNameErr4 := ValidateName(command.Name)
	if validateNameErr4 != nil {
		return Completed{}, validateNameErr4
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := a.loadOwnedPage(ctx, transaction, resourceID, actor.ID)
	if err != nil {
		return Completed{}, err
	}
	if page.MetadataRevision != command.Revision {
		return Completed{}, failure("metadata_conflict", "Page changed. Refresh and retry.")
	}
	_, err = a.repository.Within(transaction).PageRenamePage(ctx, storage.PageRenamePageParams{Name: strings.TrimSpace(command.Name), ID: page.ID})
	if err != nil {
		return Completed{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Completed{}, wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", page.WorkspaceID)

	return Completed{OK: true}, nil
}

// DeletePage deletes an owned page and its unshared assets.
func (a *Service) DeletePage(ctx context.Context, actor session.Principal, resourceID string, command DeletePageCommand) (_ Completed, err error) {
	defer func() { err = repositoryError(err) }()

	if !command.Confirm {
		return Completed{}, invalid("Confirm page deletion.")
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := a.loadOwnedPage(ctx, transaction, resourceID, actor.ID)
	if err != nil {
		return Completed{}, err
	}
	if page.MetadataRevision != command.Revision {
		return Completed{}, failure("metadata_conflict", "Page changed. Refresh and retry.")
	}
	_, err = a.images.DeletePageAssetsWithin(ctx, transaction, page.ID)
	if err != nil {
		return Completed{}, err
	}
	_, err = a.repository.Within(transaction).PageDeletePage(ctx, page.ID)
	if err != nil {
		return Completed{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Completed{}, wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", page.WorkspaceID)

	return Completed{OK: true}, nil
}

// DuplicatePage copies a page and preserves its image references.
func (a *Service) DuplicatePage(ctx context.Context, actor session.Principal, resourceID string) (_ Created, err error) {
	defer func() { err = repositoryError(err) }()

	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Created{}, wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := a.loadOwnedPage(ctx, transaction, resourceID, actor.ID)
	if err != nil {
		return Created{}, err
	}
	resourceID, err = a.CopyWithin(ctx, transaction, page.ID, page.WorkspaceID, copyName(page.Name))
	if err != nil {
		return Created{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Created{}, wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", page.WorkspaceID)

	return Created{ID: resourceID}, nil
}

// RecoveryCopy preserves disconnected edits in a new page owned by the caller.
func (a *Service) RecoveryCopy(ctx context.Context, actor session.Principal, command RecoveryCopyCommand) (_ Created, err error) {
	defer func() { err = repositoryError(err) }()

	validateDocumentErr2 := canvas.ValidateDocument(command.Document)
	if validateDocumentErr2 != nil {
		return Created{}, validateDocumentErr2
	}
	if strings.TrimSpace(command.Name) == "" {
		command.Name = "Recovered page"
	}
	validateNameErr := ValidateName(command.Name)
	if validateNameErr != nil {
		return Created{}, validateNameErr
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Created{}, wrap("canvas_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	_, accessErr := a.repository.Within(transaction).RequireWorkspace(ctx, command.WorkspaceID, actor.ID, true)
	if accessErr != nil {
		return Created{}, accessErr
	}
	var wid string
	ownedWorkspaceRow, err := a.repository.Within(transaction).PageOwnedWorkspace(ctx, storage.PageOwnedWorkspaceParams{ID: command.WorkspaceID, OwnerID: actor.ID})
	wid = ownedWorkspaceRow
	if err != nil {
		return Created{}, err
	}
	raw, err := json.Marshal(command.Document)
	if err != nil {
		return Created{}, wrap("encode document", err)
	}
	cloned, err := canvas.CloneDocument(string(raw))
	if err != nil {
		return Created{}, err
	}
	resourceID := newID()
	_, err = a.repository.Within(transaction).PageCreateRecoveryPage(ctx, storage.PageCreateRecoveryPageParams{ID: resourceID, WorkspaceID: wid, Name: command.Name, Document: cloned, UpdatedAt: time.Now().Unix(), WorkspaceID_2: wid})
	if err != nil {
		return Created{}, err
	}
	err = a.canvas.ValidateAssetsWithin(ctx, transaction, actor.ID, resourceID, command.Document)
	if err != nil {
		return Created{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Created{}, wrap("canvas_repository.go", err)
	}
	a.publish(actor.ID, "workspace", wid)

	return Created{ID: resourceID}, nil
}

// CreateWithin inserts page metadata and content in the caller transaction.
func (a *Service) CreateWithin(ctx context.Context, tx *storage.Transaction, wid, name, raw string) (string, error) {
	id := newID()
	_, err := a.repository.Within(tx).PageCreatePage(ctx, storage.PageCreatePageParams{ID: id, WorkspaceID: wid, Name: name, Document: raw, UpdatedAt: time.Now().Unix(), WorkspaceID_2: wid})

	return id, err
}

// CopyWithin duplicates a page and its canvas references in the caller transaction.
func (a *Service) CopyWithin(ctx context.Context, transaction *storage.Transaction, source, wid, name string) (string, error) {
	var raw string
	pageDocumentRow, err := a.repository.Within(transaction).PagePageDocument(ctx, source)
	raw = pageDocumentRow
	if err != nil {
		return "", err
	}
	cloned, err := canvas.CloneDocument(raw)
	if err != nil {
		return "", err
	}
	resourceID, err := a.CreateWithin(ctx, transaction, wid, name, cloned)
	if err != nil {
		return "", err
	}
	err = a.canvas.CopyAssetsWithin(ctx, transaction, resourceID, source)

	return resourceID, err
}

func copyName(name string) string {
	r := []rune(name)
	if len(r) > copyNamePrefixLength {
		r = r[:copyNamePrefixLength]
	}

	return string(r) + " copy"
}

func (a *Service) loadOwnedPage(ctx context.Context, transaction *storage.Transaction, resourceID, owner string) (Page, error) {
	_, accessErr := a.repository.Within(transaction).RequirePage(ctx, resourceID, owner, true)
	if accessErr != nil {
		return Page{}, accessErr
	}
	var page Page
	ownedPageRow, err := a.repository.Within(transaction).PageOwnedPage(ctx, storage.PageOwnedPageParams{ID: resourceID, OwnerID: owner})
	page.ID = ownedPageRow.ID
	page.WorkspaceID = ownedPageRow.WorkspaceID
	page.Name = ownedPageRow.Name
	page.Position = ownedPageRow.Position
	page.Revision = ownedPageRow.Revision
	page.MetadataRevision = ownedPageRow.MetadataRevision

	return page, err
}
