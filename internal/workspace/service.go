package workspace

import (
	"context"
	"database/sql"
	"mindflow/internal/page"
	"time"

	imageasset "mindflow/internal/image"
)

// NewService constructs workspace operations with explicit page and image collaborators.
// Activity records workspace recency without depending on page or canvas services.
// TouchPage updates the parent recency in a document-save transaction.

// Service owns this domain's behavior and persistence.
type Service struct {
	db      *sql.DB
	pages   *page.Service
	images  *imageasset.Service
	publish func(string, string, string)
}

// NewService constructs the domain service with explicit collaborators.
func NewService(connection *sql.DB, pages *page.Service, images *imageasset.Service, publish func(string, string, string)) *Service {
	return &Service{db: connection, pages: pages, images: images, publish: publish}
}

// Activity records workspace recency without a dependency on page or canvas services.
type Activity struct{}

// TouchPage updates parent workspace recency in the save transaction.
func (Activity) TouchPage(ctx context.Context, transaction *sql.Tx, pageID string) error {
	_, err := New(transaction).TouchWorkspace(ctx, TouchWorkspaceParams{UpdatedAt: time.Now().Unix(), ID: pageID})

	return err
}
