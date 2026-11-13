package transfer //nolint:testpackage // Exercises internal archive validation and encoding without exporting implementation details.

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io/fs"
	"mindflow/internal/workspace/page/canvas"
	"testing"
)

func emptyBundle() Bundle {
	return Bundle{Format: formatName, Version: formatVersion, Kind: "workspace", Name: "Portable", Pages: []Page{{Name: "Empty", Document: canvas.NewDocument()}}}
}

func TestArchiveRejectsInvalidManifests(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*Bundle){
		"future version":     func(b *Bundle) { b.Version++ },
		"unsafe media key":   func(b *Bundle) { b.Assets = []Asset{{ID: "../escape", MIME: "image/png"}} },
		"unreferenced media": func(b *Bundle) { b.Assets = []Asset{{ID: "unused", MIME: "image/png"}} },
		"missing media": func(b *Bundle) {
			b.Pages[0].Document.Elements = []canvas.Element{{ID: "image", Type: elementImage, Width: 100, Height: 100, AssetID: "missing"}}
		},
		"invalid geometry": func(b *Bundle) {
			b.Pages[0].Document.Elements = []canvas.Element{{ID: "invalid", Type: "shape", Shape: "rectangle", Width: -1, Height: 100}}
		},
		"empty selection": func(b *Bundle) { b.Kind = kindPages; b.Pages = nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			bundle := emptyBundle()
			mutate(&bundle)
			if validateBundle(bundle) == nil {
				t.Fatal("invalid manifest accepted")
			}
		})
	}
}

func rawArchive(t *testing.T, names []string, mode fs.FileMode, manifest []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		if mode != 0 {
			header.SetMode(mode)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		_, err = entry.Write(manifest)
		if err != nil {
			t.Fatal(err)
		}
	}
	err := writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	return buffer.Bytes()
}

func TestArchiveRejectsUnexpectedDuplicateAndSymlinkEntries(t *testing.T) {
	t.Parallel()
	metadata, err := json.Marshal(emptyBundle())
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"traversal":     rawArchive(t, []string{manifestPath, "../escape"}, 0, metadata),
		"duplicate":     rawArchive(t, []string{manifestPath, manifestPath}, 0, metadata),
		"symlink":       rawArchive(t, []string{manifestPath}, fs.ModeSymlink|0o600, metadata),
		"trailing JSON": rawArchive(t, []string{manifestPath}, 0, append(metadata, []byte("{}")...)),
		"unknown field": rawArchive(t, []string{manifestPath}, 0, []byte(`{"unsupported":true}`)),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, _, readErr := Read(raw)
			if readErr == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}

func TestArchiveRoundTripAndEmptyWorkspace(t *testing.T) {
	t.Parallel()
	bundle := emptyBundle()
	raw, err := encode(bundle, nil)
	if err != nil {
		t.Fatal(err)
	}
	restored, _, err := Read(raw)
	if err != nil || len(restored.Pages) != 1 {
		t.Fatal("round trip failed", err)
	}
	bundle.Pages = nil
	raw, err = encode(bundle, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = Read(raw)
	if err != nil {
		t.Fatal("empty workspace rejected", err)
	}
}
