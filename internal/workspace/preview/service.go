// Package preview owns stored first-page workspace thumbnails.
package preview

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/png"
	"mindflow/internal/workspace"
	"mindflow/internal/workspace/storage"
)

// Service uses the parent workspace storage boundary.
type Service struct {
	repository *storage.Repository
	publish    func(string, string, string)
}

// NewService constructs the thumbnail service.
func NewService(repository *storage.Repository, publish func(string, string, string)) *Service {
	return &Service{repository: repository, publish: publish}
}

// State identifies the saved first-page revision to render.
type State struct {
	PageID    string `json:"pageId"`
	Revision  int    `json:"revision"`
	PreviewID string `json:"previewId"`
	Needed    bool   `json:"needed"`
	CanWrite  bool   `json:"canWrite"`
}

// Inspect checks access and whether a stored preview needs refreshing.
func (a *Service) Inspect(ctx context.Context, userID, workspaceID string) (State, error) {
	access, err := a.repository.RequireWorkspace(ctx, workspaceID, userID, false)
	if err != nil {
		return State{}, repositoryError(err)
	}
	first, err := a.repository.PreviewFirstPage(ctx, workspaceID)
	if storage.IsNotFound(err) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}
	current, err := a.repository.PreviewCurrent(ctx, workspaceID)
	if err != nil && !storage.IsNotFound(err) {
		return State{}, err
	}
	state := State{PageID: first.ID, Revision: first.Revision, CanWrite: storage.CanEdit(access.Role), Needed: current.PageID != first.ID || current.Revision != first.Revision}
	if current.PageID == first.ID {
		state.PreviewID = current.ContentID
	}

	return state, nil
}

// Store validates the PNG and rejects stale renders atomically.
func (a *Service) Store(ctx context.Context, userID, workspaceID, pageID string, revision int, raw []byte) (State, error) {
	config, err := png.DecodeConfig(bytes.NewReader(raw))
	if err != nil || len(raw) > maxPreviewBytes || config.Width <= 0 || config.Height <= 0 || config.Width > previewSize || config.Height > previewSize {
		return State{}, invalid("Choose a PNG preview up to 512 pixels and 2 MiB.")
	}
	source, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return State{}, invalid("The preview could not be decoded.")
	}
	var normalized bytes.Buffer
	err = png.Encode(&normalized, source)
	if err != nil {
		return State{}, fmt.Errorf("encode workspace preview: %w", err)
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return State{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	_, err = queries.RequireWorkspace(ctx, workspaceID, userID, true)
	if err != nil {
		return State{}, repositoryError(err)
	}
	first, err := queries.PreviewFirstPage(ctx, workspaceID)
	if err != nil {
		return State{}, repositoryError(err)
	}
	if first.ID != pageID || first.Revision != revision {
		return State{}, &workspace.Error{Code: "revision_conflict", Message: "The first page changed. Refresh its preview."}
	}
	digest := sha256.Sum256(normalized.Bytes())
	contentID := hex.EncodeToString(digest[:])
	err = queries.PreviewStore(ctx, storage.PreviewStoreParams{WorkspaceID: workspaceID, PageID: pageID, Revision: revision, ContentID: contentID, Data: normalized.Bytes(), Empty: int(first.Empty)})
	if err != nil {
		return State{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return State{}, fmt.Errorf("commit workspace preview: %w", err)
	}

	a.publish(userID, "workspace", workspaceID)

	return State{PageID: pageID, Revision: revision, PreviewID: contentID, CanWrite: true}, nil
}

// Get loads a versioned thumbnail only for an accessible current first page.
func (a *Service) Get(ctx context.Context, userID, workspaceID, contentID string) ([]byte, error) {
	_, err := a.repository.RequireWorkspace(ctx, workspaceID, userID, false)
	if err != nil {
		return nil, repositoryError(err)
	}
	current, err := a.repository.PreviewCurrent(ctx, workspaceID)
	if err != nil {
		return nil, repositoryError(err)
	}
	first, err := a.repository.PreviewFirstPage(ctx, workspaceID)
	if err != nil {
		return nil, repositoryError(err)
	}
	if current.ContentID != contentID || current.PageID != first.ID {
		return nil, missing()
	}

	return current.Data, nil
}

func invalid(message string) error {
	return &workspace.Error{Code: "validation_failed", Message: message}
}
func missing() error { return &workspace.Error{Code: "not_found", Message: "Preview unavailable."} }
func repositoryError(err error) error {
	if storage.IsNotFound(err) {
		return missing()
	}

	return err
}

const (
	previewSize     = 512
	maxPreviewBytes = 2 << 20
)
