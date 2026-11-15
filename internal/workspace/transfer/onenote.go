package transfer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/onenote"
	"mindflow/internal/workspace/page/canvas"
	"slices"
	"strings"
	"time"
)

// ImportOneNote converts external bytes, then applies the same validation and atomic import as native files.
func (a *Service) ImportOneNote(ctx context.Context, actor session.Principal, reader io.Reader, name, target, format string) (Result, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, maxArchiveBytes+1))
	if err != nil {
		return Result{}, invalid("The OneNote file could not be read.")
	}
	if target != "" {
		_, err = a.repository.RequireWorkspace(ctx, target, actor.ID, true)
		if err != nil {
			return Result{}, repositoryError(err)
		}
	}
	data, notes, err := a.onenote.Convert(ctx, "read-"+format, raw)
	if err != nil {
		return Result{}, err
	}
	result, err := a.Import(ctx, actor, bytes.NewReader(data), name, target)
	result.Warnings = append(result.Warnings, notes...)

	return result, err
}

// ExportOneNote translates an authorized consistent snapshot into a native section or notebook package.
func (a *Service) ExportOneNote(ctx context.Context, actor session.Principal, workspaceID string, selection Selection, format string) ([]byte, []string, error) {
	if !a.externalProcessing.TryLock() {
		return nil, nil, busy()
	}
	defer a.externalProcessing.Unlock()
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	selection.IncludeTemplates = false
	raw, err := a.Export(ctx, actor, workspaceID, selection)
	if err != nil {
		return nil, nil, err
	}
	// Image transforms and shapes are normalized before handing the validated archive to the codec.
	raw, notes, err := prepareOneNote(ctx, raw)
	if err != nil {
		return nil, nil, fmt.Errorf("prepare OneNote export: %w", err)
	}
	data, codecNotes, err := a.onenote.Convert(ctx, "write-"+format, raw)

	return data, append(notes, codecNotes...), err
}

func prepareOneNote(ctx context.Context, raw []byte) ([]byte, []string, error) {
	bundle, contents, err := Read(raw)
	if err != nil {
		return nil, nil, err
	}
	assets := map[string]Asset{}
	for _, asset := range bundle.Assets {
		assets[asset.ID] = asset
	}
	used := map[string]bool{}
	notes := []string{}
	budget := mediaBudget{}
	for pageIndex := range bundle.Pages {
		for elementIndex := range bundle.Pages[pageIndex].Document.Elements {
			element := &bundle.Pages[pageIndex].Document.Elements[elementIndex]
			element.GroupID = ""
			element.Locked = false
			element.Link = nil
			if normalizeOneNoteList(element) {
				notes = appendNote(notes, "Lists retain editable text and visible markers; checkboxes are not interactive in OneNote exports.")
			}
			if element.Type == elementPDF {
				id := fmt.Sprintf("pdf-%d-%d", pageIndex, elementIndex)
				contents[assetPath(id)] = contents[previewPath(element.AssetID)]
				assets[id] = Asset{ID: id, MIME: mediaPNG}
				element.Type = elementImage
				element.AssetID = id
				notes = appendNote(notes, "Embedded PDFs become first-page preview images; export PDF separately to retain all pages.")
			}
			fallback := element.Type == "shape" || (element.Type == "text" && element.Rotation != 0) || (element.Type == elementImage && (element.Rotation != 0 || element.Crop != nil))
			if fallback {
				data, replacement, renderErr := prepareOneNoteFallback(ctx, *element, contents[assetPath(element.AssetID)], &budget)
				if renderErr != nil {
					return nil, nil, renderErr
				}

				id := fmt.Sprintf("raster-%d-%d", pageIndex, elementIndex)
				replacement.AssetID = id
				*element = replacement
				assets[id] = Asset{ID: id, MIME: mediaPNG}
				contents[assetPath(id)] = data
				notes = appendNote(notes, "Shapes, rotated text, and cropped or rotated images become individual PNG images. Other text and ink remain editable. Fallback font metrics can differ.")
			}
			if element.Type == elementImage {
				used[element.AssetID] = true
			}
		}
	}
	bundle.Assets = []Asset{}
	for id := range used {
		bundle.Assets = append(bundle.Assets, assets[id])
	}
	data, err := encode(bundle, contents)

	return data, notes, err
}

func appendNote(notes []string, note string) []string {
	if slices.Contains(notes, note) {
		return notes
	}

	return append(notes, note)
}

func normalizeOneNoteList(element *canvas.Element) bool {
	if element.Type != "text" || element.ListStyle == "" {
		return false
	}
	checked := map[int]bool{}
	for _, index := range element.Checked {
		checked[index] = true
	}
	lines := strings.Split(element.Text, "\n")
	for index := range lines {
		prefix := "• "
		switch element.ListStyle {
		case "number":
			prefix = fmt.Sprintf("%d. ", index+1)
		case "check":
			prefix = "☐ "
			if checked[index] {
				prefix = "☑ "
			}
		}
		lines[index] = prefix + lines[index]
	}
	element.Text = strings.Join(lines, "\n")
	element.ListStyle = ""
	element.Checked = nil

	return true
}

func prepareOneNoteFallback(ctx context.Context, element canvas.Element, media []byte, budget *mediaBudget) ([]byte, canvas.Element, error) {
	data, replacement, err := onenote.RasterFallback(ctx, element, media)
	if err != nil {
		return nil, element, err
	}
	err = budget.add(data)
	if err != nil {
		return nil, element, err
	}

	return data, replacement, nil
}
