package image

import (
	"math"
)

// AssetInfo describes an original image and its available preview sizes.
type AssetInfo struct {
	ID       string `json:"id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	MIME     string `json:"mime"`
	Variants []int  `json:"variants"`
}

// Blob contains image bytes and their display metadata.
type Blob struct {
	Data          []byte
	MIME          string
	Width, Height int
}

// RequestVariantCommand supplies the canvas dimensions needed for an image preview.
type RequestVariantCommand struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// VariantBucket selects a preview size only when a useful downscale is possible.
func VariantBucket(ow, oh int, rw, rh float64) int {
	scale := math.Min(1, math.Max(rw/float64(ow), rh/float64(oh)))
	if scale >= variantScaleThreshold {
		return 0
	}
	long := math.Max(float64(ow), float64(oh))
	target := long * scale
	for _, buffer := range []int{128, 256, 512, 1024, 2048, 4096, 8192} {
		if float64(buffer) >= target {
			// One stable asset-specific bucket bridges the gap between the
			// power-of-two sizes and the strict 20% reduction threshold.
			limit := int(math.Ceil(long*variantScaleThreshold)) - 1
			buffer = min(buffer, limit)
			if float64(buffer) < target || buffer < 1 {
				return 0
			}

			return buffer
		}
	}

	return 0
}
