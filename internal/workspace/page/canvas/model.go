package canvas

import (
	"encoding/json"
	"math"
	"regexp"
	"strings"
)

// ImageCrop identifies a rectangle in normalized original-image coordinates.
type ImageCrop struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Element is an independently transformed canvas object.
type Element struct {
	Link          *ElementLink `json:"link,omitempty"`
	ID            string       `json:"id"`
	Type          string       `json:"type"`
	X             float64      `json:"x"`
	Y             float64      `json:"y"`
	Width         float64      `json:"width"`
	Height        float64      `json:"height"`
	Rotation      float64      `json:"rotation"`
	GroupID       string       `json:"groupId,omitempty"`
	Locked        bool         `json:"locked,omitempty"`
	Shape         string       `json:"shape,omitempty"`
	Fill          string       `json:"fill,omitempty"`
	Border        string       `json:"border,omitempty"`
	BorderWidth   float64      `json:"borderWidth,omitempty"`
	Text          string       `json:"text,omitempty"`
	ListStyle     string       `json:"listStyle,omitempty"`
	Checked       []int        `json:"checked,omitempty"`
	FontSize      float64      `json:"fontSize,omitempty"`
	Bold          bool         `json:"bold,omitempty"`
	Italic        bool         `json:"italic,omitempty"`
	Align         string       `json:"align,omitempty"`
	VerticalAlign string       `json:"verticalAlign,omitempty"`
	Crop          *ImageCrop   `json:"crop,omitempty"`
	Color         string       `json:"color,omitempty"`
	AssetID       string       `json:"assetId,omitempty"`
	Points        []float64    `json:"points,omitempty"`
	Pressures     []float64    `json:"pressures,omitempty"`
	PenWidth      float64      `json:"penWidth,omitempty"`
	Highlight     bool         `json:"highlight,omitempty"`
}

// Settings stores a page's background and grid preferences.
type Settings struct {
	BackgroundColor  string  `json:"backgroundColor,omitempty"`
	DefaultFill      string  `json:"defaultFill,omitempty"`
	DefaultBorder    string  `json:"defaultBorder,omitempty"`
	DefaultTextColor string  `json:"defaultTextColor,omitempty"`
	Background       string  `json:"background"`
	GridSize         float64 `json:"gridSize"`
	Snap             bool    `json:"snap"`
}

// Document is versioned renderer-independent page content.
type Document struct {
	SchemaVersion int       `json:"schemaVersion"`
	Elements      []Element `json:"elements"`
	Settings      Settings  `json:"settings"`
}

// NewDocument returns a blank canvas with default settings.
func NewDocument() Document {
	return Document{SchemaVersion: 1, Elements: []Element{}, Settings: Settings{Background: "dots", GridSize: defaultGridSpacing, DefaultFill: defaultFillColor, DefaultBorder: defaultGoldColor, DefaultTextColor: defaultGoldColor}}
}

// ValidColor accepts supported CSS hex colors and transparency.
func ValidColor(c string) bool { return c == "transparent" || colorPattern.MatchString(c) }

// Finite rejects NaN and infinite geometry values.
func Finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// ValidateDocument checks document structure and element constraints.
func ValidateDocument(document Document) error {
	if document.Elements == nil {
		return invalid("Document elements must be an array.")
	}
	if document.SchemaVersion != 1 {
		return invalid("Unsupported document format.")
	}
	if len(document.Elements) > maxElements {
		return invalid("A page can contain at most 10,000 elements.")
	}
	if document.Settings.BackgroundColor != "" && !ValidColor(document.Settings.BackgroundColor) {
		return invalid("Invalid canvas background color.")
	}
	if document.Settings.Background != "dots" && document.Settings.Background != "plain" {
		return invalid("Invalid canvas background.")
	}
	if !Finite(document.Settings.GridSize) || document.Settings.GridSize < 1 || document.Settings.GridSize > 1000 {
		return invalid("Grid spacing must be between 1 and 1,000.")
	}
	defaultsErr := validateDefaultColors(document.Settings)
	if defaultsErr != nil {
		return defaultsErr
	}
	ids := map[string]bool{}
	groups := map[string]bool{}
	locks := map[string]bool{}
	for _, entry := range document.Elements {
		if entry.ID == "" || len(entry.ID) > maxIdentifierLength || ids[entry.ID] {
			return invalid("Invalid or duplicate element ID.")
		}
		ids[entry.ID] = true
		if len(entry.GroupID) > maxIdentifierLength {
			return invalid("Invalid group ID.")
		}
		if entry.GroupID != "" {
			groups[entry.GroupID] = true
			if locked, exists := locks[entry.GroupID]; exists && locked != entry.Locked {
				return invalid("All group members must have the same lock state.")
			}
			locks[entry.GroupID] = entry.Locked
		}
		if entry.Locked && entry.GroupID == "" {
			return invalid("Only groups can be locked.")
		}
		validationErr := validateElement(entry)
		if validationErr != nil {
			return validationErr
		}
	}
	for id := range groups {
		if ids[id] {
			return invalid("Group IDs must be separate from element IDs.")
		}
	}

	return nil
}

