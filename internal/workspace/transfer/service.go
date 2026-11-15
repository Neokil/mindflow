package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace"
	imageasset "mindflow/internal/workspace/image"
	"mindflow/internal/workspace/onenote"
	"mindflow/internal/workspace/page"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/pdf"
	"mindflow/internal/workspace/storage"
	"mindflow/internal/workspace/template"
	"sync"
)

// Service composes permission-aware parent storage, media, page and template domains.
type Service struct {
	onenote            *onenote.Codec
	repository         *storage.Repository
	workspaces         *workspace.Service
	pages              *page.Service
	images             *imageasset.Service
	pdfs               *pdf.Service
	templates          *template.Service
	publish            func(string, string, string)
	processing         sync.Mutex
	externalProcessing sync.Mutex
}

// NewService supplies the explicit collaborators used by atomic transfers.
func NewService(repository *storage.Repository, workspaces *workspace.Service, pages *page.Service, images *imageasset.Service, pdfs *pdf.Service, templates *template.Service, publish func(string, string, string), codecPath string) *Service {
	return &Service{onenote: onenote.NewCodec(codecPath), repository: repository, workspaces: workspaces, pages: pages, images: images, pdfs: pdfs, templates: templates, publish: publish}
}

func busy() error {
	return &workspace.Error{Code: "metadata_conflict", Message: "Another import or export is running. Please retry shortly."}
}

// Snapshot returns a consistent authorized view of saved pages for visual rendering.
func (a *Service) Snapshot(ctx context.Context, actor session.Principal, workspaceID string, selection Selection) (Snapshot, error) {
	if !a.processing.TryLock() {
		return Snapshot{}, busy()
	}
	defer a.processing.Unlock()
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	defer func() { _ = transaction.Rollback() }()

	return a.snapshotWithin(ctx, transaction, actor, workspaceID, selection)
}

// Export creates a self-contained native file with only the selected content's media.
func (a *Service) Export(ctx context.Context, actor session.Principal, workspaceID string, selection Selection) ([]byte, error) {
	if !a.processing.TryLock() {
		return nil, busy()
	}
	defer a.processing.Unlock()
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = transaction.Rollback() }()
	snapshot, err := a.snapshotWithin(ctx, transaction, actor, workspaceID, selection)
	if err != nil {
		return nil, err
	}
	kind := "workspace"
	if len(selection.PageIDs) > 0 {
		kind = kindPages
	}
	bundle := Bundle{Format: formatName, Version: formatVersion, Kind: kind, Name: snapshot.Name, Pages: snapshot.Pages, Templates: []Template{}, Assets: []Asset{}}
	queries := a.repository.Within(transaction)
	if selection.IncludeTemplates {
		rows, readErr := queries.TransferTemplates(ctx, workspaceID)
		if readErr != nil {
			return nil, readErr
		}
		for _, row := range rows {
			var elements []canvas.Element
			err = json.Unmarshal([]byte(row.Elements), &elements)
			if err != nil {
				return nil, fmt.Errorf("decode export template: %w", err)
			}
			bundle.Templates = append(bundle.Templates, Template{Name: row.Name, Elements: elements})
		}
	}
	portableLinks(&bundle)
	contents := map[string][]byte{}
	assets := map[string]string{}
	size := 0
	collect := func(elements []canvas.Element) error {
		return collectAssets(ctx, queries, actor.ID, workspaceID, elements, assets, contents, &bundle.Assets, &size)
	}
	for index := range bundle.Pages {
		err = collect(bundle.Pages[index].Document.Elements)
		if err != nil {
			return nil, err
		}
	}
	for index := range bundle.Templates {
		err = collect(bundle.Templates[index].Elements)
		if err != nil {
			return nil, err
		}
	}
	// All database reads are complete before ZIP compression starts.
	err = transaction.Commit()
	if err != nil {
		return nil, fmt.Errorf("commit export snapshot: %w", err)
	}

	return encode(bundle, contents)
}

