package page

import (
	"context"
	"encoding/json"
	"mindflow/internal/canvas"
	"mindflow/internal/page/storage"

	imageasset "mindflow/internal/image"
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
func (a *Service) CopyWorkspaceWithin(ctx context.Context, transaction *storage.Transaction, source, target string) error {
	pages, err := a.repository.Within(transaction).PagesToCopy(ctx, source)
	if err != nil {
		return err
	}
	for _, entry := range pages {
		_, err = a.CopyWithin(ctx, transaction, entry.ID, target, entry.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

// CountWithin counts the parent workspace's pages in the caller transaction.
func (a *Service) CountWithin(ctx context.Context, transaction *storage.Transaction, workspaceID string) (int64, error) {
	return a.repository.Within(transaction).CountPages(ctx, workspaceID)
}

// PositionWithin updates a page position in the caller transaction.
func (a *Service) PositionWithin(ctx context.Context, transaction *storage.Transaction, index int, id, workspaceID string) (int64, error) {
	return a.repository.PositionWithin(ctx, transaction, index, id, workspaceID)
}
