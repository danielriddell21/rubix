package gui

// The recorder turns a stream of rendered frames into an animated GIF. It is deliberately
// free of any build tag and of any Ebiten dependency: it works on plain image.Image
// values, so it can be unit-tested headlessly. The Ebiten capture path in gui.go feeds it
// the screen pixels each frame.

import (
	"image"
	colorpalette "image/color/palette"
	"image/draw"
	"image/gif"
	"os"
)

// recorder accumulates downscaled, palettized frames and writes them as a looping GIF.
type recorder struct {
	scale  int // integer nearest-neighbour downscale factor
	delay  int // per-frame delay in 1/100s (100/fps)
	frames []*image.Paletted
}

// newRecorder makes a recorder that downscales each frame by scale (nearest-neighbour)
// and plays back at fps frames per second. scale is clamped to >=1 and fps to 1..100.
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

// add captures one frame: an integer nearest-neighbour downscale, then Floyd–Steinberg
// dithering onto the standard Plan9 256-colour palette.
func (r *recorder) add(img image.Image) {
	small := downscale(img, r.scale)
	p := image.NewPaletted(small.Bounds(), colorpalette.Plan9)
	draw.FloydSteinberg.Draw(p, small.Bounds(), small, small.Bounds().Min)
	r.frames = append(r.frames, p)
}

// len reports how many frames have been captured so far.
func (r *recorder) len() int { return len(r.frames) }

// save writes the captured frames as a single looping animated GIF.
func (r *recorder) save(path string) error {
	g := &gif.GIF{LoopCount: 0}
	for _, f := range r.frames {
		g.Image = append(g.Image, f)
		g.Delay = append(g.Delay, r.delay)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := gif.EncodeAll(f, g); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// downscale shrinks src by an integer factor using nearest-neighbour sampling, always
// returning a freshly owned *image.RGBA (a factor of 1 yields a same-size copy).
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
