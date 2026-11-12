package image_test

import (
	"encoding/binary"
	"image"
	"image/color"
	imageasset "mindflow/internal/workspace/image"
	"testing"
)

func TestVariantThresholdAndOrientation(t *testing.T) {
	t.Parallel()

	if imageasset.VariantBucket(1000, 1000, 800, 800) != 0 {
		t.Fatal("generated a variant at the 20% threshold")
	}
	b := imageasset.VariantBucket(1000, 1000, 700, 700)
	if b < 700 || b >= 800 {
		t.Fatalf("missing useful downscale bucket: %d", b)
	}
	if imageasset.VariantBucket(1000, 500, 200, 500) != 0 {
		t.Fatal("downscaled despite needing full source resolution on one axis")
	}
	source := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	source.Set(0, 0, color.NRGBA{R: 255, A: 255})
	rotated := imageasset.OrientForTest(source, 6)
	if rotated.Bounds().Dx() != 2 || rotated.Bounds().Dy() != 3 {
		t.Fatal("EXIF orientation dimensions not swapped")
	}
	red, green, blue, alpha := rotated.At(1, 0).RGBA()
	if green != 0 || blue != 0 || alpha != 65535 {
		t.Fatal("EXIF orientation changed pixel color")
	}
	if red != 65535 {
		t.Fatal("EXIF orientation rotated pixels incorrectly")
	}
}

func TestJPEGOrientationParsesBothByteOrders(t *testing.T) {
	t.Parallel()
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		for orientation := uint16(1); orientation <= 8; orientation++ {
			tiff := make([]byte, 26)
			copy(tiff, "II")
			if order == binary.BigEndian {
				copy(tiff, "MM")
			}
			order.PutUint16(tiff[2:4], 42)
			order.PutUint32(tiff[4:8], 8)
			order.PutUint16(tiff[8:10], 1)
			order.PutUint16(tiff[10:12], 0x112)
			order.PutUint16(tiff[12:14], 3)
			order.PutUint32(tiff[14:18], 1)
			order.PutUint16(tiff[18:20], orientation)
			jpegData := append([]byte{0xff, 0xd8, 0xff, 0xe1, 0, 34}, []byte("Exif\x00\x00")...)
			jpegData = append(jpegData, tiff...)
			if actual := imageasset.JPEGOrientationForTest(jpegData); actual != int(orientation) {
				t.Fatalf("orientation: got %d, want %d", actual, orientation)
			}
			if actual := imageasset.JPEGOrientationForTest(jpegData[:12]); actual != 1 {
				t.Fatalf("truncated EXIF returned %d", actual)
			}
		}
	}
}
