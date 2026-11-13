package butterfly

import (
	"context"
	"io"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace"
	imageasset "mindflow/internal/workspace/image"
	"mindflow/internal/workspace/page"
	"mindflow/internal/workspace/storage"
	"path"
	"strings"
	"sync"
)

// Service imports one bounded archive at a time into the caller's private content.
type Service struct {
	repository *storage.Repository
	workspaces *workspace.Service
	pages      *page.Service
	images     *imageasset.Service
	publish    func(string, string, string)
	importing  sync.Mutex
}

// NewService supplies explicit parent, page and image collaborators.
func NewService(repository *storage.Repository, workspaces *workspace.Service, pages *page.Service, images *imageasset.Service, publish func(string, string, string)) *Service {
	return &Service{repository: repository, workspaces: workspaces, pages: pages, images: images, publish: publish}
}

// Result identifies the newly created workspace and any conversion limitations.
type Result struct {
	ID           string   `json:"id"`
	PageID       string   `json:"pageId"`
	PageCount    int      `json:"pageCount"`
	ElementCount int      `json:"elementCount"`
	Warnings     []string `json:"warnings"`
}

// Import validates the complete archive and commits all pages/images in one transaction.
func (a *Service) Import(ctx context.Context, actor session.Principal, reader io.Reader, name, filename string) (Result, error) {
	if !a.importing.TryLock() {
		return Result{}, &workspace.Error{Code: "metadata_conflict", Message: "Another workspace import is running. Please retry shortly."}
	}
	defer a.importing.Unlock()
	raw, err := io.ReadAll(io.LimitReader(reader, maxArchiveBytes+1))
	if err != nil {
		return Result{}, invalid("The import file could not be read.")
	}
	archive, err := Read(raw)
	if err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(name) == "" {
		name = archive.Name
	}
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(path.Base(filename), path.Ext(filename))
	}
	if strings.TrimSpace(name) == "" {
		name = "Imported workspace"
	}
	err = workspace.ValidateName(strings.TrimSpace(name))
	if err != nil {
		return Result{}, err
	}
	prepared := make(map[string]imageasset.PreparedImage, len(archive.Images))
	for source, data := range archive.Images {
		original, prepareErr := a.images.Prepare(data)
		if prepareErr != nil {
			return Result{}, prepareErr
		}
		prepared[source] = original
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	workspaceID, err := a.workspaces.CreateWithin(ctx, transaction, actor.ID, name)
	if err != nil {
		return Result{}, err
	}
	assetIDs := make(map[string]string, len(prepared))
	for source, original := range prepared {
		asset, storeErr := a.images.StoreWithin(ctx, transaction, actor.ID, workspaceID, original)
		if storeErr != nil {
			return Result{}, storeErr
		}
		assetIDs[source] = asset.ID
	}
	result := Result{ID: workspaceID, PageCount: len(archive.Pages), Warnings: archive.Warnings}
	for _, entry := range archive.Pages {
		for index := range entry.Document.Elements {
			element := &entry.Document.Elements[index]
			if element.Type == elementImage {
				element.AssetID = assetIDs[element.AssetID]
			}
		}
		pageID, createErr := a.pages.CreateDocumentWithin(ctx, transaction, actor.ID, workspaceID, entry.Name, entry.Document)
		if createErr != nil {
			return Result{}, createErr
		}
		if result.PageID == "" {
			result.PageID = pageID
		}
		result.ElementCount += len(entry.Document.Elements)
	}
	err = wrap("commit Butterfly workspace import", transaction.Commit())
	if err != nil {
		return Result{}, err
	}
	a.publish(actor.ID, "workspace", workspaceID)

	return result, nil
}
