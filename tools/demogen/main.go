// Command demogen renders rubix's documentation media headlessly: one short
// clip per keybind, each replaying a scripted key sequence against the same
// controller the window drives and composing frames on a software canvas — so
// it needs no display and no ebiten build tag.
//
// Regenerate every asset under docs/demos with:
//
//	just demos      // or: go run ./tools/demogen
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/danielriddell21/crucible/record"

	"github.com/danielriddell21/rubix/internal/cli"
	"github.com/danielriddell21/rubix/internal/gui"
)

const outDir = "docs/demos"

// seed fixes the scramble so every clip replays identically.
const seed = 1

// The recording settings the clips share: rubix draws a large square window,
// so the media halves it.
const (
	frames = 120
	fps    = 25
	scale  = 2
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "demogen:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	for _, c := range clips() {
		if err := c.record(); err != nil {
			return fmt.Errorf("%s: %w", c.name, err)
		}
	}
	return nil
}

// clip is one recorded run: a name, and the keybinds it presses.
type clip struct {
	name string
	// ext is the output extension; an .mp4 records video instead of a GIF.
	ext string
	// keys is the scripted keybind sequence, comma-separated.
	keys string
}

// clips is the documentation set: one clip per control, so the README can show
// each keybind doing its thing.
func clips() []clip {
	return []clip{
		{name: "unfold", ext: ".gif", keys: "space"},
		{name: "xray", ext: ".gif", keys: "x"},
		{name: "rescramble", ext: ".gif", keys: "r"},
		{name: "cycle-solver", ext: ".gif", keys: "s"},
		{name: "move-list", ext: ".gif", keys: "m"},
		{name: "orbit", ext: ".gif", keys: "left"},
		{name: "zoom", ext: ".gif", keys: "shift+up"},
		{name: "compare", ext: ".gif", keys: "plus,plus,plus,tab,tab,tab"},
	}
}

func (c clip) record() error {
	cfg := gui.Config{
		Controller: cli.DemoController(seed),
		Keys:       c.keys,
		Rec: record.Options{
			Path:   filepath.Join(outDir, c.name+c.ext),
			Frames: frames,
			FPS:    fps,
			Scale:  scale,
		},
	}
	if err := gui.Render(cfg); err != nil {
		return fmt.Errorf("capture clip: %w", err)
	}
	return nil
}
