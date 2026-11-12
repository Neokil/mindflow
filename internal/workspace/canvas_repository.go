package workspace

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"mindflow/internal/authentication"
	"mindflow/internal/fault"
	"mindflow/internal/identifier"
	"mindflow/internal/workspace/dbquery"
	"strings"

	outcome "mindflow/internal/result"
)

// GetDocument loads the current document after checking ownership.
func (a *Service) GetDocument(ctx context.Context, actor authentication.Principal, resourceID string) (_ PageSnapshot, err error) {
	defer func() { err = repositoryError(err) }()

	var raw string
	var revision, metadata int
	var name, workspace string
	getDocumentRow, err := dbquery.New(a.db).GetDocument(ctx, dbquery.GetDocumentParams{ID: resourceID, OwnerID: actor.ID})
	raw = getDocumentRow.Document
	revision = getDocumentRow.Revision
	metadata = getDocumentRow.MetadataRevision
	name = getDocumentRow.Name
	workspace = getDocumentRow.WorkspaceID

	if err != nil {
		return PageSnapshot{}, err
	}

	return PageSnapshot{Document: json.RawMessage(raw), Revision: revision, MetadataRevision: metadata, Name: name, WorkspaceID: workspace}, nil
}

// SaveDocument atomically fences the editor, validates assets and records an idempotent save.
func (a *Service) SaveDocument(ctx context.Context, actor authentication.Principal, resourceID string, command SaveDocumentCommand) (_ SaveReceipt, err error) {
	defer func() { err = repositoryError(err) }()

	if len(command.MutationID) < 8 || len(command.MutationID) > maxIdentifierLength || len(command.ClientID) < 8 || len(command.ClientID) > maxIdentifierLength {
		return SaveReceipt{}, fault.Invalid("Invalid save identity.")
	}
	validateDocumentErr := ValidateDocument(command.Document)
	if validateDocumentErr != nil {
		return SaveReceipt{}, validateDocumentErr
	}
	raw, err := json.Marshal(command.Document)
	if err != nil {
		return SaveReceipt{}, fault.Wrap("encode document", err)
	}
	if len(raw) > 16<<defaultGridSpacing {
		return SaveReceipt{}, fault.Fault("document_limit", "The page exceeds the 16 MB document limit.")
	}
	requestRaw, err := json.Marshal(command)
	if err != nil {
		return SaveReceipt{}, fault.Wrap("encode save request", err)
	}
	hash := identifier.Digest(string(requestRaw))
	page := actor
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return SaveReceipt{}, fault.Wrap("canvas_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var revision int
	ownedPageRevisionRow, err := dbquery.New(transaction).OwnedPageRevision(ctx, dbquery.OwnedPageRevisionParams{ID: resourceID, OwnerID: page.ID})
	revision = ownedPageRevisionRow
	if err != nil {
		return SaveReceipt{}, err
	}
	var receiptHash string
	var receiptRev int
	saveReceiptRow, err := dbquery.New(transaction).SaveReceipt(ctx, dbquery.SaveReceiptParams{PageID: resourceID, ClientID: command.ClientID, MutationID: command.MutationID, ExpiresAt: identifier.Now()})
	receiptHash = saveReceiptRow.Digest
	receiptRev = saveReceiptRow.Revision

	if err == nil {
		if hash != receiptHash {
			return SaveReceipt{}, fault.Invalid("Mutation ID was reused with different content.")
		}

		return SaveReceipt{Revision: receiptRev, MutationID: command.MutationID}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return SaveReceipt{}, err
	}
	var session sql.NullString
	var client string
	var epoch, expires int64
	pageLeaseRow, err := dbquery.New(transaction).PageLease(ctx, resourceID)
	session = pageLeaseRow.SessionID
	client = pageLeaseRow.ClientID
	epoch = pageLeaseRow.Epoch
	expires = pageLeaseRow.ExpiresAt

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SaveReceipt{}, fault.Fault("lease_lost", "Editor control expired.")
		}

		return SaveReceipt{}, err
	}
	state := LeaseState{SessionID: session.String, ClientID: client, Epoch: epoch, ExpiresAt: expires}
	err = state.CheckSave(page.SessionID, command.ClientID, command.Epoch, command.BaseRevision, revision, identifier.Now())
	if err != nil {
		return SaveReceipt{}, err
	}
	err = validateAssets(ctx, transaction, page.ID, resourceID, command.Document)
	if err != nil {
		return SaveReceipt{}, err
	}
	revision++
	_, err = dbquery.New(transaction).SaveDocument(ctx, dbquery.SaveDocumentParams{Document: string(raw), Revision: revision, UpdatedAt: identifier.Now(), ID: resourceID})
	if err != nil {
		return SaveReceipt{}, err
	}
	_, err = dbquery.New(transaction).TouchWorkspace(ctx, dbquery.TouchWorkspaceParams{UpdatedAt: identifier.Now(), ID: resourceID})
	if err != nil {
		return SaveReceipt{}, err
	}
	_, err = dbquery.New(transaction).CreateSaveReceipt(ctx, dbquery.CreateSaveReceiptParams{PageID: resourceID, ClientID: command.ClientID, MutationID: command.MutationID, Digest: hash, Revision: revision, ExpiresAt: identifier.Now() + receiptLifetimeSeconds})
	if err != nil {
		return SaveReceipt{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return SaveReceipt{}, fault.Wrap("canvas_repository.go", err)
	}
	a.publish(page.ID, "page", resourceID)
	a.enqueueImageVariants(ctx, page.ID, command.Document)

	return SaveReceipt{Revision: revision, MutationID: command.MutationID}, nil
}

