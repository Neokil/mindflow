package image

import (
	"context"
	"database/sql"
	"math"
	"sync"
	// NewService constructs image storage and preview processing.
	// CountOwnedWithin checks an original's ownership in a caller transaction.
	// DeleteWorkspaceAssetsWithin removes assets exclusive to the deleted workspace.
	// DeletePageAssetsWithin removes assets exclusive to the deleted page.
	// Cleanup removes orphan preview jobs and unreferenced stale uploads.
)

// Service owns this domain's behavior and persistence.
type Service struct {
	db      *sql.DB
	mediaMu sync.Mutex
	publish func(string, string, string)
}

// NewService constructs the domain service with explicit collaborators.
func NewService(connection *sql.DB, publish func(string, string, string)) *Service {
	return &Service{db: connection, publish: publish}
}
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// CountOwnedWithin checks image ownership in the caller transaction.
func (a *Service) CountOwnedWithin(ctx context.Context, transaction *sql.Tx, owner, id string) (int64, error) {
	return New(transaction).CountOwnedAsset(ctx, CountOwnedAssetParams{ID: id, OwnerID: owner})
}

// DeleteWorkspaceAssetsWithin removes images referenced only by the deleted workspace.
func (a *Service) DeleteWorkspaceAssetsWithin(ctx context.Context, transaction *sql.Tx, owner, id string) (sql.Result, error) {
	return New(transaction).DeleteWorkspaceAssets(ctx, DeleteWorkspaceAssetsParams{OwnerID: owner, WorkspaceID: id, WorkspaceID_2: id})
}

// DeletePageAssetsWithin removes images referenced only by the deleted page.
func (a *Service) DeletePageAssetsWithin(ctx context.Context, transaction *sql.Tx, owner, id string) (sql.Result, error) {
	return New(transaction).DeletePageAssets(ctx, DeletePageAssetsParams{OwnerID: owner, PageID: id, PageID_2: id})
}

// Cleanup removes this domain's expired records.
func (a *Service) Cleanup(ctx context.Context) error {
	queries := New(a.db)
	err := queries.DeleteOrphanVariants(ctx)
	if err != nil {
		return err
	}

	return queries.DeleteUnusedAssets(ctx)
}
