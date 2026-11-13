// Package pdf owns retained PDF uploads and batch PDF-page imports.
// All persistence remains in the parent workspace's storage.
package pdf

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/png"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace"
	"mindflow/internal/workspace/page"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/storage"
	"time"
)

const (
	maxPDFBytes     = 50 << 20
	maxPreviewBytes = 2 << 20
	maxPages        = 100
	previewBucket   = 1024
	maxPreviewSide  = 2048
	identifierBytes = 32
)

// Service composes permission-aware storage and page creation.
type Service struct {
	repository *storage.Repository
	pages      *page.Service
	publish    func(string, string, string)
}

// NewService supplies explicit collaborators.
func NewService(repository *storage.Repository, pages *page.Service, publish func(string, string, string)) *Service {
	return &Service{repository: repository, pages: pages, publish: publish}
}

func invalid(message string) error {
	return &workspace.Error{Code: "validation_failed", Message: message}
}
func newID() string { return hex.EncodeToString(randBytes()) }
func randBytes() []byte {
	data := make([]byte, identifierBytes)
	_, err := rand.Read(data)
	if err != nil {
		panic(err)
	}

	return data
}

// Asset identifies a protected PDF and its first-page preview.
type Asset struct {
	ID     string `json:"id"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Upload stores opaque PDF bytes and a validated PNG thumbnail atomically.
func (a *Service) Upload(ctx context.Context, actor session.Principal, workspaceID string, data, preview []byte) (_ Asset, err error) {
	defer func() { err = repositoryError(err) }()
	prepared, err := Prepare(data, preview)
	if err != nil {
		return Asset{}, err
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Asset{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	access, err := a.repository.Within(transaction).RequireWorkspace(ctx, workspaceID, actor.ID, true)
	if err != nil {
		return Asset{}, err
	}
	asset, err := a.StoreWithin(ctx, transaction, access.OwnerID, workspaceID, prepared)
	if err != nil {
		return Asset{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Asset{}, fmt.Errorf("commit PDF asset: %w", err)
	}

	return asset, nil
}

// Prepared contains a validated retained PDF and its first-page preview.
type Prepared struct {
	data, preview []byte
	width, height int
}

// Prepare validates portable PDF bytes and a bounded PNG preview without persistence.
func Prepare(data, preview []byte) (Prepared, error) {
	if len(data) > maxPDFBytes || !bytes.HasPrefix(data, []byte("%PDF-")) || !bytes.Contains(data[max(0, len(data)-previewBucket):], []byte("%%EOF")) {
		return Prepared{}, invalid("Choose a valid PDF up to 50 MiB.")
	}
	if len(preview) > maxPreviewBytes {
		return Prepared{}, invalid("PDF preview is too large.")
	}
	config, err := png.DecodeConfig(bytes.NewReader(preview))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > maxPreviewSide || config.Height > maxPreviewSide {
		return Prepared{}, invalid("Choose a valid PDF preview up to 2,048 pixels per side.")
	}
	_, err = png.Decode(bytes.NewReader(preview))
	if err != nil {
		return Prepared{}, invalid("The PDF preview could not be decoded.")
	}

	return Prepared{data: data, preview: preview, width: config.Width, height: config.Height}, nil
}

// StoreWithin retains a prepared PDF inside the caller's atomic workspace transaction.
func (a *Service) StoreWithin(ctx context.Context, transaction *storage.Transaction, ownerID, workspaceID string, prepared Prepared) (Asset, error) {
	data, preview := prepared.data, prepared.preview
	config := struct{ Width, Height int }{prepared.width, prepared.height}
	resourceID := newID()
	digest := sha256.Sum256(data)
	err := a.repository.CreateImportedAssetWithin(ctx, transaction, workspaceID, storage.ImageCreateAssetParams{ID: resourceID, OwnerID: ownerID, Original: data, Mime: "application/pdf", Width: config.Width, Height: config.Height, Digest: hex.EncodeToString(digest[:]), CreatedAt: time.Now().Unix()})
	if err != nil {
		return Asset{}, err
	}
	_, err = a.repository.Within(transaction).ImageCreateVariant(ctx, storage.ImageCreateVariantParams{AssetID: resourceID, Bucket: previewBucket, Mime: "image/png", Data: preview, Width: config.Width, Height: config.Height})
	if err != nil {
		return Asset{}, err
	}

	return Asset{ID: resourceID, Width: config.Width, Height: config.Height}, nil
}

// PageImage describes a previously uploaded rasterized PDF page.
type PageImage struct {
	Name    string  `json:"name"`
	AssetID string  `json:"assetId"`
	Width   float64 `json:"width"`
	Height  float64 `json:"height"`
}

// ImportPages creates all requested pages in one transaction.
func (a *Service) ImportPages(ctx context.Context, actor session.Principal, workspaceID string, images []PageImage) (_ []string, err error) {
	defer func() { err = repositoryError(err) }()
	if len(images) == 0 || len(images) > maxPages {
		return nil, invalid("Import between 1 and 100 PDF pages.")
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = transaction.Rollback() }()
	_, err = a.repository.Within(transaction).RequireWorkspace(ctx, workspaceID, actor.ID, true)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(images))
	for _, entry := range images {
		document := canvas.NewDocument()
		document.Elements = []canvas.Element{{ID: newID(), Type: "image", Width: entry.Width, Height: entry.Height, AssetID: entry.AssetID}}
		resourceID, createErr := a.pages.CreateDocumentWithin(ctx, transaction, actor.ID, workspaceID, entry.Name, document)
		if createErr != nil {
			return nil, createErr
		}
		ids = append(ids, resourceID)
	}
	err = transaction.Commit()
	if err != nil {
		return nil, fmt.Errorf("commit PDF pages: %w", err)
	}
	a.publish(actor.ID, "workspace", workspaceID)

	return ids, nil
}
