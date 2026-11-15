package butterfly

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"mindflow/internal/workspace/page/canvas"
	"strconv"
	"strings"
)

type exportElement map[string]any

// Write creates a version-13 Butterfly document from editable canvas content.
// Embedded media must already be authorized and bounded by the transfer service.
func Write(archive Archive) ([]byte, []string, error) {
	if len(archive.Pages) < 1 || len(archive.Pages) > maxPages {
		return nil, nil, invalid("Export between 1 and 100 pages.")
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	total := 0
	write := func(name string, data []byte) error {
		total += len(data)
		if total > maxExpandedBytes {
			return invalid("This Butterfly export exceeds 128 MiB.")
		}
		entry, err := writer.Create(name)
		if err != nil {
			return wrap("create Butterfly entry", err)
		}
		_, err = entry.Write(data)

		return wrap("write Butterfly entry", err)
	}
	metadata, err := json.Marshal(map[string]any{"fileVersion": fileVersion, fieldType: "document", "name": archive.Name})
	if err != nil {
		return nil, nil, wrap("encode Butterfly metadata", err)
	}
	err = write("META-INF/butterfly.json", metadata)
	if err != nil {
		return nil, nil, err
	}
	notes := archive.Warnings
	archive.Warnings = []string{}
	converter := converter{archive: &archive}
	for _, note := range notes {
		converter.warn(note)
	}
	count := 0
	for _, page := range archive.Pages {
		count += len(page.Document.Elements)
	}
	if count > maxImportElements {
		return nil, nil, invalid("A Butterfly transfer supports up to 50,000 elements.")
	}
	for index, page := range archive.Pages {
		data, pageErr := converter.exportPage(page.Document)
		if pageErr != nil {
			return nil, nil, pageErr
		}
		title := strings.NewReplacer("/", "_", "\\", "_").Replace(page.Name)
		err = write(fmt.Sprintf("BFLY/pages/%d.%s.json", index, title), data)
		if err != nil {
			return nil, nil, err
		}
	}
	info := []byte(`{"tools":[{fieldType:"select","mode":"lasso"},{fieldType:"hand"},{fieldType:elementPen,"property":{fieldType:elementPen,"strokeWidth":5,"paint":{fieldType:"solid","color":4278190080}}},{fieldType:"undo"},{fieldType:"redo"}]}`)
	err = write("BFLY/info.json", info)
	if err != nil {
		return nil, nil, err
	}
	for name, data := range archive.Images {
		if len(data) > maxImageBytes {
			return nil, nil, invalid("An exported Butterfly image exceeds 20 MiB.")
		}
		if !strings.HasPrefix(name, "BFLY/images/") || strings.Contains(strings.TrimPrefix(name, "BFLY/images/"), "/") || strings.Contains(name, "..") {
			return nil, nil, invalid("Invalid Butterfly image path.")
		}
		err = write(name, data)
		if err != nil {
			return nil, nil, err
		}
	}
	err = writer.Close()
	if err != nil {
		return nil, nil, wrap("close Butterfly export", err)
	}
	if output.Len() > maxArchiveBytes {
		return nil, nil, invalid("This Butterfly export exceeds 64 MiB.")
	}

	return output.Bytes(), archive.Warnings, nil
}

func (c *converter) exportPage(document canvas.Document) ([]byte, error) {
	err := canvas.ValidateDocument(document)
	if err != nil {
		return nil, err
	}
	content := make([]exportElement, 0, len(document.Elements))
	for _, element := range document.Elements {
		if element.Link != nil {
			c.warn("Element links are not exported to Butterfly.")
		}
		if element.Locked {
			c.warn("Group locks are not exported to Butterfly.")
		}
		converted, err := c.exportElement(element)
		if err != nil {
			return nil, err
		}
		content = append(content, converted)
	}
	data := map[string]any{"layers": []any{map[string]any{"name": "", "content": content}}}
	if document.Settings.BackgroundColor != "" {
		data["backgrounds"] = []any{map[string]any{fieldType: backgroundTexture, backgroundTexture: map[string]any{"boxColor": cssARGB(document.Settings.BackgroundColor), "boxWidth": 0, "boxHeight": 0}}}
	}
	encoded, err := json.Marshal(data)

	return encoded, wrap("encode Butterfly page", err)
}

func (c *converter) exportElement(element canvas.Element) (exportElement, error) {
	common := exportElement{"rotation": element.Rotation * 180 / math.Pi, "collection": element.GroupID, "position": exportPoint(element.X-element.Width/2, element.Y-element.Height/2)}
	switch element.Type {
	case "image":
		if _, exists := c.archive.Images[element.AssetID]; !exists {
			return nil, invalid("An exported Butterfly image is missing.")
		}
		if element.Crop != nil {
			return nil, invalid("Crop image media before Butterfly conversion.")
		}
		common[fieldType] = "image"
		common["source"] = element.AssetID
		common["width"] = element.Width
		common["height"] = element.Height
		common["constraints"] = map[string]any{fieldType: "scaled", "scaleX": 1, "scaleY": 1}
	case elementShape:
		if element.Shape == "star" {
			exportStar(common, element)
			c.warn("Stars are exported as editable closed pen paths.")

			return common, nil
		}
		common[fieldType] = elementShape
		common["firstPosition"] = exportPoint(element.X-element.Width/2, element.Y-element.Height/2)
		common["secondPosition"] = exportPoint(element.X+element.Width/2, element.Y+element.Height/2)
		common["property"] = map[string]any{fieldType: elementShape, "strokeWidth": element.BorderWidth, "paint": exportPaint(element.Border), elementShape: map[string]any{fieldType: element.Shape, "fillPaint": exportPaint(element.Fill)}}
	case elementText:
		c.exportText(common, element)
	case "stroke":
		c.exportStroke(common, element)
	default:
		return nil, invalid("Convert unsupported content before Butterfly export.")
	}

	return common, nil
}

func (c *converter) exportText(common exportElement, element canvas.Element) {
	text := exportTextContent(element)
	if element.ListStyle != "" {
		c.warn("Lists are exported as plain text; interactive checkboxes are not retained.")
	}
	weight := 3
	if element.Bold {
		weight = 6
	}
	style := map[string]any{fieldType: styleDefined, "size": element.FontSize, "color": cssARGB(element.Color), "fontWeight": weight, "italic": element.Italic}
	vertical := element.VerticalAlign
	if vertical == alignMiddle {
		vertical = alignCenter
	}
	common[fieldType] = elementText
	common["scale"] = 1
	common["foreground"] = cssARGB(element.Color)
	common["constraint"] = map[string]float64{"size": element.Width, "length": element.Height}
	common["area"] = map[string]any{"areaProperty": map[string]string{"alignment": vertical}, "paragraph": map[string]any{"property": map[string]any{fieldType: styleDefined, "alignment": element.Align, "span": style}, "textSpans": []any{map[string]any{fieldType: elementText, elementText: text, "property": style}}}}
	c.warn("Text uses destination font metrics; line wrapping can differ.")
}

func exportTextContent(element canvas.Element) string {
	if element.ListStyle == "" {
		return element.Text
	}
	checked := map[int]bool{}
	for _, index := range element.Checked {
		checked[index] = true
	}
	lines := strings.Split(element.Text, "\n")
	for index := range lines {
		prefix := "• "
		switch element.ListStyle {
		case "number":
			prefix = fmt.Sprintf("%d. ", index+1)
		case "check":
			prefix = "☐ "
			if checked[index] {
				prefix = "☑ "
			}
		}
		lines[index] = prefix + lines[index]
	}

	return strings.Join(lines, "\n")
}

func (c *converter) exportStroke(common exportElement, element canvas.Element) {
	points := make([]map[string]float64, 0, len(element.Points)/pointDimensions)
	for index := 0; index < len(element.Points); index += 2 {
		points = append(points, exportPoint(element.X+(element.Points[index]-exportHalf)*element.Width, element.Y+(element.Points[index+1]-exportHalf)*element.Height))
	}
	color := element.Color
	if element.Highlight {
		color = withAlpha(color, highlightAlpha)
		c.warn("Highlights use translucent ink; multiply blending can differ.")
	}
	common[fieldType] = elementPen
	common["points"] = points
	common["property"] = exportPen(element.PenWidth, color, "transparent")
	if len(element.Pressures) > 0 {
		c.warn("Pressure-sensitive ink is exported with constant stroke width.")
	}
}

func exportStar(common exportElement, element canvas.Element) {
	points := make([]map[string]float64, 0, starPointCount)
	for index := range starPointCount {
		angle := float64(index)*math.Pi/starArms - math.Pi/pointDimensions
		radius := .5
		if index%2 == 1 {
			radius = .25
		}
		points = append(points, exportPoint(element.X+math.Cos(angle)*element.Width*radius, element.Y+math.Sin(angle)*element.Height*radius))
	}
	common[fieldType] = elementPen
	common["points"] = points
	common["property"] = exportPen(math.Max(minDimension, element.BorderWidth), element.Border, element.Fill)
}
func exportPoint(x, y float64) map[string]float64 { return map[string]float64{"x": x, "y": y} }
func exportPaint(color string) map[string]any {
	return map[string]any{fieldType: "solid", "color": cssARGB(color)}
}

func exportPen(width float64, color, fill string) map[string]any {
	return map[string]any{fieldType: elementPen, "strokeWidth": width, "thinning": 0, "smoothing": 0, "streamline": 0, "paint": exportPaint(color), "fillPaint": exportPaint(fill)}
}

func cssARGB(color string) uint64 {
	if color == "" || color == "transparent" {
		return 0
	}
	value, _ := strconv.ParseUint(strings.TrimPrefix(color, "#"), 16, 32)
	if len(color) == rgbColorLength {
		return opaqueColorMask | value
	}

	return (value&opaqueAlpha)<<alphaShift | value>>bitsPerByte
}

func withAlpha(color string, alpha float64) string {
	value := cssARGB(color)
	a := uint64(math.Round(float64(value>>alphaShift) * alpha))

	return fmt.Sprintf("#%06x%02x", value&rgbMask, a)
}

const (
	elementPen        = "pen"
	elementShape      = "shape"
	elementText       = "text"
	backgroundTexture = "texture"
	alignCenter       = "center"
	alignMiddle       = "middle"
	styleDefined      = "defined"
	exportHalf        = .5
	highlightAlpha    = .35
	starArms          = 5
	starPointCount    = 11
	rgbColorLength    = 7
	opaqueColorMask   = 0xff000000
	bitsPerByte       = 8
)

const fieldType = "type"

const fileVersion = 13
