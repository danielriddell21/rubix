package gui

import (
	"fmt"
	"image"
	colorpalette "image/color/palette"
	"image/draw"
	"image/gif"
	"os"
)

type recorder struct {
	scale  int
	delay  int
	frames []*image.Paletted
}

func newRecorder(scale, fps int) *recorder {
	if scale < 1 {
		scale = 1
	}
	if fps < 1 {
		fps = 1
	}
	if fps > 100 {
		fps = 100
	}
	return &recorder{scale: scale, delay: 100 / fps}
}

func (r *recorder) add(img image.Image) {
	small := downscale(img, r.scale)
	p := image.NewPaletted(small.Bounds(), colorpalette.Plan9)
	draw.FloydSteinberg.Draw(p, small.Bounds(), small, small.Bounds().Min)
	r.frames = append(r.frames, p)
}

func (r *recorder) len() int { return len(r.frames) }

func (r *recorder) save(path string) error {
	g := &gif.GIF{LoopCount: 0}
	for _, f := range r.frames {
		g.Image = append(g.Image, f)
		g.Delay = append(g.Delay, r.delay)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if err := gif.EncodeAll(f, g); err != nil {
		_ = f.Close()
		return fmt.Errorf("encode gif: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	return nil
}

func downscale(src image.Image, factor int) *image.RGBA {
	if factor < 1 {
		factor = 1
	}
	b := src.Bounds()
	w, h := b.Dx()/factor, b.Dy()/factor
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := b.Min.Y + y*factor
		for x := 0; x < w; x++ {
			dst.Set(x, y, src.At(b.Min.X+x*factor, sy))
		}
	}
	return dst
}
