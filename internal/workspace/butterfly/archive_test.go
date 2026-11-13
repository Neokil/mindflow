package butterfly_test

import (
	"archive/zip"
	"bytes"
	"io"
	"mindflow/internal/workspace/butterfly"
	"os"
	"strings"
	"testing"
)

func sample(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/two-pages.bfly")
	if err != nil {
		t.Fatal(err)
	}

	return raw
}

func TestTwoPagesPreserveGeometryColorsAndEmbeddedImages(t *testing.T) {
	t.Parallel()
	archive, err := butterfly.Read(sample(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.Pages) != 2 || archive.Pages[0].Name != "Page 1" || archive.Pages[1].Name != "Page 2" {
		t.Fatal("page order or names changed")
	}
	first, second := archive.Pages[0].Document, archive.Pages[1].Document
	if len(first.Elements) != 3 || len(second.Elements) != 3 {
		t.Fatal("elements lost or empty labels imported")
	}
	if first.Settings.BackgroundColor != "#000000" || second.Settings.BackgroundColor != "#000000" {
		t.Fatal("white content lost its dark background")
	}
	stroke, image, title := first.Elements[0], first.Elements[1], first.Elements[2]
	if stroke.Type != "stroke" || stroke.Color != "#ff5722" || len(stroke.Points) != 6 || stroke.Points[0] != 0 || stroke.Points[5] != 1 {
		t.Fatal("pen geometry or color changed", stroke)
	}
	if image.Type != "image" || image.X != 1770 || image.Y != 453 || image.Width != 512 || image.Height != 512 || len(archive.Images[image.AssetID]) == 0 {
		t.Fatal("image geometry/bytes lost")
	}
	if title.Text != "Title of Image" || title.FontSize != 24 || title.Color != white {
		t.Fatal("white title lost")
	}
	text, circle, rectangle := second.Elements[0], second.Elements[1], second.Elements[2]
	if text.Text != "Text on second page" || text.Color != white || circle.Shape != "circle" || circle.Border != white || rectangle.Shape != "rectangle" || rectangle.Border != "#f44336" {
		t.Fatal("second page content changed")
	}
	if len(archive.Warnings) == 0 {
		t.Fatal("approximate pen/text conversion was not reported")
	}
}

func mutateArchive(t *testing.T, raw []byte, mutate func(string, []byte) (string, []byte)) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, entry := range reader.File {
		stream, openErr := entry.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		data, readErr := io.ReadAll(stream)
		_ = stream.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		name, data := mutate(entry.Name, data)
		target, createErr := writer.Create(name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		_, writeErr := target.Write(data)
		if writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	return output.Bytes()
}

func TestRejectsUnsafePathsExternalImagesAndUnknownVersions(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"path", "external", "version", "missing-image"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			changed := mutateArchive(t, sample(t), func(name string, data []byte) (string, []byte) {
				if kind == "path" && strings.HasPrefix(name, "BFLY/images/") {
					name = "../outside.png"
				}
				if kind == "missing-image" && strings.HasPrefix(name, "BFLY/images/") {
					name = "BFLY/images/other.png"
				}
				if kind == "external" {
					data = bytes.ReplaceAll(data, []byte("BFLY/images/example.png"), []byte("https://example.invalid/private.png"))
				}
				if kind == "version" && name == "META-INF/butterfly.json" {
					data = bytes.ReplaceAll(data, []byte(`"fileVersion": 13`), []byte(`"fileVersion": 99`))
				}

				return name, data
			})
			_, err := butterfly.Read(changed)
			if err == nil {
				t.Fatal("unsafe or unsupported archive accepted", kind)
			}
		})
	}
}

func TestRejectsExpandedArchiveBudgetBeforeDecompression(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	_, err := writer.CreateRaw(&zip.FileHeader{Name: "oversized.bin", Method: zip.Store, UncompressedSize64: 129 << 20})
	if err != nil {
		t.Fatal(err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}
	_, err = butterfly.Read(output.Bytes())
	if err == nil {
		t.Fatal("unbounded expanded entry accepted")
	}
}

const white = "#ffffff"
