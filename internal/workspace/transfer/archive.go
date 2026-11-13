package transfer

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mindflow/internal/workspace"
	"mindflow/internal/workspace/page/canvas"
	"regexp"
	"strings"
)

var assetKey = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,80}$`)

// Read validates a portable archive without extracting files or resolving URLs.
func Read(raw []byte) (Bundle, map[string][]byte, error) {
	if len(raw) > maxArchiveBytes {
		return Bundle{}, nil, invalid("Choose a mindflow export up to 64 MiB.")
	}
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil || len(reader.File) > maxFiles {
		return Bundle{}, nil, invalid("Choose a valid .mindflow export.")
	}
	files := make(map[string]*zip.File, len(reader.File))
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() || !entry.Mode().IsRegular() || entry.Flags&1 != 0 || files[entry.Name] != nil {
			return Bundle{}, nil, invalid("The archive contains duplicate, encrypted or unsupported entries.")
		}
		files[entry.Name] = entry
	}
	total := int64(0)
	read := func(name string, limit uint64) ([]byte, error) {
		entry := files[name]
		if entry == nil || entry.UncompressedSize64 > limit {
			return nil, invalid("The archive is missing content or exceeds the import limits.")
		}
		stream, openErr := entry.Open()
		if openErr != nil {
			return nil, invalid("An archive entry cannot be read.")
		}
		defer func() { _ = stream.Close() }()
		data, readErr := io.ReadAll(io.LimitReader(stream, min(int64(min(limit, maxExpandedBytes)), maxExpandedBytes-total)+1))
		total += int64(len(data))
		if readErr != nil || uint64(len(data)) > limit || total > maxExpandedBytes {
			return nil, invalid("The archive is corrupt or expands beyond 128 MiB.")
		}
		delete(files, name)

		return data, nil
	}
	data, err := read(manifestPath, maxManifestBytes)
	if err != nil {
		return Bundle{}, nil, err
	}
	var bundle Bundle
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&bundle) != nil {
		return Bundle{}, nil, invalid("The mindflow manifest is invalid or unsupported.")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return Bundle{}, nil, invalid("The manifest contains trailing data.")
	}
	err = validateBundle(bundle)
	if err != nil {
		return Bundle{}, nil, err
	}
	contents := make(map[string][]byte, len(bundle.Assets)*entriesPerAsset)
	for _, asset := range bundle.Assets {
		name := assetPath(asset.ID)
		contents[name], err = read(name, maxAssetBytes)
		if err != nil {
			return Bundle{}, nil, err
		}
		if asset.Preview {
			name = previewPath(asset.ID)
			contents[name], err = read(name, maxPreviewBytes)
			if err != nil {
				return Bundle{}, nil, err
			}
		}
	}
	if len(files) != 0 {
		return Bundle{}, nil, invalid("The archive contains unexpected files.")
	}

	return bundle, contents, nil
}

func validateBundle(bundle Bundle) error {
	if bundle.Format != formatName || bundle.Version != formatVersion || (bundle.Kind != "workspace" && bundle.Kind != kindPages) {
		return invalid("This mindflow export version is not supported.")
	}
	if workspace.ValidateName(bundle.Name) != nil || (bundle.Kind == kindPages && len(bundle.Pages) < 1) || len(bundle.Pages) > maxPages || len(bundle.Assets) > maxAssets || len(bundle.Templates) > maxTemplates {
		return invalid("An export supports 1–100 pages, 100 templates and 1,000 assets.")
	}
	assets, err := validateAssets(bundle.Assets)
	if err != nil {
		return err
	}
	used := make(map[string]bool)
	elements := 0
	for _, entry := range bundle.Pages {
		if workspace.ValidateName(entry.Name) != nil {
			return invalid("Invalid imported page name.")
		}
		err := checkDocument(entry.Document, maxPageBytes, assets, used, &elements)
		if err != nil {
			return err
		}
	}
	names := map[string]bool{}
	for _, entry := range bundle.Templates {
		if workspace.ValidateName(entry.Name) != nil || names[entry.Name] || len(entry.Elements) < 1 || len(entry.Elements) > maxTemplateElements {
			return invalid("Invalid or duplicate template name or content.")
		}
		names[entry.Name] = true
		document := canvas.NewDocument()
		document.Elements = entry.Elements
		err := checkDocument(document, maxTemplateBytes, assets, used, &elements)
		if err != nil {
			return err
		}
	}
	if len(used) != len(assets) {
		return invalid("The archive contains unreferenced media.")
	}

	return nil
}

func encode(bundle Bundle, contents map[string][]byte) ([]byte, error) {
	err := validateBundle(bundle)
	if err != nil {
		return nil, err
	}
	metadata, err := json.Marshal(bundle)
	if err != nil {
		return nil, fmt.Errorf("encode mindflow manifest: %w", err)
	}
	if len(metadata) > maxManifestBytes {
		return nil, invalid("This export's manifest exceeds 32 MiB. Export fewer pages.")
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	total := 0
	write := func(name string, data []byte) error {
		total += len(data)
		if total > maxExpandedBytes {
			return invalid("This export exceeds 128 MiB of content. Export fewer pages.")
		}
		entry, createErr := writer.Create(name)
		if createErr != nil {
			return fmt.Errorf("create archive entry: %w", createErr)
		}
		_, writeErr := entry.Write(data)
		if buffer.Len() > maxArchiveBytes {
			return invalid("This export exceeds 64 MiB. Export fewer pages.")
		}

		if writeErr != nil {
			return fmt.Errorf("write archive entry: %w", writeErr)
		}

		return nil
	}
	err = write(manifestPath, metadata)
	if err != nil {
		return nil, err
	}
	for _, asset := range bundle.Assets {
		err = write(assetPath(asset.ID), contents[assetPath(asset.ID)])
		if err != nil {
			return nil, err
		}
		if asset.Preview {
			err = write(previewPath(asset.ID), contents[previewPath(asset.ID)])
			if err != nil {
				return nil, err
			}
		}
	}
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("close mindflow export: %w", err)
	}
	if buffer.Len() > maxArchiveBytes {
		return nil, invalid("This export exceeds 64 MiB. Export fewer pages.")
	}

	return buffer.Bytes(), nil
}

func portableName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Imported workspace"
	}

	return value
}

func validateAssets(entries []Asset) (map[string]Asset, error) {
	assets := make(map[string]Asset, len(entries))
	for _, asset := range entries {
		if !assetKey.MatchString(asset.ID) || assets[asset.ID].ID != "" {
			return nil, invalid("Invalid or duplicate portable asset identity.")
		}
		switch asset.MIME {
		case "image/png", "image/jpeg", "image/webp":
			if asset.Preview {
				return nil, invalid("Only retained PDFs have separate previews.")
			}
		case mediaPDF:
			if !asset.Preview {
				return nil, invalid("A retained PDF requires its preview.")
			}
		default:
			return nil, invalid("The export contains an unsupported asset format.")
		}
		assets[asset.ID] = asset
	}

	return assets, nil
}

func checkDocument(document canvas.Document, limit int, assets map[string]Asset, used map[string]bool, elements *int) error {
	encoded, err := json.Marshal(document)
	if err != nil || len(encoded) > limit {
		return invalid("An exported page or template exceeds its size limit.")
	}
	err = canvas.ValidateDocument(document)
	if err != nil {
		return err
	}
	*elements += len(document.Elements)
	if *elements > maxElements {
		return invalid("An export can contain at most 100,000 elements.")
	}
	for _, element := range document.Elements {
		if element.Type != elementImage && element.Type != elementPDF {
			continue
		}
		asset, exists := assets[element.AssetID]
		if !exists || (element.Type == elementPDF) != (asset.MIME == mediaPDF) {
			return invalid("An element references missing or incompatible media.")
		}
		used[asset.ID] = true
	}

	return nil
}
