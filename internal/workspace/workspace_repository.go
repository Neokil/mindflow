package workspace

import (
	"context"
	"database/sql"
	"encoding/json"
	"mindflow/internal/authentication"
	"mindflow/internal/fault"
	"mindflow/internal/identifier"
	"mindflow/internal/workspace/dbquery"
	"strings"

	outcome "mindflow/internal/result"
)

// ListWorkspaces lists only workspaces owned by the caller.
func (a *Service) ListWorkspaces(ctx context.Context, actor authentication.Principal) (_ []Workspace, err error) {
	defer func() { err = repositoryError(err) }()

	rows, err := dbquery.New(a.db).ListWorkspaces(ctx, actor.ID)
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
func (a *Service) CreateWorkspace(ctx context.Context, actor authentication.Principal, command CreateWorkspaceCommand) (_ Created, err error) {
	defer func() { err = repositoryError(err) }()

	command.Name = strings.TrimSpace(command.Name)
	validateNameErr := ValidateName(command.Name)
	if validateNameErr != nil {
		return Created{}, validateNameErr
	}
	resourceID := identifier.New()
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return Created{}, fault.Wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	_, err = dbquery.New(transaction).CreateWorkspace(ctx, dbquery.CreateWorkspaceParams{ID: resourceID, OwnerID: actor.ID, Name: command.Name, CreatedAt: identifier.Now(), UpdatedAt: identifier.Now()})
	if err != nil {
		return Created{}, err
	}
	raw, err := json.Marshal(NewDocument())
	if err != nil {
		return Created{}, fault.Wrap("encode empty document", err)
	}
	pid, err := insertPage(ctx, transaction, resourceID, "Untitled", string(raw))
	if err != nil {
		return Created{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Created{}, fault.Wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", resourceID)

	return Created{ID: resourceID, PageID: pid}, nil
}

// RenameWorkspace renames an owned workspace using its metadata revision.
func (a *Service) RenameWorkspace(ctx context.Context, actor authentication.Principal, resourceID string, command RenameWorkspaceCommand) (_ outcome.Completed, err error) {
	defer func() { err = repositoryError(err) }()

	validateNameErr2 := ValidateName(command.Name)
	if validateNameErr2 != nil {
		return outcome.Completed{}, validateNameErr2
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Completed{}, fault.Wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var revision int
	workspaceRevisionRow, err := dbquery.New(transaction).WorkspaceRevision(ctx, dbquery.WorkspaceRevisionParams{ID: resourceID, OwnerID: actor.ID})
	revision = workspaceRevisionRow
	if err != nil {
		return outcome.Completed{}, err
	}
	if revision != command.Revision {
		return outcome.Completed{}, fault.Fault("metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	_, err = dbquery.New(transaction).RenameWorkspace(ctx, dbquery.RenameWorkspaceParams{Name: strings.TrimSpace(command.Name), UpdatedAt: identifier.Now(), ID: resourceID})
	if err != nil {
		return outcome.Completed{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return outcome.Completed{}, fault.Wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", resourceID)

	return outcome.Completed{OK: true}, nil
}

// DeleteWorkspace deletes an owned workspace and its unshared assets.
func (a *Service) DeleteWorkspace(ctx context.Context, actor authentication.Principal, resourceID string, command DeleteWorkspaceCommand) (_ outcome.Completed, err error) {
	defer func() { err = repositoryError(err) }()

	if !command.Confirm {
		return outcome.Completed{}, fault.Invalid("Confirm workspace deletion.")
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Completed{}, fault.Wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var rev int
	workspaceRevisionRow, err := dbquery.New(transaction).WorkspaceRevision(ctx, dbquery.WorkspaceRevisionParams{ID: resourceID, OwnerID: actor.ID})
	rev = workspaceRevisionRow
	if err != nil {
		return outcome.Completed{}, err
	}
	if rev != command.Revision {
		return outcome.Completed{}, fault.Fault("metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	_, err = dbquery.New(transaction).DeleteWorkspaceAssets(ctx, dbquery.DeleteWorkspaceAssetsParams{OwnerID: actor.ID, WorkspaceID: resourceID, WorkspaceID_2: resourceID})
	if err != nil {
		return outcome.Completed{}, err
	}
	_, err = dbquery.New(transaction).DeleteWorkspace(ctx, dbquery.DeleteWorkspaceParams{ID: resourceID, OwnerID: actor.ID})
	if err != nil {
		return outcome.Completed{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return outcome.Completed{}, fault.Wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", resourceID)

	return outcome.Completed{OK: true}, nil
}

// DuplicateWorkspace copies owned pages and assigns new element identifiers.
func (a *Service) DuplicateWorkspace(ctx context.Context, actor authentication.Principal, resourceID string) (_ outcome.Created, err error) {
	defer func() { err = repositoryError(err) }()

	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Created{}, fault.Wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var name string
	source := resourceID
	workspaceNameRow, err := dbquery.New(transaction).WorkspaceName(ctx, dbquery.WorkspaceNameParams{ID: source, OwnerID: actor.ID})
	name = workspaceNameRow
	if err != nil {
		return outcome.Created{}, err
	}
	resourceID = identifier.New()
	name = copyName(name)
	_, err = dbquery.New(transaction).CreateWorkspace(ctx, dbquery.CreateWorkspaceParams{ID: resourceID, OwnerID: actor.ID, Name: name, CreatedAt: identifier.Now(), UpdatedAt: identifier.Now()})
	if err != nil {
		return outcome.Created{}, err
	}
	rows, err := dbquery.New(transaction).PagesToCopy(ctx, source)
	if err != nil {
		return outcome.Created{}, err
	}
	for _, row := range rows {
		_, err = copyPage(ctx, transaction, row.ID, resourceID, row.Name)
		if err != nil {
			return outcome.Created{}, err
		}
	}
	err = transaction.Commit()
	if err != nil {
		return outcome.Created{}, fault.Wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", resourceID)

	return outcome.Created{ID: resourceID}, nil
}

// ListPages lists pages after checking workspace ownership.
func (a *Service) ListPages(ctx context.Context, actor authentication.Principal, resourceID string) (_ []Page, err error) {
	defer func() { err = repositoryError(err) }()

	ownedWorkspaceRow, err := dbquery.New(a.db).OwnedWorkspace(ctx, dbquery.OwnedWorkspaceParams{ID: resourceID, OwnerID: actor.ID})
	resourceID = ownedWorkspaceRow
	if err != nil {
		return nil, err
	}
	rows, err := dbquery.New(a.db).ListPages(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	pages := make([]Page, 0, len(rows))
	for _, row := range rows {
		pages = append(pages, Page{ID: row.ID, WorkspaceID: row.WorkspaceID, Name: row.Name, Position: row.Position, Revision: row.Revision, MetadataRevision: row.MetadataRevision})
	}

	return pages, nil
}

// CreatePage appends a blank page to an owned workspace.
func (a *Service) CreatePage(ctx context.Context, actor authentication.Principal, resourceID string, command CreatePageCommand) (_ outcome.Created, err error) {
	defer func() { err = repositoryError(err) }()

	validateNameErr3 := ValidateName(command.Name)
	if validateNameErr3 != nil {
		return outcome.Created{}, validateNameErr3
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Created{}, fault.Wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var wid string
	ownedWorkspaceRow, err := dbquery.New(transaction).OwnedWorkspace(ctx, dbquery.OwnedWorkspaceParams{ID: resourceID, OwnerID: actor.ID})
	wid = ownedWorkspaceRow
	if err != nil {
		return outcome.Created{}, err
	}
	raw, err := json.Marshal(NewDocument())
	if err != nil {
		return outcome.Created{}, fault.Wrap("encode empty document", err)
	}
	resourceID, err = insertPage(ctx, transaction, wid, command.Name, string(raw))
	if err != nil {
		return outcome.Created{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return outcome.Created{}, fault.Wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", wid)

	return outcome.Created{ID: resourceID}, nil
}

// RenamePage renames an owned page using its metadata revision.
func (a *Service) RenamePage(ctx context.Context, actor authentication.Principal, resourceID string, command RenamePageCommand) (_ outcome.Completed, err error) {
	defer func() { err = repositoryError(err) }()

	validateNameErr4 := ValidateName(command.Name)
	if validateNameErr4 != nil {
		return outcome.Completed{}, validateNameErr4
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Completed{}, fault.Wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := ownedPage(ctx, transaction, resourceID, actor.ID)
	if err != nil {
		return outcome.Completed{}, err
	}
	if page.MetadataRevision != command.Revision {
		return outcome.Completed{}, fault.Fault("metadata_conflict", "Page changed. Refresh and retry.")
	}
	_, err = dbquery.New(transaction).RenamePage(ctx, dbquery.RenamePageParams{Name: strings.TrimSpace(command.Name), ID: page.ID})
	if err != nil {
		return outcome.Completed{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return outcome.Completed{}, fault.Wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", page.WorkspaceID)

	return outcome.Completed{OK: true}, nil
}

// DeletePage deletes an owned page and its unshared assets.
func (a *Service) DeletePage(ctx context.Context, actor authentication.Principal, resourceID string, command DeletePageCommand) (_ outcome.Completed, err error) {
	defer func() { err = repositoryError(err) }()

	if !command.Confirm {
		return outcome.Completed{}, fault.Invalid("Confirm page deletion.")
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Completed{}, fault.Wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := ownedPage(ctx, transaction, resourceID, actor.ID)
	if err != nil {
		return outcome.Completed{}, err
	}
	if page.MetadataRevision != command.Revision {
		return outcome.Completed{}, fault.Fault("metadata_conflict", "Page changed. Refresh and retry.")
	}
	_, err = dbquery.New(transaction).DeletePageAssets(ctx, dbquery.DeletePageAssetsParams{OwnerID: actor.ID, PageID: page.ID, PageID_2: page.ID})
	if err != nil {
		return outcome.Completed{}, err
	}
	_, err = dbquery.New(transaction).DeletePage(ctx, page.ID)
	if err != nil {
		return outcome.Completed{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return outcome.Completed{}, fault.Wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", page.WorkspaceID)

	return outcome.Completed{OK: true}, nil
}

// DuplicatePage copies a page and preserves its image references.
func (a *Service) DuplicatePage(ctx context.Context, actor authentication.Principal, resourceID string) (_ outcome.Created, err error) {
	defer func() { err = repositoryError(err) }()

	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Created{}, fault.Wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := ownedPage(ctx, transaction, resourceID, actor.ID)
	if err != nil {
		return outcome.Created{}, err
	}
	resourceID, err = copyPage(ctx, transaction, page.ID, page.WorkspaceID, copyName(page.Name))
	if err != nil {
		return outcome.Created{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return outcome.Created{}, fault.Wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", page.WorkspaceID)

	return outcome.Created{ID: resourceID}, nil
}

// ReorderPages atomically validates and updates the full page order.
func (a *Service) ReorderPages(ctx context.Context, actor authentication.Principal, resourceID string, command ReorderPagesCommand) (_ outcome.Completed, err error) {
	defer func() { err = repositoryError(err) }()

	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Completed{}, fault.Wrap("workspace_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var rev int
	wid := resourceID
	workspaceRevisionRow, err := dbquery.New(transaction).WorkspaceRevision(ctx, dbquery.WorkspaceRevisionParams{ID: wid, OwnerID: actor.ID})
	rev = workspaceRevisionRow
	if err != nil {
		return outcome.Completed{}, err
	}
	if rev != command.Revision {
		return outcome.Completed{}, fault.Fault("metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	var count int
	countPagesRow, err := dbquery.New(transaction).CountPages(ctx, wid)
	count = int(countPagesRow)
	if err != nil {
		return outcome.Completed{}, err
	}
	if count != len(command.IDs) {
		return outcome.Completed{}, fault.Invalid("Include every page exactly once.")
	}
	seen := map[string]bool{}
	for i, id := range command.IDs {
		if seen[id] {
			return outcome.Completed{}, fault.Invalid("Duplicate page in order.")
		}
		seen[id] = true
		res, e := dbquery.New(transaction).PositionPage(ctx, dbquery.PositionPageParams{Position: i, ID: id, WorkspaceID: wid})
		if e != nil {
			return outcome.Completed{}, e
		}
		if count, _ := res.RowsAffected(); count != 1 {
			return outcome.Completed{}, fault.NotFound()
		}
	}
	_, err = dbquery.New(transaction).IncrementWorkspaceRevision(ctx, wid)
	if err != nil {
		return outcome.Completed{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return outcome.Completed{}, fault.Wrap("workspace_repository.go", err)
	}
	a.publish(actor.ID, "workspace", wid)

	return outcome.Completed{OK: true}, nil
}

func insertPage(ctx context.Context, tx *sql.Tx, wid, name, raw string) (string, error) {
	id := identifier.New()
	_, err := dbquery.New(tx).CreatePage(ctx, dbquery.CreatePageParams{ID: id, WorkspaceID: wid, Name: name, Document: raw, UpdatedAt: identifier.Now(), WorkspaceID_2: wid})

	return id, err
}

func copyPage(ctx context.Context, transaction *sql.Tx, source, wid, name string) (string, error) {
	var raw string
	pageDocumentRow, err := dbquery.New(transaction).PageDocument(ctx, source)
	raw = pageDocumentRow
	if err != nil {
		return "", err
	}
	cloned, err := cloneDocument(raw)
	if err != nil {
		return "", err
	}
	resourceID, err := insertPage(ctx, transaction, wid, name, cloned)
	if err != nil {
		return "", err
	}
	_, err = dbquery.New(transaction).CopyPageAssets(ctx, dbquery.CopyPageAssetsParams{PageID: resourceID, PageID_2: source})

	return resourceID, err
}

func copyName(name string) string {
	r := []rune(name)
	if len(r) > copyNamePrefixLength {
		r = r[:copyNamePrefixLength]
	}

	return string(r) + " copy"
}

func ownedPage(ctx context.Context, tx *sql.Tx, id, owner string) (Page, error) {
	var page Page
	ownedPageRow, err := dbquery.New(tx).OwnedPage(ctx, dbquery.OwnedPageParams{ID: id, OwnerID: owner})
	page.ID = ownedPageRow.ID
	page.WorkspaceID = ownedPageRow.WorkspaceID
	page.Name = ownedPageRow.Name
	page.Position = ownedPageRow.Position
	page.Revision = ownedPageRow.Revision
	page.MetadataRevision = ownedPageRow.MetadataRevision

	return page, err
}

func cloneDocument(raw string) (string, error) {
	var document Document
	unmarshalErr := json.Unmarshal([]byte(raw), &document)
	if unmarshalErr != nil {
		return "", fault.Wrap("workspace_repository.go", unmarshalErr)
	}
	groups := map[string]string{}
	for iValue := range document.Elements {
		document.Elements[iValue].ID = identifier.New()
		g := document.Elements[iValue].GroupID
		if g != "" {
			if groups[g] == "" {
				groups[g] = identifier.New()
			}
			document.Elements[iValue].GroupID = groups[g]
		}
	}
	data, err := json.Marshal(document)

	return string(data), err
}
