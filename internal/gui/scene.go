package gui

import (
	"image/color"
	"math"
)

const (
	winW       = 640
	scale      = 40
	turnFrames = 9
)

var (
	plastic    = color.RGBA{34, 34, 40, 255}
	outline    = color.RGBA{12, 12, 14, 255}
	background = color.RGBA{30, 30, 36, 255}
	focusLine  = color.RGBA{200, 200, 80, 255}
	cellLine   = color.RGBA{70, 70, 80, 255}
	highlight  = color.RGBA{70, 70, 30, 255}
)

func wrapHelp(segs []string, maxWidth int) []string {
	const glyph = 6
	const sep = "   "
	var lines []string
	cur := ""
	for _, s := range segs {
		cand := s
		if cur != "" {
			cand = cur + sep + s
		}
		if cur != "" && len(cand)*glyph > maxWidth {
			lines = append(lines, cur)
			cur = s
		} else {
			cur = cand
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func shade(c color.RGBA, f float32) color.RGBA {
	f = clamp(f, 0, 1)
	return color.RGBA{uint8(float32(c.R) * f), uint8(float32(c.G) * f), uint8(float32(c.B) * f), c.A}
}

func clamp(v, lo, hi float32) float32 { return max(lo, min(hi, v)) }

func fsincos(a float32) (float32, float32) {
	s, c := math.Sincos(float64(a))
	return float32(s), float32(c)
}
