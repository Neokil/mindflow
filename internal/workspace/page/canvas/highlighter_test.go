package canvas_test

import (
	"mindflow/internal/workspace/page/canvas"
	"testing"
)

func TestHighlightStrokeValidation(t *testing.T) {
	t.Parallel()
	document := canvas.NewDocument()
	document.Elements = []canvas.Element{{ID: "marker", Type: "stroke", Width: 100, Height: 10, Points: []float64{0, 0, 1, 1}, PenWidth: 24, Color: testBlack, Highlight: true}}
	err := canvas.ValidateDocument(document)
	if err != nil {
		t.Fatal(err)
	}
	document.Elements[0].Type = "shape"
	document.Elements[0].Shape = "rectangle"
	document.Elements[0].Fill = "transparent"
	document.Elements[0].Border = testBlack
	if canvas.ValidateDocument(document) == nil {
		t.Fatal("accepted a highlight flag on a shape")
	}
}
