package page

import (
	"context"
	"database/sql"
	"encoding/json"
	"mindflow/internal/canvas"
	"mindflow/internal/session"
	"strings"
	"time"
)

// ListPages lists metadata after checking parent workspace ownership.
func (a *Service) ListPages(ctx context.Context, actor session.Principal, resourceID string) (_ []Page, err error) {
	defer func() { err = repositoryError(err) }()
	ownedWorkspaceRow, err := New(a.db).OwnedWorkspace(ctx, OwnedWorkspaceParams{ID: resourceID, OwnerID: actor.ID})
	resourceID = ownedWorkspaceRow
	if err != nil {
		return nil, err
	}
	rows, err := New(a.db).ListPages(ctx, resourceID)
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
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Created{}, wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var wid string
	ownedWorkspaceRow, err := New(transaction).OwnedWorkspace(ctx, OwnedWorkspaceParams{ID: resourceID, OwnerID: actor.ID})
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
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Completed{}, wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := loadOwnedPage(ctx, transaction, resourceID, actor.ID)
	if err != nil {
		return Completed{}, err
	}
	if page.MetadataRevision != command.Revision {
		return Completed{}, failure("metadata_conflict", "Page changed. Refresh and retry.")
	}
	_, err = New(transaction).RenamePage(ctx, RenamePageParams{Name: strings.TrimSpace(command.Name), ID: page.ID})
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
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Completed{}, wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := loadOwnedPage(ctx, transaction, resourceID, actor.ID)
	if err != nil {
		return Completed{}, err
	}
	if page.MetadataRevision != command.Revision {
		return Completed{}, failure("metadata_conflict", "Page changed. Refresh and retry.")
	}
	_, err = a.images.DeletePageAssetsWithin(ctx, transaction, actor.ID, page.ID)
	if err != nil {
		return Completed{}, err
	}
	_, err = New(transaction).DeletePage(ctx, page.ID)
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

	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Created{}, wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := loadOwnedPage(ctx, transaction, resourceID, actor.ID)
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
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Created{}, wrap("canvas_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var wid string
	ownedWorkspaceRow, err := New(transaction).OwnedWorkspace(ctx, OwnedWorkspaceParams{ID: command.WorkspaceID, OwnerID: actor.ID})
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
	_, err = New(transaction).CreateRecoveryPage(ctx, CreateRecoveryPageParams{ID: resourceID, WorkspaceID: wid, Name: command.Name, Document: cloned, UpdatedAt: time.Now().Unix(), WorkspaceID_2: wid})
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
func (a *Service) CreateWithin(ctx context.Context, tx *sql.Tx, wid, name, raw string) (string, error) {
	id := newID()
	_, err := New(tx).CreatePage(ctx, CreatePageParams{ID: id, WorkspaceID: wid, Name: name, Document: raw, UpdatedAt: time.Now().Unix(), WorkspaceID_2: wid})

	return id, err
}

// CopyWithin duplicates a page and its canvas references in the caller transaction.
func (a *Service) CopyWithin(ctx context.Context, transaction *sql.Tx, source, wid, name string) (string, error) {
	var raw string
	pageDocumentRow, err := New(transaction).PageDocument(ctx, source)
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

func loadOwnedPage(ctx context.Context, tx *sql.Tx, id, owner string) (Page, error) {
	var page Page
	ownedPageRow, err := New(tx).OwnedPage(ctx, OwnedPageParams{ID: id, OwnerID: owner})
	page.ID = ownedPageRow.ID
	page.WorkspaceID = ownedPageRow.WorkspaceID
	page.Name = ownedPageRow.Name
	page.Position = ownedPageRow.Position
	page.Revision = ownedPageRow.Revision
	page.MetadataRevision = ownedPageRow.MetadataRevision

	return page, err
}
