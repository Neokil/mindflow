// Package butterfly imports supported Butterfly documents into private workspaces.
package butterfly

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mindflow/internal/workspace"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Archive is a bounded Butterfly document with ordered pages and embedded images.
type Archive struct {
	Name     string
	Pages    []Page
	Images   map[string][]byte
	Warnings []string
}

type archiveReader struct {
	files map[string]*zip.File
	total int64
}

// Read converts version-13 ZIP exports without extracting files or fetching URLs.
func Read(raw []byte) (Archive, error) {
	reader, err := openArchive(raw)
	if err != nil {
		return Archive{}, err
	}
	metadata, err := reader.read("META-INF/butterfly.json", maxMetadataBytes)
	if err != nil {
		return Archive{}, err
	}
	var meta struct {
		FileVersion int    `json:"fileVersion"`
		Type        string `json:"type"`
		Name        string `json:"name"`
	}
	if json.Unmarshal(metadata, &meta) != nil || meta.Type != "document" || meta.FileVersion != 13 {
		return Archive{}, invalid("Only unencrypted Butterfly document exports in file format version 13 are supported.")
	}
	type entry struct {
		name, title string
		order       int
	}
	entries := []entry{}
	orders := map[int]bool{}
	for name := range reader.files {
		if !strings.HasPrefix(name, "BFLY/pages/") || !strings.HasSuffix(name, ".json") {
			continue
		}
		basename := strings.TrimPrefix(name, "BFLY/pages/")
		if strings.Contains(basename, "/") {
			return Archive{}, invalid("Nested Butterfly page folders are not supported yet.")
		}
		number, title, found := strings.Cut(strings.TrimSuffix(basename, ".json"), ".")
		order, parseErr := strconv.Atoi(number)
		if !found || parseErr != nil || order < 0 || orders[order] || workspace.ValidateName(title) != nil {
			return Archive{}, invalid("The export contains an invalid page name or ordering.")
		}
		orders[order] = true
		entries = append(entries, entry{name, title, order})
	}
	if len(entries) == 0 || len(entries) > maxPages {
		return Archive{}, invalid("Choose a Butterfly export containing between 1 and 100 pages.")
	}
	sort.Slice(entries, func(left, right int) bool { return entries[left].order < entries[right].order })
	result := Archive{Name: strings.TrimSpace(meta.Name), Pages: make([]Page, 0, len(entries)), Images: make(map[string][]byte), Warnings: []string{}}
	converter := converter{archive: &result}
	for _, entry := range entries {
		encoded, readErr := reader.read(entry.name, maxPageBytes)
		if readErr != nil {
			return Archive{}, readErr
		}
		page, convertErr := converter.page(encoded)
		if convertErr != nil {
			return Archive{}, convertErr
		}
		result.Pages = append(result.Pages, Page{Name: entry.title, Document: page})
	}
	for source := range result.Images {
		if !strings.HasPrefix(source, "BFLY/images/") {
			return Archive{}, invalid("Imported images must be embedded in the Butterfly archive; external URLs are not loaded.")
		}
		image, readErr := reader.read(source, maxImageBytes)
		if readErr != nil {
			return Archive{}, readErr
		}
		result.Images[source] = image
	}

	return result, nil
}

func (reader *archiveReader) read(name string, limit int64) ([]byte, error) {
	file, exists := reader.files[name]
	if !exists || file.FileInfo().IsDir() {
		return nil, invalid("The export is missing required page, metadata or image data.")
	}
	if file.UncompressedSize64 > uint64(max(0, limit)) {
		return nil, invalid("An embedded page or image exceeds the import limits.")
	}
	stream, err := file.Open()
	if err != nil {
		return nil, invalid("An archive entry could not be opened.")
	}
	defer func() { _ = stream.Close() }()
	data, err := io.ReadAll(io.LimitReader(stream, limit+1))
	if err != nil || int64(len(data)) > limit || reader.total+int64(len(data)) > maxExpandedBytes {
		return nil, invalid("An archive entry is corrupt or exceeds the import limits.")
	}
	reader.total += int64(len(data))

	return data, nil
}

func invalid(message string) error {
	return &workspace.Error{Code: "validation_failed", Message: message}
}

func wrap(operation string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", operation, err)
}

const (
	maxArchiveBytes  = 64 << 20
	maxExpandedBytes = 128 << 20
	maxPageBytes     = 16 << 20
	maxImageBytes    = 20 << 20
	maxMetadataBytes = 64 << 10
	maxEntries       = 2048
	maxPages         = 100
	maxPathLength    = 1024
)

func openArchive(raw []byte) (archiveReader, error) {
	if len(raw) > maxArchiveBytes {
		return archiveReader{}, invalid("Choose a Butterfly export up to 64 MiB.")
	}
	zipReader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return archiveReader{}, invalid("This file is not a readable Butterfly ZIP export.")
	}
	if len(zipReader.File) > maxEntries {
		return archiveReader{}, invalid("The export contains too many files.")
	}
	reader := archiveReader{files: make(map[string]*zip.File)}
	var expanded uint64
	for _, file := range zipReader.File {
		name := strings.TrimSuffix(file.Name, "/")
		if name == "" || strings.ContainsAny(name, "\\\x00") || strings.HasPrefix(name, "/") || path.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") || len(name) > maxPathLength || file.Mode()&os.ModeSymlink != 0 || file.Flags&1 != 0 {
			return archiveReader{}, invalid("The export contains an unsafe path, symbolic link or encrypted entry.")
		}
		if _, exists := reader.files[file.Name]; exists {
			return archiveReader{}, invalid("The export contains duplicate archive paths.")
		}
		if file.UncompressedSize64 > maxExpandedBytes-expanded {
			return archiveReader{}, invalid("The expanded export exceeds 128 MiB.")
		}
		expanded += file.UncompressedSize64
		reader.files[file.Name] = file
	}

	return reader, nil
}
