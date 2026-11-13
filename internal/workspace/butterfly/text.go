package butterfly

import (
	"math"
	"mindflow/internal/workspace/page/canvas"
	"strings"
	"unicode/utf8"
)

func (c *converter) text(source sourceElement) (canvas.Element, error) {
	var value strings.Builder
	for _, span := range source.Area.Paragraph.TextSpans {
		value.WriteString(span.Text)
	}
	text := value.String()
	if text == "" {
		return canvas.Element{}, nil
	}
	if len(text) > maxTextBytes {
		return canvas.Element{}, invalid("An imported text element exceeds 100,000 bytes.")
	}
	paragraph, style := resolveStyle(source)
	foreground := source.Foreground
	if style.Color != nil {
		foreground = style.Color
	}
	color, err := argb(foreground, "#000000")
	if err != nil {
		return canvas.Element{}, err
	}
	size := math.Round(number(style.Size, defaultFontSize) * number(source.Scale, 1))
	if size < 1 || size > maxFontSize {
		return canvas.Element{}, invalid("Imported text size must be between 1 and 2,000.")
	}
	width := minTextWidth
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		width = math.Max(width, float64(utf8.RuneCountInString(line))*size)
	}
	if source.Constraint.Size > 0 {
		width = math.Min(width, source.Constraint.Size)
	}
	height := float64(len(lines)) * size * lineHeight
	if source.Constraint.Length > 0 {
		height = math.Max(height, source.Constraint.Length)
	}
	align := paragraph.Alignment
	if align != "left" && align != "center" && align != "right" {
		align = "left"
	}
	vertical := source.Area.AreaProperty.Alignment
	if vertical == "center" {
		vertical = "middle"
	}
	if vertical != "middle" && vertical != "bottom" {
		vertical = "top"
	}
	bold := style.FontWeight != nil && *style.FontWeight >= boldFontWeight
	italic := style.Italic != nil && *style.Italic
	c.warn("Text uses one style per element and mindflow's font metrics; mixed formatting and advanced text layout are flattened.")

	return canvas.Element{Type: "text", X: source.Position.X + width/2, Y: source.Position.Y + height/2, Width: width, Height: height, Text: text, FontSize: size, Color: color, Align: align, VerticalAlign: vertical, Bold: bold, Italic: italic}, nil
}

const (
	defaultFontSize = 12
	maxFontSize     = 2000
	boldFontWeight  = 6
	lineHeight      = 1.3
	minTextWidth    = 1.0
	maxTextBytes    = 100000
)

func resolveStyle(source sourceElement) (sourceStyle, sourceStyle) {
	paragraph := source.Area.Paragraph.Property
	if paragraph.Type == "named" {
		paragraph = source.StyleSheet.Item.ParagraphProperties[paragraph.Name]
	}
	style := sourceStyle{}
	if paragraph.Span != nil {
		style = *paragraph.Span
	}
	for _, span := range source.Area.Paragraph.TextSpans {
		if span.Text == "" {
			continue
		}
		own := span.Property
		if own.Type == "named" {
			own = source.StyleSheet.Item.SpanProperties[own.Name]
		}
		if own.Size != nil {
			style.Size = own.Size
		}
		if own.Color != nil {
			style.Color = own.Color
		}
		if own.FontWeight != nil {
			style.FontWeight = own.FontWeight
		}
		if own.Italic != nil {
			style.Italic = own.Italic
		}

		break
	}

	return paragraph, style
}
