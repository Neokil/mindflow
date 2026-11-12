package canvas

import (
	"context"
	"database/sql"
	"mindflow/internal/editor"
	imageasset "mindflow/internal/image"
	// Service owns canvas documents and save receipts.
	// NewService composes the collaborators used by document saves.
	// CopyAssetsWithin preserves image references while copying a page.
	// Cleanup removes expired idempotency receipts.
)

// WorkspaceActivity records workspace recency in the save transaction.
type WorkspaceActivity interface {
	TouchPage(ctx context.Context, transaction *sql.Tx, pageID string) error
}

// Service owns this domain's behavior and persistence.
type Service struct {
	db       *sql.DB
	images   *imageasset.Service
	editors  *editor.Service
	activity WorkspaceActivity
	publish  func(string, string, string)
}

// NewService constructs the domain service with explicit collaborators.
func NewService(connection *sql.DB, images *imageasset.Service, editors *editor.Service, activity WorkspaceActivity, publish func(string, string, string)) *Service {
	return &Service{db: connection, images: images, editors: editors, activity: activity, publish: publish}
}

// CopyAssetsWithin preserves image references in a caller-owned transaction.
func (a *Service) CopyAssetsWithin(ctx context.Context, transaction *sql.Tx, target, source string) error {
	_, err := New(transaction).CopyPageAssets(ctx, CopyPageAssetsParams{PageID: target, PageID_2: source})

	return err
}

// Cleanup removes this domain's expired records.
func (a *Service) Cleanup(ctx context.Context) error { return New(a.db).DeleteExpiredReceipts(ctx) }
