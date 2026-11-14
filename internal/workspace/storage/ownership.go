package storage

import (
	"context"
	"database/sql"
)

// TransferAssetsWithin reassigns exclusive media and clones media retained by other workspaces.
func (a *Repository) TransferAssetsWithin(ctx context.Context, transaction *Transaction, workspaceID, newOwner string, identifier func() string) (map[string]string, error) {
	queries := a.Within(transaction)
	rows, err := queries.OwnershipAssets(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	mapping := map[string]string{}
	scope := sql.NullString{String: workspaceID, Valid: true}
	for _, row := range rows {
		if !row.SharedElsewhere.Bool {
			err = queries.OwnershipReassignAsset(ctx, OwnershipReassignAssetParams{ID: row.ID, OwnerID: newOwner, WorkspaceID: scope})
			if err != nil {
				return nil, err
			}
			err = queries.OwnershipMoveJobs(ctx, OwnershipMoveJobsParams{AssetID: row.ID, UserID: newOwner})
			if err != nil {
				return nil, err
			}

			continue
		}
		copiedID := identifier()
		err = queries.OwnershipCloneAsset(ctx, OwnershipCloneAssetParams{NewID: copiedID, SourceID: row.ID, OwnerID: newOwner, WorkspaceID: scope})
		if err != nil {
			return nil, err
		}
		err = queries.OwnershipCloneVariants(ctx, OwnershipCloneVariantsParams{NewID: copiedID, SourceID: row.ID})
		if err != nil {
			return nil, err
		}
		err = queries.OwnershipPageAssets(ctx, OwnershipPageAssetsParams{AssetID: copiedID, AssetID_2: row.ID, WorkspaceID: workspaceID})
		if err != nil {
			return nil, err
		}
		err = queries.OwnershipTemplateAssets(ctx, OwnershipTemplateAssetsParams{AssetID: copiedID, AssetID_2: row.ID, WorkspaceID: workspaceID})
		if err != nil {
			return nil, err
		}
		err = queries.OwnershipDetachOriginal(ctx, OwnershipDetachOriginalParams{ID: row.ID, WorkspaceID: scope})
		if err != nil {
			return nil, err
		}
		mapping[row.ID] = copiedID
	}

	return mapping, nil
}
