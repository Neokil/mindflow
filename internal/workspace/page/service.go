package page

import (
	"context"
	"encoding/json"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/storage"

	imageasset "mindflow/internal/workspace/image"
)

// Service owns page metadata, ordering and page lifecycle operations.
// NewService supplies content and image collaborators for page operations.
// CreateBlankWithin inserts a blank page in a caller transaction.
// CopyWorkspaceWithin duplicates the source workspace's pages in order.
// CountWithin counts the parent's pages in the caller transaction.
// PositionWithin updates one page position in the caller transaction.

// Created identifies a newly created domain resource.
type Created struct {
	ID string `json:"id"`
}

// Service owns this domain's behavior and persistence.
type Service struct {
	repository *storage.Repository
	canvas     *canvas.Service
	images     *imageasset.Service
	publish    func(string, string, string)
}

// NewService constructs the domain service with explicit collaborators.
func NewService(repository *storage.Repository, content *canvas.Service, images *imageasset.Service, publish func(string, string, string)) *Service {
	return &Service{repository: repository, canvas: content, images: images, publish: publish}
}

// CreateBlankWithin creates a page with blank canvas content in the caller transaction.
func (a *Service) CreateBlankWithin(ctx context.Context, transaction *storage.Transaction, workspaceID, name string) (string, error) {
	raw, err := json.Marshal(canvas.NewDocument())
	if err != nil {
		return "", wrap("encode blank canvas", err)
	}

	return a.CreateWithin(ctx, transaction, workspaceID, name, string(raw))
}

// CopyWorkspaceWithin copies pages in order using the caller's transaction.
func (a *Service) CopyWorkspaceWithin(ctx context.Context, transaction *storage.Transaction, source, target string) (map[canvas.ElementLink]canvas.ElementLink, error) {
	pages, err := a.repository.Within(transaction).PagePagesToCopy(ctx, source)
	if err != nil {
		return nil, err
	}
	copies := make([]canvas.PageCopy, len(pages))
	for pageIndex, entry := range pages {
		raw, readErr := a.repository.Within(transaction).PagePageDocument(ctx, entry.ID)
		if readErr != nil {
			return nil, readErr
		}
		var document canvas.Document
		decodeErr := json.Unmarshal([]byte(raw), &document)
		if decodeErr != nil {
			return nil, wrap("decode copied page", decodeErr)
		}
		copies[pageIndex] = canvas.PageCopy{SourceID: entry.ID, TargetID: newID(), Document: document}
	}
	documents := canvas.ClonePages(copies, false)
	for pageIndex, document := range documents {
		err = a.copyDocumentWithin(ctx, transaction, pages[pageIndex].ID, copies[pageIndex].TargetID, target, pages[pageIndex].Name, document)
		if err != nil {
			return nil, err
		}
	}

	targets := map[canvas.ElementLink]canvas.ElementLink{}
	for pageIndex, entry := range copies {
		for j, element := range entry.Document.Elements {
			targets[canvas.ElementLink{PageID: entry.SourceID, ElementID: element.ID}] = canvas.ElementLink{PageID: entry.TargetID, ElementID: documents[pageIndex].Elements[j].ID}
		}
	}

	return targets, nil
}

// CountWithin counts the parent workspace's pages in the caller transaction.
func (a *Service) CountWithin(ctx context.Context, transaction *storage.Transaction, workspaceID string) (int64, error) {
	return a.repository.Within(transaction).PageCountPages(ctx, workspaceID)
}

// PositionWithin updates a page position in the caller transaction.
func (a *Service) PositionWithin(ctx context.Context, transaction *storage.Transaction, index int, id, workspaceID string) (int64, error) {
	return a.repository.PositionWithin(ctx, transaction, index, id, workspaceID)
}

// CreateDocumentWithin validates and attaches imported page content atomically.
func (a *Service) CreateDocumentWithin(ctx context.Context, transaction *storage.Transaction, ownerID, workspaceID, name string, document canvas.Document) (string, error) {
	return a.CreateDocumentWithIDWithin(ctx, transaction, ownerID, workspaceID, newID(), name, document)
}

// CreateDocumentWithIDWithin creates imported content with an identity allocated by the caller.
func (a *Service) CreateDocumentWithIDWithin(ctx context.Context, transaction *storage.Transaction, ownerID, workspaceID, resourceID, name string, document canvas.Document) (string, error) {
	err := ValidateName(name)
	if err != nil {
		return "", err
	}
	err = canvas.ValidateDocument(document)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return "", wrap("encode imported canvas", err)
	}
	if len(raw) > maxImportedDocumentBytes {
		return "", invalid("An imported page exceeds the 16 MiB document limit.")
	}
	_, err = a.createWithIDWithin(ctx, transaction, workspaceID, resourceID, name, string(raw))
	if err != nil {
		return "", err
	}
	err = a.canvas.ValidateAssetsWithin(ctx, transaction, ownerID, resourceID, document)

	return resourceID, err
}

const maxImportedDocumentBytes = 16 << 20

// copyDocumentWithin copies trusted saved content and its existing media associations.
func (a *Service) copyDocumentWithin(ctx context.Context, transaction *storage.Transaction, source, target, workspaceID, name string, document canvas.Document) error {
	raw, err := json.Marshal(document)
	if err != nil {
		return wrap("encode copied page", err)
	}
	_, err = a.createWithIDWithin(ctx, transaction, workspaceID, target, name, string(raw))
	if err != nil {
		return err
	}

	return a.canvas.CopyAssetsWithin(ctx, transaction, target, source)
}
