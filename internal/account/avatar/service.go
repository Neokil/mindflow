// Package avatar owns normalized account profile images.
package avatar

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/jpeg" // Register the supported JPEG decoder.
	"image/png"
	"mindflow/internal/account"
	"mindflow/internal/account/storage"
	"sync"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // Register the supported WebP decoder.
)

// Service stores one bounded, metadata-free avatar per account.
type Service struct {
	repository *storage.Repository
	processing sync.Mutex
}

// NewService supplies parent account persistence.
func NewService(repository *storage.Repository) *Service { return &Service{repository: repository} }

// Get returns a visible active account's normalized PNG.
func (a *Service) Get(ctx context.Context, userID string) ([]byte, error) {
	content, err := a.repository.AvatarContent(ctx, userID)
	if storage.IsNotFound(err) {
		return nil, &account.Error{Code: "not_found", Message: "Avatar unavailable."}
	}

	return content, err
}

// GetVersioned returns an immutable representation only for the current version.
func (a *Service) GetVersioned(ctx context.Context, userID, version string) ([]byte, error) {
	content, err := a.repository.AvatarVersionedContent(ctx, storage.AvatarVersionedContentParams{UserID: userID, AvatarVersion: version})
	if storage.IsNotFound(err) {
		return nil, &account.Error{Code: "not_found", Message: "Avatar unavailable."}
	}

	return content, err
}

// Set replaces or removes the caller's avatar atomically.
func (a *Service) Set(ctx context.Context, userID string, raw []byte) (account.Completed, error) {
	a.processing.Lock()
	defer a.processing.Unlock()
	var content []byte
	if len(raw) > 0 {
		var err error
		content, err = Normalize(raw)
		if err != nil {
			return account.Completed{}, err
		}
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return account.Completed{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	version := ""
	if len(content) > 0 {
		err = queries.AvatarSet(ctx, storage.AvatarSetParams{UserID: userID, Content: content})
		token := make([]byte, avatarIdentifierBytes)
		_, _ = rand.Read(token)
		version = hex.EncodeToString(token)
	} else {
		err = queries.AvatarDelete(ctx, userID)
	}
	if err != nil {
		return account.Completed{}, err
	}
	err = queries.AvatarVersion(ctx, storage.AvatarVersionParams{ID: userID, AvatarVersion: version})
	if err != nil {
		return account.Completed{}, err
	}

	err = transaction.Commit()
	if err != nil {
		return account.Completed{}, fmt.Errorf("commit avatar: %w", err)
	}

	return account.Completed{OK: true}, nil
}

// Normalize validates dimensions before decoding and strips metadata by re-encoding.
func Normalize(raw []byte) ([]byte, error) {
	config, kind, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (kind != "png" && kind != "jpeg" && kind != "webp") || len(raw) > maxUploadBytes || config.Width <= 0 || config.Height <= 0 || config.Width > maxDimension || config.Height > maxDimension || config.Width*config.Height > maxPixels {
		return nil, &account.Error{Code: validationCode, Message: "Choose a PNG, JPEG or WebP avatar up to 2 MB and 8 megapixels."}
	}
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, &account.Error{Code: validationCode, Message: "The avatar could not be decoded."}
	}
	bounds := source.Bounds()
	side := min(bounds.Dx(), bounds.Dy())
	left, top := bounds.Min.X+(bounds.Dx()-side)/halves, bounds.Min.Y+(bounds.Dy()-side)/halves
	target := image.NewNRGBA(image.Rect(0, 0, avatarSize, avatarSize))
	draw.CatmullRom.Scale(target, target.Bounds(), source, image.Rect(left, top, left+side, top+side), draw.Src, nil)
	var output bytes.Buffer
	err = png.Encode(&output, target)
	if err != nil {
		return nil, fmt.Errorf("encode avatar: %w", err)
	}

	return output.Bytes(), nil
}

const (
	avatarSize     = 128
	maxDimension   = 8192
	maxPixels      = 8000000
	maxUploadBytes = 2 << 20
)

const (
	validationCode        = "validation_failed"
	avatarIdentifierBytes = 16
	halves                = 2
)
