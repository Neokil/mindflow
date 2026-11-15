package onenote

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // Register JPEG decoding for transformed imported images.
	"image/png"
	"math"
	"mindflow/internal/workspace/page/canvas"
	"strconv"
	"strings"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp" // Register WebP decoding for transformed imported images.
)

const (
	redShift       = 24
	greenShift     = 16
	diameter       = 2
	rasterScale    = 2
	maxRasterEdge  = 2048
	starPoints     = 10
	starArms       = 5
	minimumRadius  = .01
	hexColorLength = 7
	byteBits       = 8
	colorMask      = 255
	fontDPI        = 72
	textLeading    = 1.3
	fixedUnits     = 64
)

// RasterFallback preserves visual objects which have no supported native OneNote representation.
// Text and ink normally bypass this path and remain editable in the destination.
func RasterFallback(ctx context.Context, element canvas.Element, media []byte) ([]byte, canvas.Element, error) {
	sine, cosine := math.Sincos(element.Rotation)
	padding := 0.0
	if element.Type == "shape" {
		padding = element.BorderWidth
	}
	width := math.Abs(element.Width*cosine) + math.Abs(element.Height*sine) + padding
	height := math.Abs(element.Width*sine) + math.Abs(element.Height*cosine) + padding
	scale := math.Min(rasterScale, maxRasterEdge/math.Max(width, height))
	target := image.NewNRGBA(image.Rect(0, 0, max(1, int(math.Ceil(width*scale))), max(1, int(math.Ceil(height*scale)))))
	var err error
	switch element.Type {
	case "image":
		err = rasterImage(target, element, media, scale, sine, cosine)
	case "shape":
		err = rasterShape(ctx, target, element, scale, sine, cosine)
	case "text":
		err = rasterText(target, element, scale, sine, cosine)
	default:
		return nil, element, invalid("This object cannot be represented in OneNote.")
	}
	if err != nil {
		return nil, element, err
	}
	var output bytes.Buffer
	err = png.Encode(&output, target)
	if err != nil {
		return nil, element, fmt.Errorf("encode OneNote fallback: %w", err)
	}
	element = canvas.Element{ID: element.ID, Type: "image", X: element.X, Y: element.Y, Width: width, Height: height}

	return output.Bytes(), element, nil
}

func rasterImage(target *image.NRGBA, element canvas.Element, media []byte, scale, sine, cosine float64) error {
	source, _, err := image.Decode(bytes.NewReader(media))
	if err != nil {
		return fmt.Errorf("decode OneNote image: %w", err)
	}
	bounds := source.Bounds()
	if crop := element.Crop; crop != nil {
		bounds = image.Rect(bounds.Min.X+int(crop.X*float64(bounds.Dx())), bounds.Min.Y+int(crop.Y*float64(bounds.Dy())), bounds.Min.X+int(math.Ceil((crop.X+crop.Width)*float64(bounds.Dx()))), bounds.Min.Y+int(math.Ceil((crop.Y+crop.Height)*float64(bounds.Dy())))).Intersect(source.Bounds())
	}
	if bounds.Empty() {
		return invalid("The image crop is empty.")
	}
	affineImage(target, source, bounds, element.Width*scale, element.Height*scale, sine, cosine)

	return nil
}

func affineImage(target *image.NRGBA, source image.Image, bounds image.Rectangle, width, height, sine, cosine float64) {
	sx, sy := width/float64(bounds.Dx()), height/float64(bounds.Dy())
	cx, cy := float64(bounds.Min.X+bounds.Max.X)/diameter, float64(bounds.Min.Y+bounds.Max.Y)/diameter
	tx := float64(target.Bounds().Dx())/diameter - cosine*sx*cx + sine*sy*cy
	ty := float64(target.Bounds().Dy())/diameter - sine*sx*cx - cosine*sy*cy
	xdraw.BiLinear.Transform(target, f64.Aff3{cosine * sx, -sine * sy, tx, sine * sx, cosine * sy, ty}, source, bounds, xdraw.Over, nil)
}

