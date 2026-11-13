package butterfly

import (
	"encoding/json"
	"mindflow/internal/workspace/page/canvas"
)

type sourcePoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type sourcePage struct {
	Layers []struct {
		Content []sourceElement `json:"content"`
	} `json:"layers"`
	Content     []sourceElement `json:"content"`
	Backgrounds []struct {
		Type    string `json:"type"`
		Texture struct {
			BoxColor  *int64  `json:"boxColor"`
			BoxWidth  float64 `json:"boxWidth"`
			BoxHeight float64 `json:"boxHeight"`
		} `json:"texture"`
	} `json:"backgrounds"`
	Animations []json.RawMessage `json:"animations"`
	Areas      []json.RawMessage `json:"areas"`
	Waypoints  []json.RawMessage `json:"waypoints"`
}
type sourcePaint struct {
	Type  string `json:"type"`
	Color *int64 `json:"color"`
}
type sourceProperty struct {
	StrokeWidth *float64    `json:"strokeWidth"`
	Paint       sourcePaint `json:"paint"`
	FillPaint   sourcePaint `json:"fillPaint"`
	Shape       struct {
		Type      string      `json:"type"`
		FillPaint sourcePaint `json:"fillPaint"`
	} `json:"shape"`
}
type sourceStyle struct {
	Type       string       `json:"type"`
	Name       string       `json:"name"`
	Size       *float64     `json:"size"`
	Color      *int64       `json:"color"`
	FontWeight *int         `json:"fontWeight"`
	Italic     *bool        `json:"italic"`
	Alignment  string       `json:"alignment"`
	Span       *sourceStyle `json:"span"`
}
type sourceElement struct {
	Type           string         `json:"type"`
	Rotation       float64        `json:"rotation"`
	Shear          float64        `json:"shear"`
	Collection     string         `json:"collection"`
	Position       sourcePoint    `json:"position"`
	FirstPosition  sourcePoint    `json:"firstPosition"`
	SecondPosition sourcePoint    `json:"secondPosition"`
	Points         []sourcePoint  `json:"points"`
	Property       sourceProperty `json:"property"`
	Source         string         `json:"source"`
	Width          float64        `json:"width"`
	Height         float64        `json:"height"`
	Constraints    struct {
		Type   string   `json:"type"`
		ScaleX *float64 `json:"scaleX"`
		ScaleY *float64 `json:"scaleY"`
		Width  float64  `json:"width"`
		Height float64  `json:"height"`
	} `json:"constraints"`
	Scale      *float64 `json:"scale"`
	Foreground *int64   `json:"foreground"`
	Constraint struct {
		Size   float64 `json:"size"`
		Length float64 `json:"length"`
	} `json:"constraint"`
	StyleSheet struct {
		Item struct {
			FontFamily          string                 `json:"fontFamily"`
			SpanProperties      map[string]sourceStyle `json:"spanProperties"`
			ParagraphProperties map[string]sourceStyle `json:"paragraphProperties"`
		} `json:"item"`
	} `json:"styleSheet"`
	Area struct {
		AreaProperty struct {
			Alignment string `json:"alignment"`
		} `json:"areaProperty"`
		Paragraph struct {
			Property  sourceStyle `json:"property"`
			TextSpans []struct {
				Text     string      `json:"text"`
				Property sourceStyle `json:"property"`
				Type     string      `json:"type"`
			} `json:"textSpans"`
		} `json:"paragraph"`
	} `json:"area"`
}
type converter struct {
	archive  *Archive
	elements int
}

// Page contains a converted editable canvas and its source title/order.
type Page struct {
	Name     string
	Document canvas.Document
}
