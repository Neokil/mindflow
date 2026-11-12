package image

import (
	"context"
	"math"
	"mindflow/internal/image/storage"
	"sync"
)

// NewService constructs image storage and preview processing.
// CountOwnedWithin checks an original's ownership in a caller transaction.
// DeleteWorkspaceAssetsWithin removes assets exclusive to the deleted workspace.
// DeletePageAssetsWithin removes assets exclusive to the deleted page.
// Cleanup removes orphan preview jobs and unreferenced stale uploads.

// Service owns this domain's behavior and persistence.
type Service struct {
	repository *storage.Repository
	mediaMu    sync.Mutex
	publish    func(string, string, string)
}

// NewService constructs the domain service with explicit collaborators.
func NewService(repository *storage.Repository, publish func(string, string, string)) *Service {
	return &Service{repository: repository, publish: publish}
}
func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

// CountOwnedWithin checks image ownership in the caller transaction.
func (a *Service) CountOwnedWithin(ctx context.Context, transaction *storage.Transaction, owner, id string) (int64, error) {
	return a.repository.Within(transaction).CountOwnedAsset(ctx, storage.CountOwnedAssetParams{ID: id, OwnerID: owner})
}

// DeleteWorkspaceAssetsWithin removes images referenced only by the deleted workspace.
func (a *Service) DeleteWorkspaceAssetsWithin(ctx context.Context, transaction *storage.Transaction, owner, id string) (int64, error) {
	return a.repository.DeleteWorkspaceAssetsWithin(ctx, transaction, owner, id)
}

// DeletePageAssetsWithin removes images referenced only by the deleted page.
func (a *Service) DeletePageAssetsWithin(ctx context.Context, transaction *storage.Transaction, owner, id string) (int64, error) {
	return a.repository.DeletePageAssetsWithin(ctx, transaction, owner, id)
}

// Cleanup removes this domain's expired records.
func (a *Service) Cleanup(ctx context.Context) error {
	queries := a.repository
	err := queries.DeleteOrphanVariants(ctx)
	if err != nil {
		return err
	}

	return queries.DeleteUnusedAssets(ctx)
}
