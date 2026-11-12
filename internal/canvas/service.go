package canvas

import (
	"context"
	"mindflow/internal/canvas/storage"
	"mindflow/internal/editor"

	imageasset "mindflow/internal/image"
)

// Service owns canvas documents and save receipts.
// NewService composes the collaborators used by document saves.
// CopyAssetsWithin preserves image references while copying a page.
// Cleanup removes expired idempotency receipts.

// WorkspaceActivity records workspace recency in the save transaction.
type WorkspaceActivity interface {
	TouchPage(ctx context.Context, transaction *storage.Transaction, pageID string) error
}

// Service owns this domain's behavior and persistence.
type Service struct {
	repository *storage.Repository
	images     *imageasset.Service
	editors    *editor.Service
	activity   WorkspaceActivity
	publish    func(string, string, string)
}

// NewService constructs the domain service with explicit collaborators.
func NewService(repository *storage.Repository, images *imageasset.Service, editors *editor.Service, activity WorkspaceActivity, publish func(string, string, string)) *Service {
	return &Service{repository: repository, images: images, editors: editors, activity: activity, publish: publish}
}

// CopyAssetsWithin preserves image references in a caller-owned transaction.
func (a *Service) CopyAssetsWithin(ctx context.Context, transaction *storage.Transaction, target, source string) error {
	_, err := a.repository.Within(transaction).CopyPageAssets(ctx, storage.CopyPageAssetsParams{PageID: target, PageID_2: source})

	return err
}

// Cleanup removes this domain's expired records.
func (a *Service) Cleanup(ctx context.Context) error { return a.repository.DeleteExpiredReceipts(ctx) }
