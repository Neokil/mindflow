package butterfly

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"mindflow/internal/workspace/page/canvas"
	"slices"
	"strings"
)

func (c *converter) page(raw []byte) (canvas.Document, error) {
	var source sourcePage
	if json.Unmarshal(raw, &source) != nil {
		return canvas.Document{}, invalid("A Butterfly page contains invalid JSON.")
	}
	document := canvas.NewDocument()
	settings, err := c.background(source)
	if err != nil {
		return document, err
	}
	document.Settings = settings
	if len(source.Animations)+len(source.Areas)+len(source.Waypoints) > 0 {
		c.warn("Animations, named areas and waypoints are not imported.")
	}
	content := source.Content
	for _, layer := range source.Layers {
		content = append(content, layer.Content...)
	}
	c.elements += len(content)
	if len(content) > maxPageElements || c.elements > maxImportElements {
		return document, invalid("The export exceeds the supported element count.")
	}
	groups := map[string]string{}
	for _, element := range content {
		converted, keep, err := c.element(element)
		if err != nil {
			return document, err
		}
		if !keep {
			continue
		}
		if element.Collection != "" {
			if groups[element.Collection] == "" {
				groups[element.Collection] = newID()
			}
			converted.GroupID = groups[element.Collection]
		}
		document.Elements = append(document.Elements, converted)
	}
	err = canvas.ValidateDocument(document)
	if err != nil {
		return document, invalid("Imported page geometry or styling exceeds mindflow's supported limits.")
	}

	return document, nil
}

func (c *converter) element(source sourceElement) (canvas.Element, bool, error) {
	var element canvas.Element
	var err error
	switch source.Type {
	case "pen":
		element, err = c.stroke(source)
	case "shape":
		element, err = c.shape(source)
	case elementImage:
		element, err = c.image(source)
	case "text":
		element, err = c.text(source)
	default:
		kind := source.Type
		if len(kind) > maxTypeLength {
			kind = kind[:maxTypeLength]
		}
		c.warn(fmt.Sprintf("Skipped unsupported Butterfly element type %q.", kind))

		return element, false, nil
	}
	if err != nil {
		return element, false, err
	}
	if element.Type == "" {
		return element, false, nil
	}
	if source.Shear != 0 {
		c.warn("Sheared elements are imported without shear.")
	}
	element.ID = newID()
	element.Rotation = source.Rotation * math.Pi / degreesPerHalfTurn

	return element, true, nil
}

func (c *converter) shape(source sourceElement) (canvas.Element, error) {
	kind := source.Property.Shape.Type
	if kind != "rectangle" && kind != "circle" {
		c.warn("Unsupported shape types were skipped.")

		return canvas.Element{}, nil
	}
	if !solid(source.Property.Paint) || !solid(source.Property.Shape.FillPaint) {
		c.warn("Shapes with gradients or image paints were skipped.")

		return canvas.Element{}, nil
	}
	border, err := argb(source.Property.Paint.Color, "#000000")
	if err != nil {
		return canvas.Element{}, err
	}
	fill, err := argb(source.Property.Shape.FillPaint.Color, "transparent")
	if err != nil {
		return canvas.Element{}, err
	}
	left, top := math.Min(source.FirstPosition.X, source.SecondPosition.X), math.Min(source.FirstPosition.Y, source.SecondPosition.Y)
	width, height := math.Abs(source.SecondPosition.X-source.FirstPosition.X), math.Abs(source.SecondPosition.Y-source.FirstPosition.Y)

	return canvas.Element{Type: "shape", Shape: kind, X: left + width/2, Y: top + height/2, Width: math.Max(minDimension, width), Height: math.Max(minDimension, height), Fill: fill, Border: border, BorderWidth: number(source.Property.StrokeWidth, defaultPenWidth)}, nil
}

