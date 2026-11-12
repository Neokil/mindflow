package workspace

import (
	"math"
	"mindflow/internal/fault"
	"regexp"
)

// Element is an independently transformed object on a canvas.
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

// CanvasSettings stores a page background and grid preferences.
type CanvasSettings struct {
	Background string  `json:"background"`
	GridSize   float64 `json:"gridSize"`
	Snap       bool    `json:"snap"`
}

// Document is the versioned, renderer-independent page content.
type Document struct {
	SchemaVersion int            `json:"schemaVersion"`
	Elements      []Element      `json:"elements"`
	Settings      CanvasSettings `json:"settings"`
}

// NewDocument returns a blank canvas with default settings.
func NewDocument() Document {
	return Document{SchemaVersion: 1, Elements: []Element{}, Settings: CanvasSettings{Background: "dots", GridSize: defaultGridSpacing}}
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}([0-9a-fA-F]{2})?$`)

// ValidColor accepts supported CSS hex colors and transparency.
func ValidColor(c string) bool { return c == "transparent" || colorPattern.MatchString(c) }

// Finite rejects NaN and infinite geometry values.
func Finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// ValidateDocument checks document structure and element constraints.
func ValidateDocument(document Document) error {
	if document.Elements == nil {
		return fault.Invalid("Document elements must be an array.")
	}
	if document.SchemaVersion != 1 {
		return fault.Invalid("Unsupported document format.")
	}
	if len(document.Elements) > maxElements {
		return fault.Invalid("A page can contain at most 10,000 elements.")
	}
	if document.Settings.Background != "dots" && document.Settings.Background != "plain" {
		return fault.Invalid("Invalid canvas background.")
	}
	if !Finite(document.Settings.GridSize) || document.Settings.GridSize < 1 || document.Settings.GridSize > 1000 {
		return fault.Invalid("Grid spacing must be between 1 and 1,000.")
	}
	ids := map[string]bool{}
	groups := map[string]bool{}
	for _, entry := range document.Elements {
		if entry.ID == "" || len(entry.ID) > maxIdentifierLength || ids[entry.ID] {
			return fault.Invalid("Invalid or duplicate element ID.")
		}
		ids[entry.ID] = true
		if len(entry.GroupID) > maxIdentifierLength {
			return fault.Invalid("Invalid group ID.")
		}
		if entry.GroupID != "" {
			groups[entry.GroupID] = true
		}
		validationErr := validateElement(entry)
		if validationErr != nil {
			return validationErr
		}
	}
	for id := range groups {
		if ids[id] {
			return fault.Invalid("Group IDs must be separate from element IDs.")
		}
	}

	return nil
}

func validateElement(entry Element) error {
	geometryErr := validateGeometry(entry)
	if geometryErr != nil {
		return geometryErr
	}
	switch entry.Type {
	case "shape":
		if entry.Shape != "rectangle" && entry.Shape != "circle" && entry.Shape != "star" {
			return fault.Invalid("Unsupported shape.")
		}
		if !ValidColor(entry.Fill) || !ValidColor(entry.Border) || entry.BorderWidth < 0 || entry.BorderWidth > 100 {
			return fault.Invalid("Invalid shape style.")
		}
	case "text":
		if len(entry.Text) > 100000 || entry.FontSize < 1 || entry.FontSize > 2000 || !ValidColor(entry.Color) || (entry.Align != "left" && entry.Align != "center" && entry.Align != "right") {
			return fault.Invalid("Invalid text style or size.")
		}
	case elementImage:
		if entry.AssetID == "" || len(entry.AssetID) > maxIdentifierLength {
			return fault.Invalid("Image asset is missing.")
		}
	case "stroke":
		return validateStroke(entry)
	default:
		return fault.Invalid("Unsupported element type.")
	}

	return nil
}

func validateGeometry(entry Element) error {
	for _, n := range []float64{entry.X, entry.Y, entry.Width, entry.Height, entry.Rotation, entry.BorderWidth, entry.PenWidth, entry.FontSize} {
		if !Finite(n) || math.Abs(n) > 1e9 {
			return fault.Invalid("Invalid element geometry.")
		}
	}
	if entry.Width < .1 || entry.Height < .1 || entry.Width > 1e6 || entry.Height > 1e6 {
		return fault.Invalid("Element dimensions are out of range.")
	}

	return nil
}

func validateStroke(entry Element) error {
	if len(entry.Points) < 4 || len(entry.Points) > 40000 || len(entry.Points)%2 != 0 || entry.PenWidth <= 0 || entry.PenWidth > 100 || !ValidColor(entry.Color) {
		return fault.Invalid("Invalid pen stroke.")
	}
	for _, n := range entry.Points {
		if !Finite(n) || math.Abs(n) > 1e6 {
			return fault.Invalid("Invalid drawing point.")
		}
	}

	return nil
}
