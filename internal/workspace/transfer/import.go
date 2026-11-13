package transfer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace"
	imageasset "mindflow/internal/workspace/image"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/pdf"
	"mindflow/internal/workspace/storage"
	"mindflow/internal/workspace/template"
	"strings"
)

// Import validates all content, then creates private content or appends pages atomically.
func (a *Service) Import(ctx context.Context, actor session.Principal, reader io.Reader, name, targetID string) (Result, error) {
	if !a.processing.TryLock() {
		return Result{}, busy()
	}
	defer a.processing.Unlock()
	if targetID != "" {
		_, err := a.repository.RequireWorkspace(ctx, targetID, actor.ID, true)
		if err != nil {
			return Result{}, repositoryError(err)
		}
	}
	raw, err := io.ReadAll(io.LimitReader(reader, maxArchiveBytes+1))
	if err != nil {
		return Result{}, invalid("The import file could not be read.")
	}
	bundle, contents, err := Read(raw)
	if err != nil {
		return Result{}, err
	}
	if targetID != "" || strings.TrimSpace(name) == "" {
		name = bundle.Name
	}
	name = portableName(name)
	err = workspace.ValidateName(name)
	if err != nil {
		return Result{}, err
	}
	images, pdfs, err := a.prepareAssets(ctx, bundle, contents)
	if err != nil {
		return Result{}, err
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	ownerID := actor.ID
	workspaceID := targetID
	if targetID == "" {
		workspaceID, err = a.workspaces.CreateWithin(ctx, transaction, actor.ID, name)
	} else {
		var access storage.Access
		access, err = a.repository.Within(transaction).RequireWorkspace(ctx, targetID, actor.ID, true)
		ownerID = access.OwnerID
	}
	if err != nil {
		return Result{}, repositoryError(err)
	}
	assetIDs, err := a.storeAssets(ctx, transaction, ownerID, workspaceID, bundle, images, pdfs)
	if err != nil {
		return Result{}, err
	}
	result := Result{ID: workspaceID, PageCount: len(bundle.Pages), Warnings: []string{}}
	for _, entry := range bundle.Pages {
		document, cloneErr := remapDocument(entry.Document, assetIDs)
		if cloneErr != nil {
			return Result{}, cloneErr
		}
		pageID, createErr := a.pages.CreateDocumentWithin(ctx, transaction, actor.ID, workspaceID, entry.Name, document)
		if createErr != nil {
			return Result{}, createErr
		}
		if result.PageID == "" {
			result.PageID = pageID
		}
		result.ElementCount += len(document.Elements)
	}
	result.TemplateCount, result.Warnings, err = a.importTemplates(ctx, transaction, actor.ID, workspaceID, bundle.Templates, assetIDs)
	if err != nil {
		return Result{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Result{}, fmt.Errorf("commit mindflow import: %w", err)
	}
	a.publish(actor.ID, "workspace", workspaceID)

	return result, nil
}

func (a *Service) prepareAssets(ctx context.Context, bundle Bundle, contents map[string][]byte) (map[string]imageasset.PreparedImage, map[string]pdf.Prepared, error) {
	var err error
	images := map[string]imageasset.PreparedImage{}
	pdfs := map[string]pdf.Prepared{}
	for _, asset := range bundle.Assets {
		err = ctx.Err()
		if err != nil {
			return nil, nil, fmt.Errorf("prepare import: %w", err)
		}
		if asset.MIME == mediaPDF {
			pdfs[asset.ID], err = pdf.Prepare(contents[assetPath(asset.ID)], contents[previewPath(asset.ID)])
		} else {
			images[asset.ID], err = a.images.Prepare(contents[assetPath(asset.ID)])
		}
		if err != nil {
			return nil, nil, err
		}
	}

	return images, pdfs, nil
}

func (a *Service) storeAssets(ctx context.Context, transaction *storage.Transaction, ownerID, workspaceID string, bundle Bundle, images map[string]imageasset.PreparedImage, pdfs map[string]pdf.Prepared) (map[string]string, error) {
	assetIDs := map[string]string{}
	for _, asset := range bundle.Assets {
		if asset.MIME == mediaPDF {
			stored, storeErr := a.pdfs.StoreWithin(ctx, transaction, ownerID, workspaceID, pdfs[asset.ID])
			if storeErr != nil {
				return nil, storeErr
			}
			assetIDs[asset.ID] = stored.ID
		} else {
			stored, storeErr := a.images.StoreWithin(ctx, transaction, ownerID, workspaceID, images[asset.ID])
			if storeErr != nil {
				return nil, storeErr
			}
			if stored.MIME != asset.MIME {
				return nil, invalid("Embedded image format does not match its manifest.")
			}
			assetIDs[asset.ID] = stored.ID
		}
	}

	return assetIDs, nil
}

func remapDocument(document canvas.Document, assetIDs map[string]string) (canvas.Document, error) {
	for index := range document.Elements {
		element := &document.Elements[index]
		if element.Type == elementImage || element.Type == elementPDF {
			element.AssetID = assetIDs[element.AssetID]
		}
	}
	encoded, encodeErr := json.Marshal(document)
	if encodeErr != nil {
		return canvas.Document{}, fmt.Errorf("encode imported document: %w", encodeErr)
	}
	cloned, cloneErr := canvas.CloneDocument(string(encoded))
	if cloneErr != nil {
		return canvas.Document{}, cloneErr
	}
	var result canvas.Document
	decodeErr := json.Unmarshal([]byte(cloned), &result)

	if decodeErr != nil {
		return canvas.Document{}, fmt.Errorf("decode imported document: %w", decodeErr)
	}

	return result, nil
}

func (a *Service) importTemplates(ctx context.Context, transaction *storage.Transaction, userID, workspaceID string, entries []Template, assetIDs map[string]string) (int, []string, error) {
	var err error
	warnings := []string{}
	for _, entry := range entries {
		templateName := entry.Name
		for suffix := 2; ; suffix++ {
			count, countErr := a.repository.Within(transaction).TemplateNameExists(ctx, storage.TemplateNameExistsParams{WorkspaceID: workspaceID, Name: templateName})
			if countErr != nil {
				return 0, nil, countErr
			}
			if count == 0 {
				break
			}
			base := []rune(entry.Name)
			if len(base) > templateBaseLength {
				base = base[:templateBaseLength]
			}
			templateName = fmt.Sprintf("%s (imported %d)", string(base), suffix)
		}
		if templateName != entry.Name {
			warnings = append(warnings, fmt.Sprintf("Template %q was imported as %q.", entry.Name, templateName))
		}
		document := canvas.NewDocument()
		document.Elements = entry.Elements
		document, err = remapDocument(document, assetIDs)
		if err != nil {
			return 0, nil, err
		}
		_, err = a.templates.CreateWithin(ctx, transaction, userID, workspaceID, template.CreateCommand{Name: templateName, Elements: document.Elements})
		if err != nil {
			return 0, nil, err
		}
	}

	return len(entries), warnings, nil
}