func validateElement(entry Element) error {
	attributesErr := validateElementAttributes(entry)
	if attributesErr != nil {
		return attributesErr
	}
	switch entry.Type {
	case "shape":
		if entry.Shape != "rectangle" && entry.Shape != "circle" && entry.Shape != "star" {
			return invalid("Unsupported shape.")
		}
		if !ValidColor(entry.Fill) || !ValidColor(entry.Border) || entry.BorderWidth < 0 || entry.BorderWidth > 100 {
			return invalid("Invalid shape style.")
		}
	case elementText:
		if len(entry.Text) > 100000 || entry.FontSize < 1 || entry.FontSize > 2000 || !ValidColor(entry.Color) || (entry.Align != "left" && entry.Align != "center" && entry.Align != "right") {
			return invalid("Invalid text style or size.")
		}
	case elementImage, elementPDF:
		if entry.AssetID == "" || len(entry.AssetID) > maxIdentifierLength {
			return invalid("Image asset is missing.")
		}
	case elementStroke:
		return validateStroke(entry)
	default:
		return invalid("Unsupported element type.")
	}

	return nil
}

func validateList(entry Element) error {
	if entry.ListStyle != "" {
		if entry.Type != elementText || (entry.ListStyle != "bullet" && entry.ListStyle != "number" && entry.ListStyle != "check") || strings.Count(entry.Text, "\n") >= 1000 {
			return invalid("Invalid list style or too many list items (maximum 1,000).")
		}
	}
	seenChecked := map[int]bool{}
	for _, index := range entry.Checked {
		if entry.Type != elementText || entry.ListStyle != "check" || index < 0 || index > strings.Count(entry.Text, "\n") || seenChecked[index] {
			return invalid("Invalid checked list item.")
		}
		seenChecked[index] = true
	}

	return nil
}

func validateGeometry(entry Element) error {
	for _, n := range []float64{entry.X, entry.Y, entry.Width, entry.Height, entry.Rotation, entry.BorderWidth, entry.PenWidth, entry.FontSize} {
		if !Finite(n) || math.Abs(n) > 1e9 {
			return invalid("Invalid element geometry.")
		}
	}
	if entry.Width < .1 || entry.Height < .1 || entry.Width > 1e6 || entry.Height > 1e6 {
		return invalid("Element dimensions are out of range.")
	}

	return nil
}

func validateStroke(entry Element) error {
	if len(entry.Points) < 4 || len(entry.Points) > 40000 || len(entry.Points)%2 != 0 || entry.PenWidth <= 0 || entry.PenWidth > 100 || !ValidColor(entry.Color) {
		return invalid("Invalid pen stroke.")
	}
	for _, n := range entry.Points {
		if !Finite(n) || math.Abs(n) > 1e6 {
			return invalid("Invalid drawing point.")
		}
	}

	return nil
}

// PageSnapshot pairs persisted content with its revisions and metadata.
type PageSnapshot struct {
	Document         json.RawMessage `json:"document"`
	Revision         int             `json:"revision"`
	MetadataRevision int             `json:"metadataRevision"`
	Name             string          `json:"name"`
	WorkspaceID      string          `json:"workspaceId"`
}

// SaveReceipt acknowledges an idempotent save and its resulting revision.
type SaveReceipt struct {
	Revision   int    `json:"revision"`
	MutationID string `json:"mutationId"`
}

// SaveDocumentCommand identifies a fenced document mutation and its content.
type SaveDocumentCommand struct {
	BaseRevision int      `json:"baseRevision"`
	Epoch        int64    `json:"epoch"`
	ClientID     string   `json:"clientId"`
	MutationID   string   `json:"mutationId"`
	Document     Document `json:"document"`
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}([0-9a-fA-F]{2})?$`)

func validateImageCrop(entry Element) error {
	if entry.Crop != nil {
		c := entry.Crop
		if entry.Type != elementImage || !Finite(c.X) || !Finite(c.Y) || !Finite(c.Width) || !Finite(c.Height) || c.X < 0 || c.Y < 0 || c.Width <= 0 || c.Height <= 0 || c.X+c.Width > 1+1e-9 || c.Y+c.Height > 1+1e-9 {
			return invalid("Invalid image crop.")
		}
	}

	return nil
}

func validateDefaultColors(settings Settings) error {
	for _, color := range []string{settings.DefaultFill, settings.DefaultBorder, settings.DefaultTextColor} {
		if color != "" && !ValidColor(color) {
			return invalid("Invalid canvas element default color.")
		}
	}

	return nil
}

func validateElementAttributes(entry Element) error {
	if entry.Link != nil && (entry.Link.PageID == "" || len(entry.Link.PageID) > maxIdentifierLength || entry.Link.ElementID == "" || len(entry.Link.ElementID) > maxIdentifierLength) {
		return invalid("Invalid element link.")
	}

	listErr := validateList(entry)
	if listErr != nil {
		return listErr
	}

	geometryErr := validateGeometry(entry)
	if geometryErr != nil {
		return geometryErr
	}
	if entry.VerticalAlign != "" && entry.VerticalAlign != "top" && entry.VerticalAlign != "middle" && entry.VerticalAlign != "bottom" {
		return invalid("Invalid vertical alignment.")
	}
	if entry.Highlight && entry.Type != elementStroke {
		return invalid("Only strokes can be highlights.")
	}
	cropErr := validateImageCrop(entry)
	if cropErr != nil {
		return cropErr
	}

	return validatePressure(entry)
}

func validatePressure(entry Element) error {
	if len(entry.Pressures) == 0 {
		return nil
	}
	if entry.Type != elementStroke || entry.Highlight {
		return invalid("Only pen strokes can have pressure samples.")
	}
	if len(entry.Pressures) > 0 && len(entry.Pressures) != len(entry.Points)/2 {
		return invalid("Pressure samples must match the drawing points.")
	}
	for _, pressure := range entry.Pressures {
		if !Finite(pressure) || pressure < 0 || pressure > 1 {
			return invalid("Invalid pen pressure.")
		}
	}

	return nil
}
