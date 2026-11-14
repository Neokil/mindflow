// Package trash owns the 30-day recovery and permanent expiry of workspace content.
package trash

import (
	"context"
	"fmt"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace"
	imageasset "mindflow/internal/workspace/image"
	"mindflow/internal/workspace/storage"
	"sort"
	"time"
)

const (
	kindWorkspace = "workspace"
	retention     = 30 * 24 * time.Hour
)

// Item is a restorable page or workspace; child pages of a trashed workspace remain with it.
type Item struct {
	Kind          string `json:"kind"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	WorkspaceID   string `json:"workspaceId"`
	WorkspaceName string `json:"workspaceName"`
	DeletedAt     int64  `json:"deletedAt"`
	PurgeAt       int64  `json:"purgeAt"`
}

// Service uses parent-domain storage and image cleanup without cross-domain queries.
type Service struct {
	repository *storage.Repository
	images     *imageasset.Service
	publish    func(string, string, string)
	now        func() time.Time
}

// NewService composes recovery and cleanup collaborators.
func NewService(repository *storage.Repository, images *imageasset.Service, publish func(string, string, string), now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}

	return &Service{repository: repository, images: images, publish: publish, now: now}
}

// List exposes owned deleted workspaces and deleted pages in editable active workspaces.
func (a *Service) List(ctx context.Context, actor session.Principal) ([]Item, error) {
	cutoff := a.now().Add(-retention).Unix()
	result := []Item{}
	workspaces, err := a.repository.TrashWorkspaces(ctx, storage.TrashWorkspacesParams{OwnerID: actor.ID, Cutoff: cutoff})
	if err != nil {
		return nil, err
	}
	for _, row := range workspaces {
		result = append(result, Item{Kind: kindWorkspace, ID: row.ID, Name: row.Name, WorkspaceID: row.ID, WorkspaceName: row.Name, DeletedAt: row.DeletedAt, PurgeAt: row.DeletedAt + int64(retention/time.Second)})
	}
	pages, err := a.repository.TrashPages(ctx, storage.TrashPagesParams{UserID: actor.ID, Cutoff: cutoff})
	if err != nil {
		return nil, err
	}
	for _, row := range pages {
		result = append(result, Item{Kind: "page", ID: row.ID, Name: row.Name, WorkspaceID: row.WorkspaceID, WorkspaceName: row.WorkspaceName, DeletedAt: row.DeletedAt, PurgeAt: row.DeletedAt + int64(retention/time.Second)})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].DeletedAt == result[j].DeletedAt {
			return result[i].ID < result[j].ID
		}

		return result[i].DeletedAt > result[j].DeletedAt
	})

	return result, nil
}

// Restore restores content atomically without reviving stale editor leases.
func (a *Service) Restore(ctx context.Context, actor session.Principal, kind, resourceID string) (Completed, error) {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	var count int64
	workspaceID := resourceID
	cutoff := a.now().Add(-retention).Unix()
	if kind == kindWorkspace {
		count, err = queries.TrashRestoreWorkspace(ctx, storage.TrashRestoreWorkspaceParams{ID: resourceID, OwnerID: actor.ID, Cutoff: cutoff, UpdatedAt: a.now().Unix()})
	} else {
		workspaceID, err = queries.TrashPageParent(ctx, storage.TrashPageParentParams{ID: resourceID, Cutoff: cutoff})
		if err != nil {
			return Completed{}, unavailable(err)
		}
		_, err = queries.RequireWorkspace(ctx, workspaceID, actor.ID, true)
		if err != nil {
			return Completed{}, unavailable(err)
		}
		count, err = queries.TrashRestorePage(ctx, storage.TrashRestorePageParams{ID: resourceID, Cutoff: cutoff})
	}
	if err != nil {
		return Completed{}, err
	}
	if count != 1 {
		return Completed{}, unavailable(nil)
	}
	err = transaction.Commit()
	if err != nil {
		return Completed{}, fmt.Errorf("commit trash restoration: %w", err)
	}
	a.publish(actor.ID, kindWorkspace, workspaceID)

	return Completed{OK: true}, nil
}

// Cleanup permanently removes expired content and exclusive media in one transaction.
func (a *Service) Cleanup(ctx context.Context) error {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	cutoff := a.now().Add(-retention).Unix()
	workspaces, err := queries.TrashExpiredWorkspaces(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, row := range workspaces {
		_, err = a.images.DeleteWorkspaceAssetsWithin(ctx, transaction, row.OwnerID, row.ID)
		if err != nil {
			return err
		}
		_, err = queries.WorkspaceDeleteWorkspace(ctx, storage.WorkspaceDeleteWorkspaceParams(row))
		if err != nil {
			return err
		}
	}
	pages, err := queries.TrashExpiredPages(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, resourceID := range pages {
		_, err = a.images.DeletePageAssetsWithin(ctx, transaction, resourceID)
		if err != nil {
			return err
		}
		_, err = queries.PageDeletePage(ctx, resourceID)
		if err != nil {
			return err
		}
	}

	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("commit trash expiry: %w", err)
	}

	return nil
}

// Completed acknowledges a successful restoration.
type Completed struct {
	OK bool `json:"ok"`
}

func unavailable(err error) error {
	if err == nil || storage.IsNotFound(err) {
		return &workspace.Error{Code: "not_found", Message: "This trash item is unavailable or its 30-day retention has expired."}
	}

	return err
}
