package domain

import (
	"math"
	"regexp"
)

type Element struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	X           float64   `json:"x"`
	Y           float64   `json:"y"`
	Width       float64   `json:"width"`
	Height      float64   `json:"height"`
	Rotation    float64   `json:"rotation"`
	GroupID     string    `json:"groupId,omitempty"`
	Shape       string    `json:"shape,omitempty"`
	Fill        string    `json:"fill,omitempty"`
	Border      string    `json:"border,omitempty"`
	BorderWidth float64   `json:"borderWidth,omitempty"`
	Text        string    `json:"text,omitempty"`
	FontSize    float64   `json:"fontSize,omitempty"`
	Bold        bool      `json:"bold,omitempty"`
	Italic      bool      `json:"italic,omitempty"`
	Align       string    `json:"align,omitempty"`
	Color       string    `json:"color,omitempty"`
	AssetID     string    `json:"assetId,omitempty"`
	Points      []float64 `json:"points,omitempty"`
	PenWidth    float64   `json:"penWidth,omitempty"`
}
type CanvasSettings struct {
	Background string  `json:"background"`
	GridSize   float64 `json:"gridSize"`
	Snap       bool    `json:"snap"`
}
type Document struct {
	SchemaVersion int            `json:"schemaVersion"`
	Elements      []Element      `json:"elements"`
	Settings      CanvasSettings `json:"settings"`
}

func NewDocument() Document {
	return Document{SchemaVersion: 1, Elements: []Element{}, Settings: CanvasSettings{Background: "dots", GridSize: 20}}
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}([0-9a-fA-F]{2})?$`)

func ValidColor(c string) bool { return c == "transparent" || colorPattern.MatchString(c) }
func Finite(v float64) bool    { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func ValidateDocument(d Document) error {
	if d.Elements == nil {
		return Invalid("Document elements must be an array.")
	}
	if d.SchemaVersion != 1 {
		return Invalid("Unsupported document format.")
	}
	if len(d.Elements) > 10000 {
		return Invalid("A page can contain at most 10,000 elements.")
	}
	if d.Settings.Background != "dots" && d.Settings.Background != "plain" {
		return Invalid("Invalid canvas background.")
	}
	if !Finite(d.Settings.GridSize) || d.Settings.GridSize < 1 || d.Settings.GridSize > 1000 {
		return Invalid("Grid spacing must be between 1 and 1,000.")
	}
	ids := map[string]bool{}
	groups := map[string]bool{}
	for _, e := range d.Elements {
		if e.ID == "" || len(e.ID) > 128 || ids[e.ID] {
			return Invalid("Invalid or duplicate element ID.")
		}
		ids[e.ID] = true
		if len(e.GroupID) > 128 {
			return Invalid("Invalid group ID.")
		}
		if e.GroupID != "" {
			groups[e.GroupID] = true
		}
		for _, n := range []float64{e.X, e.Y, e.Width, e.Height, e.Rotation, e.BorderWidth, e.PenWidth, e.FontSize} {
			if !Finite(n) || math.Abs(n) > 1e9 {
				return Invalid("Invalid element geometry.")
			}
		}
		if e.Width < .1 || e.Height < .1 || e.Width > 1e6 || e.Height > 1e6 {
			return Invalid("Element dimensions are out of range.")
		}
		switch e.Type {
		case "shape":
			if e.Shape != "rectangle" && e.Shape != "circle" && e.Shape != "star" {
				return Invalid("Unsupported shape.")
			}
			if !ValidColor(e.Fill) || !ValidColor(e.Border) || e.BorderWidth < 0 || e.BorderWidth > 100 {
				return Invalid("Invalid shape style.")
			}
		case "text":
			if len(e.Text) > 100000 || e.FontSize < 1 || e.FontSize > 2000 || !ValidColor(e.Color) || (e.Align != "left" && e.Align != "center" && e.Align != "right") {
				return Invalid("Invalid text style or size.")
			}
		case "image":
			if e.AssetID == "" || len(e.AssetID) > 128 {
				return Invalid("Image asset is missing.")
			}
		case "stroke":
			if len(e.Points) < 4 || len(e.Points) > 40000 || len(e.Points)%2 != 0 || e.PenWidth <= 0 || e.PenWidth > 100 || !ValidColor(e.Color) {
				return Invalid("Invalid pen stroke.")
			}
			for _, n := range e.Points {
				if !Finite(n) || math.Abs(n) > 1e6 {
					return Invalid("Invalid drawing point.")
				}
			}
		default:
			return Invalid("Unsupported element type.")
		}
	}
	for id := range groups {
		if ids[id] {
			return Invalid("Group IDs must be separate from element IDs.")
		}
	}
	return nil
}