func rasterShape(ctx context.Context, target *image.NRGBA, element canvas.Element, scale, sine, cosine float64) error {
	fill, border := rasterColor(element.Fill), rasterColor(element.Border)
	points := make([][2]float64, 0, starPoints)
	if element.Shape == "star" {
		for i := range starPoints {
			angle := float64(i)*math.Pi/starArms - math.Pi/diameter
			radius := .5
			if i%2 == 1 {
				radius = .25
			}
			points = append(points, [2]float64{math.Cos(angle) * element.Width * radius, math.Sin(angle) * element.Height * radius})
		}
	}
	halfWidth, halfHeight := element.Width/diameter, element.Height/diameter
	for pixelY := range target.Bounds().Dy() {
		err := ctx.Err()
		if err != nil {
			return fmt.Errorf("render OneNote shape: %w", err)
		}
		for pixelX := range target.Bounds().Dx() {
			dx := (float64(pixelX) + .5 - float64(target.Bounds().Dx())/diameter) / scale
			dy := (float64(pixelY) + .5 - float64(target.Bounds().Dy())/diameter) / scale
			localX, localY := dx*cosine+dy*sine, -dx*sine+dy*cosine
			inside, edge := false, false
			switch element.Shape {
			case "rectangle":
				inside = math.Abs(localX) <= halfWidth && math.Abs(localY) <= halfHeight
				edge = math.Abs(localX) <= halfWidth+element.BorderWidth/diameter && math.Abs(localY) <= halfHeight+element.BorderWidth/diameter && (math.Abs(localX) >= halfWidth-element.BorderWidth/diameter || math.Abs(localY) >= halfHeight-element.BorderWidth/diameter)
			case "circle":
				inside = localX*localX/(halfWidth*halfWidth)+localY*localY/(halfHeight*halfHeight) <= 1
				outerX, outerY := halfWidth+element.BorderWidth/diameter, halfHeight+element.BorderWidth/diameter
				innerX, innerY := math.Max(minimumRadius, halfWidth-element.BorderWidth/diameter), math.Max(minimumRadius, halfHeight-element.BorderWidth/diameter)
				edge = localX*localX/(outerX*outerX)+localY*localY/(outerY*outerY) <= 1 && localX*localX/(innerX*innerX)+localY*localY/(innerY*innerY) >= 1
			case "star":
				inside, edge = starCoverage(localX, localY, points, element.BorderWidth)
			}
			if inside {
				target.SetNRGBA(pixelX, pixelY, fill)
			}
			if edge && element.BorderWidth > 0 {
				target.SetNRGBA(pixelX, pixelY, border)
			}
		}
	}

	return nil
}

func starCoverage(localX, localY float64, points [][2]float64, width float64) (bool, bool) {
	inside, edge := false, false
	for i, start := range points {
		end := points[(i+1)%len(points)]
		if (start[1] > localY) != (end[1] > localY) && localX < (end[0]-start[0])*(localY-start[1])/(end[1]-start[1])+start[0] {
			inside = !inside
		}
		vx, vy := end[0]-start[0], end[1]-start[1]
		length := vx*vx + vy*vy
		if length > 0 {
			t := math.Max(0, math.Min(1, ((localX-start[0])*vx+(localY-start[1])*vy)/length))
			if math.Hypot(localX-start[0]-t*vx, localY-start[1]-t*vy) <= width/diameter {
				edge = true
			}
		}
	}

	return inside, edge
}

func rasterColor(value string) color.NRGBA {
	if value == "transparent" || value == "" {
		return color.NRGBA{}
	}
	raw, _ := strconv.ParseUint(strings.TrimPrefix(value, "#"), 16, 32)
	if len(value) == hexColorLength {
		raw = raw<<byteBits | colorMask
	}

	return color.NRGBA{R: uint8((raw >> redShift) & colorMask), G: uint8((raw >> greenShift) & colorMask), B: uint8((raw >> byteBits) & colorMask), A: uint8(raw & colorMask)}
}

func rasterText(target *image.NRGBA, element canvas.Element, scale, sine, cosine float64) error {
	data := goregular.TTF
	if element.Bold {
		data = gobold.TTF
	}
	parsed, err := opentype.Parse(data)
	if err != nil {
		return fmt.Errorf("load fallback font: %w", err)
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: element.FontSize * scale, DPI: fontDPI, Hinting: font.HintingFull})
	if err != nil {
		return fmt.Errorf("create fallback font: %w", err)
	}
	defer func() { _ = face.Close() }()
	local := image.NewNRGBA(image.Rect(0, 0, max(1, int(math.Ceil(element.Width*scale))), max(1, int(math.Ceil(element.Height*scale)))))
	drawer := font.Drawer{Dst: local, Src: image.NewUniform(rasterColor(element.Color)), Face: face}
	lines := strings.Split(element.Text, "\n")
	lineHeight := element.FontSize * scale * textLeading
	top := 0.0
	switch element.VerticalAlign {
	case "middle":
		top = (float64(local.Bounds().Dy()) - float64(len(lines))*lineHeight) / diameter
	case "bottom":
		top = float64(local.Bounds().Dy()) - float64(len(lines))*lineHeight
	}
	for index, line := range lines {
		localX := 0.0
		switch element.Align {
		case "center":
			localX = (float64(local.Bounds().Dx()) - float64(drawer.MeasureString(line))/fixedUnits) / diameter
		case "right":
			localX = float64(local.Bounds().Dx()) - float64(drawer.MeasureString(line))/fixedUnits
		}
		drawer.Dot = fixed.P(int(localX), int(top+float64(index)*lineHeight+element.FontSize*scale))
		drawer.DrawString(line)
	}
	affineImage(target, local, local.Bounds(), element.Width*scale, element.Height*scale, sine, cosine)

	return nil
}
