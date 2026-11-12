package domain

import "math"

type AssetInfo struct {
	ID       string `json:"id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	MIME     string `json:"mime"`
	Variants []int  `json:"variants"`
}

func VariantBucket(ow, oh int, rw, rh float64) int {
	scale := math.Min(1, math.Max(rw/float64(ow), rh/float64(oh)))
	if scale >= .8 {
		return 0
	}
	long := math.Max(float64(ow), float64(oh))
	target := long * scale
	for _, b := range []int{128, 256, 512, 1024, 2048, 4096, 8192} {
		if float64(b) >= target {
			// One stable asset-specific bucket bridges the gap between the
			// power-of-two sizes and the strict 20% reduction threshold.
			cap := int(math.Ceil(long*.8)) - 1
			b = min(b, cap)
			if float64(b) < target || b < 1 {
				return 0
			}
			return b
		}
	}
	return 0
}

type Blob struct {
	Data          []byte
	MIME          string
	Width, Height int
}
