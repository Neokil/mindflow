package canvas

import (
	"context"
	"encoding/json"
	"math"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/storage"
	"time"
)

// GetDocument loads content after checking page ownership.
func (a *Service) GetDocument(ctx context.Context, actor session.Principal, resourceID string) (_ PageSnapshot, err error) {
	defer func() { err = repositoryError(err) }()
	var raw string
	var revision, metadata int
	var name, workspace string
	getDocumentRow, err := a.repository.CanvasGetDocument(ctx, storage.CanvasGetDocumentParams{ID: resourceID, OwnerID: actor.ID})
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
func (a *Service) SaveDocument(ctx context.Context, actor session.Principal, resourceID string, command SaveDocumentCommand) (_ SaveReceipt, err error) {
	defer func() { err = repositoryError(err) }()

	if len(command.MutationID) < 8 || len(command.MutationID) > maxIdentifierLength || len(command.ClientID) < 8 || len(command.ClientID) > maxIdentifierLength {
		return SaveReceipt{}, invalid("Invalid save identity.")
	}
	validateDocumentErr := ValidateDocument(command.Document)
	if validateDocumentErr != nil {
		return SaveReceipt{}, validateDocumentErr
	}
	raw, err := json.Marshal(command.Document)
	if err != nil {
		return SaveReceipt{}, wrap("encode document", err)
	}
	if len(raw) > 16<<defaultGridSpacing {
		return SaveReceipt{}, failure("document_limit", "The page exceeds the 16 MB document limit.")
	}
	requestRaw, err := json.Marshal(command)
	if err != nil {
		return SaveReceipt{}, wrap("encode save request", err)
	}
	hash := secretDigest(string(requestRaw))
	page := actor
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return SaveReceipt{}, wrap("canvas_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	_, accessErr := a.repository.Within(transaction).RequirePage(ctx, resourceID, actor.ID, true)
	if accessErr != nil {
		return SaveReceipt{}, accessErr
	}
	var revision int
	ownedPageRevisionRow, err := a.repository.Within(transaction).CanvasOwnedPageRevision(ctx, storage.CanvasOwnedPageRevisionParams{ID: resourceID, OwnerID: page.ID})
	revision = ownedPageRevisionRow
	if err != nil {
		return SaveReceipt{}, err
	}
	var receiptHash string
	var receiptRev int
	saveReceiptRow, err := a.repository.Within(transaction).CanvasSaveReceipt(ctx, storage.CanvasSaveReceiptParams{PageID: resourceID, ClientID: command.ClientID, MutationID: command.MutationID, ExpiresAt: time.Now().Unix()})
	receiptHash = saveReceiptRow.Digest
	receiptRev = saveReceiptRow.Revision

	if err == nil {
		if hash != receiptHash {
			return SaveReceipt{}, invalid("Mutation ID was reused with different content.")
		}

		return SaveReceipt{Revision: receiptRev, MutationID: command.MutationID}, nil
	}
	if !storage.IsNotFound(err) {
		return SaveReceipt{}, err
	}
	err = a.editors.CheckSaveWithin(ctx, transaction, page.SessionID, resourceID, command.ClientID, command.Epoch, command.BaseRevision, revision)
	if err != nil {
		return SaveReceipt{}, err
	}
	err = a.validateLinksWithin(ctx, transaction, resourceID, command.Document)
	if err != nil {
		return SaveReceipt{}, err
	}
	err = a.ValidateAssetsWithin(ctx, transaction, page.ID, resourceID, command.Document)
	if err != nil {
		return SaveReceipt{}, err
	}
	revision++
	_, err = a.repository.Within(transaction).CanvasSaveDocument(ctx, storage.CanvasSaveDocumentParams{Document: string(raw), Revision: revision, UpdatedAt: time.Now().Unix(), ID: resourceID})
	if err != nil {
		return SaveReceipt{}, err
	}
	err = a.activity.TouchPage(ctx, transaction, resourceID)
	if err != nil {
		return SaveReceipt{}, err
	}
	_, err = a.repository.Within(transaction).CanvasCreateSaveReceipt(ctx, storage.CanvasCreateSaveReceiptParams{PageID: resourceID, ClientID: command.ClientID, MutationID: command.MutationID, Digest: hash, Revision: revision, ExpiresAt: time.Now().Unix() + receiptLifetimeSeconds})
	if err != nil {
		return SaveReceipt{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return SaveReceipt{}, wrap("canvas_repository.go", err)
	}
	a.publish(page.ID, "page", resourceID)
	a.enqueueImageVariants(ctx, page.ID, command.Document)

	return SaveReceipt{Revision: revision, MutationID: command.MutationID}, nil
}

// ValidateAssetsWithin checks image ownership and records canvas references in the caller transaction.
func (a *Service) ValidateAssetsWithin(ctx context.Context, transaction *storage.Transaction, owner, pageID string, d Document) error {
	seen := map[string]bool{}
	for _, entry := range d.Elements {
		if entry.Type != elementImage && entry.Type != elementPDF {
			continue
		}
		var count int
		countOwnedAssetRow, err := a.images.CountOwnedWithin(ctx, transaction, owner, pageID, entry.AssetID)
		count = int(countOwnedAssetRow)
		if err != nil {
			return err
		}
		if count != 1 {
			return failure("asset_missing", "An image is unavailable. Reattach it before saving.")
		}
		mime, mimeErr := a.repository.Within(transaction).CanvasAssetMIME(ctx, entry.AssetID)
		if mimeErr != nil {
			return mimeErr
		}
		if (entry.Type == elementPDF && mime != "application/pdf") || (entry.Type == elementImage && mime != "image/png" && mime != "image/jpeg" && mime != "image/webp") {
			return invalid("Asset format does not match the element.")
		}
		if seen[entry.AssetID] {
			continue
		}
		seen[entry.AssetID] = true
		_, attachAssetErr := a.repository.Within(transaction).CanvasAttachAsset(ctx, storage.CanvasAttachAssetParams{PageID: pageID, AssetID: entry.AssetID})
		if attachAssetErr != nil {
			return attachAssetErr
		}
	}

	return nil
}

func (a *Service) enqueueImageVariants(ctx context.Context, owner string, d Document) {
	targets := imageTargets(d)
	for id, target := range targets {
		_ = a.images.QueueVariant(ctx, owner, id, target[0], target[1])
	}
}

func imageTargets(d Document) map[string][2]float64 {
	targets := map[string][2]float64{}
	for _, entry := range d.Elements {
		if entry.Type == elementImage {
			target := targets[entry.AssetID]
			width, height := entry.Width, entry.Height
			if entry.Crop != nil {
				width /= entry.Crop.Width
				height /= entry.Crop.Height
			}
			target[0] = math.Max(target[0], width)
			target[1] = math.Max(target[1], height)
			targets[entry.AssetID] = target
		}
	}

	return targets
}

// CloneDocument assigns new element and group identities to a copied canvas.
func CloneDocument(raw string) (string, error) {
	var document Document
	unmarshalErr := json.Unmarshal([]byte(raw), &document)
	if unmarshalErr != nil {
		return "", wrap("workspace_repository.go", unmarshalErr)
	}
	document = ClonePages([]PageCopy{{Document: document}}, false)[0]
	data, err := json.Marshal(document)

	return string(data), err
}
