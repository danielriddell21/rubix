//go:build !ebiten

// The recorder is tag-free, so it is exercised in the default (headless) build. Under the
// ebiten tag the gui package pulls in Ebiten's init, which needs a display, so the test
// stays out of that build — the recorder code under test is identical either way.

package gui

import (
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"
)

// solidRGBA returns a w×h image filled with c.
func solidRGBA(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestRecorderSaveDownscalesAndEncodes(t *testing.T) {
	rec := newRecorder(2, 25)
	colors := []color.RGBA{
		{255, 0, 0, 255},
		{0, 255, 0, 255},
		{0, 0, 255, 255},
	}
	for _, c := range colors {
		rec.add(solidRGBA(100, 80, c))
	}
	if rec.len() != len(colors) {
		t.Fatalf("len() = %d, want %d", rec.len(), len(colors))
	}

	path := filepath.Join(t.TempDir(), "out.gif")
	if err := rec.save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(g.Image) != len(colors) {
		t.Fatalf("decoded %d frames, want %d", len(g.Image), len(colors))
	}
	b := g.Image[0].Bounds()
	if b.Dx() != 50 || b.Dy() != 40 {
		t.Fatalf("frame bounds = %dx%d, want 50x40 (downscaled by 2)", b.Dx(), b.Dy())
	}
	if g.Delay[0] == 0 {
		t.Fatalf("frame delay = 0, want non-zero (100/25 = 4)")
	}
	if g.Delay[0] != 4 {
		t.Errorf("frame delay = %d, want 4", g.Delay[0])
	}
}

func TestNewRecorderClampsParams(t *testing.T) {
	r := newRecorder(0, 0) // scale<1, fps<1
	if r.scale != 1 {
		t.Errorf("scale = %d, want clamped to 1", r.scale)
	}
	if r.delay != 100 {
		t.Errorf("delay = %d, want 100 (fps clamped to 1)", r.delay)
	}
}