func (a *Service) snapshotWithin(ctx context.Context, transaction *storage.Transaction, actor session.Principal, workspaceID string, selection Selection) (Snapshot, error) {
	queries := a.repository.Within(transaction)
	_, err := queries.RequireWorkspace(ctx, workspaceID, actor.ID, false)
	if err != nil {
		return Snapshot{}, repositoryError(err)
	}
	name, err := queries.TransferWorkspace(ctx, workspaceID)
	if err != nil {
		return Snapshot{}, err
	}
	if len(selection.PageIDs) > maxPages {
		return Snapshot{}, invalid("Export up to 100 pages at a time.")
	}
	selected := map[string]bool{}
	for _, resourceID := range selection.PageIDs {
		if resourceID == "" || selected[resourceID] {
			return Snapshot{}, invalid("Select each page only once.")
		}
		selected[resourceID] = true
	}
	rows, err := queries.TransferPages(ctx, workspaceID)
	if err != nil {
		return Snapshot{}, err
	}
	result := Snapshot{Name: name, Pages: []Page{}}
	documentBytes, elementCount := 0, 0
	for _, row := range rows {
		if len(selected) > 0 && !selected[row.ID] {
			continue
		}
		if len(row.Document) > maxPageBytes {
			return Snapshot{}, invalid("An exported page exceeds 16 MiB.")
		}
		var document canvas.Document
		err = json.Unmarshal([]byte(row.Document), &document)
		if err != nil {
			return Snapshot{}, fmt.Errorf("decode export page: %w", err)
		}
		err = canvas.ValidateDocument(document)
		if err != nil {
			return Snapshot{}, err
		}
		documentBytes += len(row.Document)
		elementCount += len(document.Elements)
		if documentBytes > maxManifestBytes || elementCount > maxElements || len(result.Pages) >= maxPages {
			return Snapshot{}, invalid("This export exceeds the page or document limits. Select fewer pages.")
		}
		result.Pages = append(result.Pages, Page{ID: row.ID, Name: row.Name, Document: document})
	}
	if len(result.Pages) > maxPages {
		return Snapshot{}, invalid("Export up to 100 pages at a time. Select fewer pages.")
	}
	if len(selected) > 0 && len(selected) != len(result.Pages) {
		return Snapshot{}, invalid("A selected page is not available in this workspace.")
	}

	return result, nil
}

func collectAssets(ctx context.Context, queries *storage.Queries, userID, workspaceID string, elements []canvas.Element, portableIDs map[string]string, contents map[string][]byte, manifestAssets *[]Asset, size *int) error {
	for index := range elements {
		element := &elements[index]
		if element.Type != elementImage && element.Type != elementPDF {
			continue
		}
		originalID := element.AssetID
		if portableID, exists := portableIDs[originalID]; exists {
			element.AssetID = portableID

			continue
		}
		if len(portableIDs) >= maxAssets {
			return invalid("An export can contain up to 1,000 media assets.")
		}
		allowed, readErr := queries.TemplateAssetAllowed(ctx, storage.TemplateAssetAllowedParams{WorkspaceID: workspaceID, AssetID: originalID, UserID: userID})
		if readErr != nil {
			return readErr
		}
		if allowed != 1 {
			return invalid("An export media asset is no longer available in this workspace.")
		}
		original, readErr := queries.TransferAsset(ctx, originalID)
		if readErr != nil {
			return readErr
		}
		*size += len(original.Original)
		if *size > maxExpandedBytes {
			return invalid("This export exceeds 128 MiB of media. Select fewer pages.")
		}
		portableID := fmt.Sprintf("a%d", len(portableIDs)+1)
		asset := Asset{ID: portableID, MIME: original.Mime, Preview: original.Mime == mediaPDF}
		contents[assetPath(portableID)] = original.Original
		if asset.Preview {
			preview, previewErr := queries.TransferPDFPreview(ctx, originalID)
			if previewErr != nil {
				return previewErr
			}
			*size += len(preview)
			contents[previewPath(portableID)] = preview
		}
		portableIDs[originalID] = portableID
		element.AssetID = portableID
		*manifestAssets = append(*manifestAssets, asset)
	}

	return nil
}

// portableLinks prevents exported links from referring to content outside the bundle.
func portableLinks(bundle *Bundle) {
	targets := map[canvas.ElementLink]canvas.ElementLink{}
	for i, entry := range bundle.Pages {
		id := fmt.Sprintf("p%d", i+1)
		for _, element := range entry.Document.Elements {
			targets[canvas.ElementLink{PageID: entry.ID, ElementID: element.ID}] = canvas.ElementLink{PageID: id, ElementID: element.ID}
		}
		bundle.Pages[i].ID = id
	}
	for _, entry := range bundle.Pages {
		canvas.RemapLinks(entry.Document.Elements, targets, false)
	}
	for _, entry := range bundle.Templates {
		canvas.RemapLinks(entry.Elements, targets, false)
	}
}
