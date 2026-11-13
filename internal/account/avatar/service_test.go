package avatar_test

import (
	"bytes"
	"image"
	"image/png"
	"mindflow/internal/account/avatar"
	"testing"
)

func TestNormalizeStripsMetadataAndBoundsDimensions(t *testing.T) {
	t.Parallel()
	var raw bytes.Buffer
	err := png.Encode(&raw, image.NewNRGBA(image.Rect(0, 0, 320, 160)))
	if err != nil {
		t.Fatal(err)
	}
	content, err := avatar.Normalize(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	config, kind, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || kind != "png" || config.Width != 128 || config.Height != 128 {
		t.Fatal("avatar was not normalized", config, kind, err)
	}
	_, err = avatar.Normalize([]byte(`<svg onload="alert(1)"></svg>`))
	if err == nil {
		t.Fatal("active image accepted")
	}
	var oversized bytes.Buffer
	err = png.Encode(&oversized, image.NewNRGBA(image.Rect(0, 0, 8193, 1)))
	if err != nil {
		t.Fatal(err)
	}
	_, err = avatar.Normalize(oversized.Bytes())
	if err == nil {
		t.Fatal("oversized dimensions accepted")
	}
}
