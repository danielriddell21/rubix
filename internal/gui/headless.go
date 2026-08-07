package gui

import (
	"fmt"
	"image"

	"github.com/danielriddell21/crucible/canvas"
	"github.com/danielriddell21/crucible/demo"
	"github.com/danielriddell21/crucible/record"
)

// defaultRecordFrames is how long a clip runs when the caller does not say.
const defaultRecordFrames = 120

// Render records a scripted run to cfg.Rec.Path without opening a window. It
// drives the same [gameState] the window drives, replaying cfg.Keys as if a
// player had pressed them, and composes frames with the same [DrawScene] — so
// the media matches what a player sees and needs no display.
//
// The file extension picks the format: .gif or .mp4.
func Render(cfg Config) error {
	frames := cfg.Rec.Frames
	if frames < 1 {
		frames = defaultRecordFrames
	}
	script := buildScript(cfg.Keys, frames)
	g := newState(cfg, script)
	c := canvas.New(g.w, g.h)
	// The cube is flat colour over a still background, and the canvas
	// anti-aliases its edges; frame diffing quantises without dithering, so
	// those edges stay put instead of churning between frames.
	rec := record.New(cfg.Rec, record.WithFrameDiff())

	clip := demo.Clip{
		Frames: frames,
		Step: func(int) error {
			script.advance()
			return g.step()
		},
		Frame: func(int) image.Image {
			DrawScene(c, g.scene())
			return record.FromRGBA(c.Pixels(), g.w, g.h)
		},
	}
	if _, err := clip.Record(rec); err != nil {
		return fmt.Errorf("capture run: %w", err)
	}
	if err := rec.Save(cfg.Rec.Path); err != nil {
		return fmt.Errorf("save recording: %w", err)
	}
	fmt.Printf("%s: %d frames\n", cfg.Rec.Path, rec.Len())
	return nil
}
