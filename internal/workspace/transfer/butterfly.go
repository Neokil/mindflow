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
)

// ExportButterfly reuses the authorized atomic snapshot and exports only its media.
func (a *Service) ExportButterfly(ctx context.Context, actor session.Principal, workspaceID string, selection Selection) ([]byte, []string, error) {
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
	for _, page := range bundle.Pages {
		for index := range page.Document.Elements {
			element := &page.Document.Elements[index]
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
			data := contents[assetPath(element.AssetID)]
			if element.Type == elementPDF {
				data = contents[previewPath(element.AssetID)]
				element.Type = elementImage
				archive.Warnings = append(archive.Warnings, "Embedded PDFs are exported as their first-page preview images.")
			}
			if element.Crop != nil {
				source, _, decodeErr := image.Decode(bytes.NewReader(data))
				if decodeErr != nil {
					return nil, nil, fmt.Errorf("decode export crop: %w", decodeErr)
				}
				size := source.Bounds()
				crop := element.Crop
				box := image.Rect(size.Min.X+int(math.Floor(crop.X*float64(size.Dx()))), size.Min.Y+int(math.Floor(crop.Y*float64(size.Dy()))), size.Min.X+int(math.Ceil((crop.X+crop.Width)*float64(size.Dx()))), size.Min.Y+int(math.Ceil((crop.Y+crop.Height)*float64(size.Dy())))).Intersect(size)
				cropped := image.NewNRGBA(image.Rect(0, 0, box.Dx(), box.Dy()))
				draw.Draw(cropped, cropped.Bounds(), source, box.Min, draw.Src)
				var encoded bytes.Buffer
				err = png.Encode(&encoded, cropped)
				if err != nil {
					return nil, nil, fmt.Errorf("encode export crop: %w", err)
				}
				data = encoded.Bytes()
				element.Crop = nil
				archive.Warnings = append(archive.Warnings, "Image crops are baked into exported media; cropped-away pixels are not included.")
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