// RecoveryCopy preserves disconnected edits in a new page owned by the caller.
func (a *Service) RecoveryCopy(ctx context.Context, actor authentication.Principal, command RecoveryCopyCommand) (_ outcome.Created, err error) {
	defer func() { err = repositoryError(err) }()

	validateDocumentErr2 := ValidateDocument(command.Document)
	if validateDocumentErr2 != nil {
		return outcome.Created{}, validateDocumentErr2
	}
	if strings.TrimSpace(command.Name) == "" {
		command.Name = "Recovered page"
	}
	validateNameErr := ValidateName(command.Name)
	if validateNameErr != nil {
		return outcome.Created{}, validateNameErr
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Created{}, fault.Wrap("canvas_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var wid string
	ownedWorkspaceRow, err := dbquery.New(transaction).OwnedWorkspace(ctx, dbquery.OwnedWorkspaceParams{ID: command.WorkspaceID, OwnerID: actor.ID})
	wid = ownedWorkspaceRow
	if err != nil {
		return outcome.Created{}, err
	}
	raw, err := json.Marshal(command.Document)
	if err != nil {
		return outcome.Created{}, fault.Wrap("encode document", err)
	}
	cloned, err := cloneDocument(string(raw))
	if err != nil {
		return outcome.Created{}, err
	}
	resourceID := identifier.New()
	_, err = dbquery.New(transaction).CreateRecoveryPage(ctx, dbquery.CreateRecoveryPageParams{ID: resourceID, WorkspaceID: wid, Name: command.Name, Document: cloned, UpdatedAt: identifier.Now(), WorkspaceID_2: wid})
	if err != nil {
		return outcome.Created{}, err
	}
	err = validateAssets(ctx, transaction, actor.ID, resourceID, command.Document)
	if err != nil {
		return outcome.Created{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return outcome.Created{}, fault.Wrap("canvas_repository.go", err)
	}
	a.publish(actor.ID, "workspace", wid)

	return outcome.Created{ID: resourceID}, nil
}

// Lease applies an editor-control action in the ownership transaction.
func (a *Service) Lease(ctx context.Context, actor authentication.Principal, resourceID string, command LeaseCommand) (_ LeaseResult, err error) {
	defer func() { err = repositoryError(err) }()

	if len(command.ClientID) < 8 || len(command.ClientID) > maxIdentifierLength {
		return LeaseResult{}, fault.Invalid("Invalid editor identity.")
	}
	if command.Action != "acquire" && command.Action != "takeover" && command.Action != leaseRenew && command.Action != "release" {
		return LeaseResult{}, fault.Invalid("Invalid editor action.")
	}
	caller := actor
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return LeaseResult{}, fault.Wrap("canvas_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	page, err := ownedPage(ctx, transaction, resourceID, caller.ID)
	if err != nil {
		return LeaseResult{}, err
	}
	_, err = dbquery.New(transaction).EnsureLease(ctx, page.ID)
	if err != nil {
		return LeaseResult{}, err
	}
	var session sql.NullString
	var client string
	var epoch, expires int64
	pageLeaseRow, err := dbquery.New(transaction).PageLease(ctx, page.ID)
	session = pageLeaseRow.SessionID
	client = pageLeaseRow.ClientID
	epoch = pageLeaseRow.Epoch
	expires = pageLeaseRow.ExpiresAt
	if err != nil {
		return LeaseResult{}, err
	}
	state, changed, err := (LeaseState{SessionID: session.String, ClientID: client, Epoch: epoch, ExpiresAt: expires}).Advance(caller.SessionID, command, identifier.Now())
	if err != nil {
		return LeaseResult{}, err
	}
	epoch, expires, client = state.Epoch, state.ExpiresAt, state.ClientID
	session = sql.NullString{String: state.SessionID, Valid: state.SessionID != ""}
	if changed {
		_, err = dbquery.New(transaction).UpdateLease(ctx, dbquery.UpdateLeaseParams{SessionID: session, ClientID: client, Epoch: epoch, ExpiresAt: expires, PageID: page.ID})
		if err != nil {
			return LeaseResult{}, err
		}
	}
	err = transaction.Commit()
	if err != nil {
		return LeaseResult{}, fault.Wrap("canvas_repository.go", err)
	}
	if changed && command.Action != leaseRenew {
		a.publish(caller.ID, "lease", page.ID)
	}

	return LeaseResult{Editable: session.String == caller.SessionID && client == command.ClientID && expires > identifier.Now(), Epoch: epoch, ExpiresAt: expires, Revision: page.Revision}, nil
}

// LeaseStatus reports editor control and revision for an owned page.
func (a *Service) LeaseStatus(ctx context.Context, actor authentication.Principal, resourceID string, clientID string) (_ LeaseResult, err error) {
	defer func() { err = repositoryError(err) }()

	var rev int
	ownedPageRevisionRow, err := dbquery.New(a.db).OwnedPageRevision(ctx, dbquery.OwnedPageRevisionParams{ID: resourceID, OwnerID: actor.ID})
	rev = ownedPageRevisionRow
	if err != nil {
		return LeaseResult{}, err
	}
	var session sql.NullString
	var client string
	var epoch, expires int64
	pageLeaseRow, err := dbquery.New(a.db).PageLease(ctx, resourceID)
	session = pageLeaseRow.SessionID
	client = pageLeaseRow.ClientID
	epoch = pageLeaseRow.Epoch
	expires = pageLeaseRow.ExpiresAt

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return LeaseResult{}, err
	}

	return LeaseResult{Editable: session.String == actor.SessionID && client == clientID && expires > identifier.Now(), Epoch: epoch, ExpiresAt: expires, Revision: rev}, nil
}

func validateAssets(ctx context.Context, transaction *sql.Tx, owner, pageID string, d Document) error {
	seen := map[string]bool{}
	for _, entry := range d.Elements {
		if entry.Type != elementImage || seen[entry.AssetID] {
			continue
		}
		seen[entry.AssetID] = true
		var count int
		countOwnedAssetRow, err := dbquery.New(transaction).CountOwnedAsset(ctx, dbquery.CountOwnedAssetParams{ID: entry.AssetID, OwnerID: owner})
		count = int(countOwnedAssetRow)
		if err != nil {
			return err
		}
		if count != 1 {
			return fault.Fault("asset_missing", "An image is unavailable. Reattach it before saving.")
		}
		_, attachAssetErr := dbquery.New(transaction).AttachAsset(ctx, dbquery.AttachAssetParams{PageID: pageID, AssetID: entry.AssetID})
		if attachAssetErr != nil {
			return attachAssetErr
		}
	}

	return nil
}
