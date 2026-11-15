package transfer

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/butterfly"
	"mindflow/internal/workspace/page/canvas"
	"time"
)

// ExportButterfly reuses the authorized atomic snapshot and exports only its media.
func (a *Service) ExportButterfly(ctx context.Context, actor session.Principal, workspaceID string, selection Selection) ([]byte, []string, error) {
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
	bundle, contents, err := Read(raw)
	if err != nil {
		return nil, nil, err
	}
	archive := butterfly.Archive{Name: bundle.Name, Images: map[string][]byte{}, Warnings: []string{"Sharing permissions, comments, templates, and canvas-specific settings are not included."}}
	cached := map[string]string{}
	budget := mediaBudget{}
	for _, page := range bundle.Pages {
		for index := range page.Document.Elements {
			element := &page.Document.Elements[index]
			if ctx.Err() != nil {
				return nil, nil, invalid("Notebook conversion timed out or was cancelled. Export fewer pages.")
			}
			if element.Type != elementImage && element.Type != elementPDF {
				continue
			}
			key := element.AssetID
			if element.Crop != nil {
				key += fmt.Sprintf("-%g-%g-%g-%g", element.Crop.X, element.Crop.Y, element.Crop.Width, element.Crop.Height)
			}
			if path, exists := cached[key]; exists {
				element.Type = elementImage
				element.AssetID = path
				element.Crop = nil

				continue
			}
			data, notes, mediaErr := butterflyMedia(element, contents)
			if mediaErr != nil {
				return nil, nil, mediaErr
			}
			archive.Warnings = append(archive.Warnings, notes...)
			budgetErr := budget.add(data)
			if budgetErr != nil {
				return nil, nil, budgetErr
			}
			path := fmt.Sprintf("BFLY/images/image-%d.bin", len(archive.Images)+1)
			archive.Images[path] = data
			cached[key] = path
			element.AssetID = path
		}
		archive.Pages = append(archive.Pages, butterfly.Page{Name: page.Name, Document: page.Document})
	}

	return butterfly.Write(archive)
}

func cropButterflyImage(data []byte, crop *canvas.ImageCrop) ([]byte, error) {
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode export crop: %w", err)
	}
	size := source.Bounds()
	box := image.Rect(size.Min.X+int(math.Floor(crop.X*float64(size.Dx()))), size.Min.Y+int(math.Floor(crop.Y*float64(size.Dy()))), size.Min.X+int(math.Ceil((crop.X+crop.Width)*float64(size.Dx()))), size.Min.Y+int(math.Ceil((crop.Y+crop.Height)*float64(size.Dy())))).Intersect(size)
	cropped := image.NewNRGBA(image.Rect(0, 0, box.Dx(), box.Dy()))
	draw.Draw(cropped, cropped.Bounds(), source, box.Min, draw.Src)
	var encoded bytes.Buffer
	err = png.Encode(&encoded, cropped)
	if err != nil {
		return nil, fmt.Errorf("encode export crop: %w", err)
	}

	return encoded.Bytes(), nil
}

func butterflyMedia(element *canvas.Element, contents map[string][]byte) ([]byte, []string, error) {
	data := contents[assetPath(element.AssetID)]
	notes := []string{}
	if element.Type == elementPDF {
		data = contents[previewPath(element.AssetID)]
		element.Type = elementImage
		notes = append(notes, "Embedded PDFs are exported as their first-page preview images.")
	}
	if element.Crop != nil {
		var err error
		data, err = cropButterflyImage(data, element.Crop)
		if err != nil {
			return nil, nil, err
		}
		element.Crop = nil
		notes = append(notes, "Image crops are baked into exported media; cropped-away pixels are not included.")
	}

	return data, notes, nil
}