func (c *converter) stroke(source sourceElement) (canvas.Element, error) {
	if len(source.Points) == 0 {
		return canvas.Element{}, nil
	}
	if len(source.Points) > maxStrokePoints {
		return canvas.Element{}, invalid("A Butterfly pen stroke exceeds 20,000 points.")
	}
	if !solid(source.Property.Paint) {
		c.warn("Pen strokes with unsupported paints were skipped.")

		return canvas.Element{}, nil
	}
	color, err := argb(source.Property.Paint.Color, "#000000")
	if err != nil {
		return canvas.Element{}, err
	}
	left, right, top, bottom := source.Points[0].X, source.Points[0].X, source.Points[0].Y, source.Points[0].Y
	for _, point := range source.Points {
		left = math.Min(left, point.X)
		right = math.Max(right, point.X)
		top = math.Min(top, point.Y)
		bottom = math.Max(bottom, point.Y)
	}
	width, height := math.Max(minDimension, right-left), math.Max(minDimension, bottom-top)
	points := make([]float64, 0, max(minStrokeCoordinates, len(source.Points)*pointDimensions))
	for _, point := range source.Points {
		points = append(points, (point.X-left)/width, (point.Y-top)/height)
	}
	if len(points) == pointDimensions {
		points = append(points, points...)
	}
	c.warn("Pen pressure, smoothing and fill are approximated as editable strokes.")

	return canvas.Element{Type: "stroke", X: left + width/2, Y: top + height/2, Width: width, Height: height, Points: points, Color: color, PenWidth: number(source.Property.StrokeWidth, defaultPenWidth)}, nil
}

func (c *converter) image(source sourceElement) (canvas.Element, error) {
	if !strings.HasPrefix(source.Source, "BFLY/images/") || len(source.Source) > maxAssetPath {
		return canvas.Element{}, invalid("Images must use an embedded BFLY/images asset path.")
	}
	width, height := source.Width, source.Height
	switch source.Constraints.Type {
	case "", "scaled":
		width *= number(source.Constraints.ScaleX, 1)
		height *= number(source.Constraints.ScaleY, 1)
	case "fixed":
		width = source.Constraints.Width
		height = source.Constraints.Height
	default:
		return canvas.Element{}, invalid("This image uses unsupported Butterfly constraints.")
	}
	c.archive.Images[source.Source] = nil

	return canvas.Element{Type: elementImage, X: source.Position.X + width/2, Y: source.Position.Y + height/2, Width: width, Height: height, AssetID: source.Source}, nil
}

func (c *converter) warn(message string) {
	if slices.Contains(c.archive.Warnings, message) {
		return
	}
	if len(c.archive.Warnings) < maxWarnings {
		c.archive.Warnings = append(c.archive.Warnings, message)
	}
}
func solid(paint sourcePaint) bool { return paint.Type == "" || paint.Type == "solid" }
func number(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}

	return *value
}

func argb(value *int64, fallback string) (string, error) {
	if value == nil {
		return fallback, nil
	}
	if *value < math.MinInt32 || *value > math.MaxUint32 {
		return "", invalid("An imported color is invalid.")
	}
	color := uint32(*value)
	alpha := color >> alphaShift
	if alpha == 0 {
		return "transparent", nil
	}
	if alpha == opaqueAlpha {
		return fmt.Sprintf("#%06x", color&rgbMask), nil
	}

	return fmt.Sprintf("#%06x%02x", color&rgbMask, alpha), nil
}

func newID() string {
	value := make([]byte, identifierBytes)
	_, _ = rand.Read(value)

	return hex.EncodeToString(value)
}

const (
	maxPageElements    = 10000
	maxImportElements  = 50000
	maxStrokePoints    = 20000
	maxWarnings        = 20
	maxTypeLength      = 64
	maxAssetPath       = 128
	identifierBytes    = 16
	minDimension       = .1
	defaultPenWidth    = 5
	degreesPerHalfTurn = 180
)

func (c *converter) background(source sourcePage) (canvas.Settings, error) {
	settings := canvas.NewDocument().Settings
	if len(source.Backgrounds) == 0 {
		return settings, nil
	}
	if len(source.Backgrounds) > 1 {
		c.warn("Only the first page background is imported.")
	}
	background := source.Backgrounds[0]
	if background.Type != "texture" || background.Texture.BoxColor == nil {
		c.warn("Unsupported page backgrounds were replaced by the default canvas background.")

		return settings, nil
	}
	color, err := argb(background.Texture.BoxColor, "#ffffff")
	if err != nil {
		return settings, err
	}
	settings.Background = "plain"
	settings.BackgroundColor = color
	if background.Texture.BoxWidth != 0 || background.Texture.BoxHeight != 0 {
		c.warn("Background grids are replaced by their solid page color.")
	}

	return settings, nil
}

const (
	minStrokeCoordinates = 4
	pointDimensions      = 2
	alphaShift           = 24
	opaqueAlpha          = 255
	rgbMask              = 0xffffff
	elementImage         = "image"
)
