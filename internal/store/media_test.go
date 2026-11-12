package store

import (
	"image"
	"image/color"
	"testing"
)

func TestVariantThresholdAndOrientation(t *testing.T) {
	if variantBucket(1000, 1000, 800, 800) != 0 {
		t.Fatal("generated a variant at the 20% threshold")
	}
	b := variantBucket(1000, 1000, 700, 700)
	if b < 700 || b >= 800 {
		t.Fatalf("missing useful downscale bucket: %d", b)
	}
	if variantBucket(1000, 500, 200, 500) != 0 {
		t.Fatal("downscaled despite needing full source resolution on one axis")
	}
	source := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	source.Set(0, 0, color.NRGBA{R: 255, A: 255})
	rotated := orient(source, 6)
	if rotated.Bounds().Dx() != 2 || rotated.Bounds().Dy() != 3 {
		t.Fatal("EXIF orientation dimensions not swapped")
	}
	r, _, _, _ := rotated.At(1, 0).RGBA()
	if r != 65535 {
		t.Fatal("EXIF orientation rotated pixels incorrectly")
	}
}
