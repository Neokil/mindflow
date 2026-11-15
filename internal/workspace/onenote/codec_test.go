package onenote //nolint:testpackage // Exercises the private bounded process-output writer to prevent io.Copy fast-path bypasses.

import (
	"bytes"
	"context"
	"errors"
	"image/color"
	"image/png"
	"io"
	"math"
	"mindflow/internal/workspace/page/canvas"
	"testing"
)

func TestCodecRejectsUnknownCommandsBeforeLaunch(t *testing.T) {
	t.Parallel()
	_, _, err := NewCodec("/must-not-run").Convert(t.Context(), "../../execute", nil)
	if err == nil {
		t.Fatal("unrecognized mode reached process runner")
	}
}

func TestOutputLimitCannotBeBypassedByIOCopy(t *testing.T) {
	t.Parallel()
	output := &limitedBuffer{limit: 16}
	// Reader deliberately lacks WriterTo; an embedded bytes.Buffer would expose
	// ReaderFrom and bypass the boundary through io.Copy's fast path.
	source := io.LimitReader(bytes.NewReader(bytes.Repeat([]byte("x"), 32)), 32)
	_, err := io.Copy(output, source)
	if !errors.Is(err, errOutputLimit) || len(output.Bytes()) > output.limit {
		t.Fatal("output limit was bypassed", err)
	}
}

func TestShapeFallbackKeepsRotationBoundsAndFill(t *testing.T) {
	t.Parallel()
	element := canvas.Element{ID: "shape", Type: "shape", X: 40, Y: 50, Width: 40, Height: 20, Rotation: math.Pi / 2, Shape: "rectangle", Fill: "#1b5e3b", Border: "#d9a441", BorderWidth: 2}
	data, replacement, err := RasterFallback(t.Context(), element, nil)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.ID != element.ID || replacement.X != element.X || replacement.Y != element.Y || replacement.Rotation != 0 || math.Abs(replacement.Width-22) > .001 || math.Abs(replacement.Height-42) > .001 {
		t.Fatal("fallback changed object bounds", replacement)
	}
	picture, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	fill := color.NRGBAModel.Convert(picture.At(picture.Bounds().Dx()/2, picture.Bounds().Dy()/2))
	if fill != (color.NRGBA{R: 27, G: 94, B: 59, A: 255}) {
		t.Fatal("fallback fill changed", fill)
	}
}

func TestCancelledRasterPreparationStopsBeforeRendering(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err := RasterFallback(ctx, canvas.Element{Type: "text", Text: "clipped text", Width: 2, Height: 2, FontSize: 2000}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled preparation was not stopped", err)
	}
}
