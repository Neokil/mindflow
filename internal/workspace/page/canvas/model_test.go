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
	document.Elements = []canvas.Element{{ID: testTextKind, Type: testTextKind, Width: 100, Height: 100, FontSize: 24, Color: testBlack, Align: "left"}}
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
	for _, color := range []string{"", testBlack, "#ffffff80", testTransparent} {
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

func TestGroupLockStateIsConsistent(t *testing.T) {
	t.Parallel()
	document := canvas.NewDocument()
	document.Elements = []canvas.Element{
		{ID: "one", Type: testShapeKind, Shape: "rectangle", Width: 100, Height: 100, Fill: testTransparent, Border: testBlack, GroupID: "group", Locked: true},
		{ID: "two", Type: testShapeKind, Shape: "circle", Width: 100, Height: 100, Fill: testTransparent, Border: testBlack, GroupID: "group", Locked: true},
	}
	err := canvas.ValidateDocument(document)
	if err != nil {
		t.Fatal(err)
	}
	document.Elements[1].Locked = false
	if canvas.ValidateDocument(document) == nil {
		t.Fatal("accepted a partially locked group")
	}
	document.Elements = document.Elements[:1]
	document.Elements[0].GroupID = ""
	if canvas.ValidateDocument(document) == nil {
		t.Fatal("accepted locked ungrouped element")
	}
}

const (
	testBlack       = "#000000"
	testTransparent = "transparent"
)

func TestElementDefaultColorsAndLegacySettings(t *testing.T) {
	t.Parallel()
	document := canvas.NewDocument()
	if document.Settings.DefaultFill != "#1b5e3b" || document.Settings.DefaultBorder != "#d9a441" || document.Settings.DefaultTextColor != "#d9a441" {
		t.Fatal("unexpected element defaults")
	}
	for _, field := range []string{"fill", "border", "text color"} {
		t.Run(field, func(t *testing.T) {
			t.Parallel()
			for _, color := range []string{"", testBlack, "#12345680", testTransparent, "invalid", "url(https://example.invalid/image.png)"} {
				current := canvas.NewDocument()
				switch field {
				case "fill":
					current.Settings.DefaultFill = color
				case "border":
					current.Settings.DefaultBorder = color
				case "text color":
					current.Settings.DefaultTextColor = color
				}
				err := canvas.ValidateDocument(current)
				valid := color == "" || canvas.ValidColor(color)
				if (err == nil) != valid {
					t.Errorf("default %s accepted/rejected %q incorrectly: %v", field, color, err)
				}
			}
		})
	}
}

func TestLists(t *testing.T) {
	t.Parallel()
	document := canvas.NewDocument()
	document.Elements = []canvas.Element{{ID: "tasks", Type: testTextKind, Width: 200, Height: 100, Text: "First\nSecond", FontSize: 24, Align: "left", Color: testBlack, ListStyle: "check", Checked: []int{1}}}
	err := canvas.ValidateDocument(document)
	if err != nil {
		t.Fatal(err)
	}
	document.Elements[0].Checked = []int{2}
	if canvas.ValidateDocument(document) == nil {
		t.Fatal("accepted nonexistent check item")
	}
	document.Elements[0].Checked = []int{1, 1}
	if canvas.ValidateDocument(document) == nil {
		t.Fatal("accepted duplicate checked item")
	}
}

const testTextKind = "text"

const testShapeKind = "shape"
