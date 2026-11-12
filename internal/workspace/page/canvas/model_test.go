package canvas_test

import (
	"mindflow/internal/workspace/page/canvas"
	"testing"
)

func TestVerticalAlignmentValidation(t *testing.T) {
	t.Parallel()
	document := canvas.NewDocument()
	document.Elements = []canvas.Element{{ID: "text", Type: "text", Width: 100, Height: 100, FontSize: 24, Color: "#000000", Align: "left"}}
	for _, alignment := range []string{"", "top", "middle", "bottom"} {
		document.Elements[0].VerticalAlign = alignment
		err := canvas.ValidateDocument(document)
		if err != nil {
			t.Fatal(err)
		}
	}
	document.Elements[0].VerticalAlign = "invalid"
	if canvas.ValidateDocument(document) == nil {
		t.Fatal("accepted invalid vertical alignment")
	}
}
