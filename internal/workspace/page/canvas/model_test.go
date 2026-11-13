package canvas_test

import (
	"math"
	"mindflow/internal/workspace/page/canvas"
	"testing"
)

func TestImageCropAndVerticalAlignmentValidation(t *testing.T) {
	t.Parallel()
	valid := canvas.Element{ID: "image", Type: "image", Width: 100, Height: 100, AssetID: "asset", Crop: &canvas.ImageCrop{X: .2, Y: .1, Width: .5, Height: .6}}
	document := canvas.NewDocument()
	document.Elements = []canvas.Element{valid}
	err := canvas.ValidateDocument(document)
	if err != nil {
		t.Fatal(err)
	}
	for _, crop := range []canvas.ImageCrop{{X: -.1, Width: 1, Height: 1}, {X: .8, Width: .3, Height: 1}, {Width: 0, Height: 1}, {Width: math.NaN(), Height: 1}} {
		document.Elements[0].Crop = &crop
		if canvas.ValidateDocument(document) == nil {
			t.Errorf("accepted invalid crop: %+v", crop)
		}
	}
	document.Elements = []canvas.Element{{ID: "text", Type: "text", Width: 100, Height: 100, FontSize: 24, Color: "#000000", Align: "left"}}
	for _, alignment := range []string{"", "top", "middle", "bottom"} {
		document.Elements[0].VerticalAlign = alignment
		err = canvas.ValidateDocument(document)
		if err != nil {
			t.Fatal(err)
		}
	}
	document.Elements[0].VerticalAlign = "invalid"
	if canvas.ValidateDocument(document) == nil {
		t.Fatal("accepted invalid vertical alignment")
	}
}

func TestCanvasBackgroundColorValidation(t *testing.T) {
	t.Parallel()
	for _, color := range []string{"", "#000000", "#ffffff80", "transparent"} {
		document := canvas.NewDocument()
		document.Settings.BackgroundColor = color
		err := canvas.ValidateDocument(document)
		if err != nil {
			t.Errorf("rejected supported background %q: %v", color, err)
		}
	}
	document := canvas.NewDocument()
	document.Settings.BackgroundColor = "url(https://example.invalid/image.png)"
	if canvas.ValidateDocument(document) == nil {
		t.Fatal("accepted a URL as a background color")
	}
}
