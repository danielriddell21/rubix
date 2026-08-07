//go:build ebiten

package gui

import (
	"errors"
	"fmt"
	"math"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebinput "github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/danielriddell21/crucible/canvas"
	"github.com/danielriddell21/crucible/window"
)

func Available() bool { return true }

// ebitenKeys maps each control to the physical keys that trigger it. Several
// controls answer to two keys so the number pad works like the main row.
var ebitenKeys = map[key][]eb.Key{
	keyArrowLeft:  {eb.KeyArrowLeft},
	keyArrowRight: {eb.KeyArrowRight},
	keyArrowUp:    {eb.KeyArrowUp},
	keyArrowDown:  {eb.KeyArrowDown},
	keyShift:      {eb.KeyShiftLeft, eb.KeyShiftRight},
	keySpace:      {eb.KeySpace},
	keyX:          {eb.KeyX},
	keyEnter:      {eb.KeyEnter, eb.KeyKPEnter},
	keyR:          {eb.KeyR},
	keyS:          {eb.KeyS},
	keyTab:        {eb.KeyTab},
	keyM:          {eb.KeyM},
	keyPlus:       {eb.KeyEqual, eb.KeyKPAdd},
	keyMinus:      {eb.KeyMinus, eb.KeyKPSubtract},
	keyU:          {eb.KeyU},
	keyF:          {eb.KeyF},
	keyD:          {eb.KeyD},
	keyL:          {eb.KeyL},
	keyB:          {eb.KeyB},
}

// liveInput reads the keyboard, wheel and mouse from the window.
type liveInput struct{}

func (liveInput) Down(k key) bool {
	for _, ek := range ebitenKeys[k] {
		if eb.IsKeyPressed(ek) {
			return true
		}
	}
	return false
}

func (liveInput) Tapped(k key) bool {
	for _, ek := range ebitenKeys[k] {
		if ebinput.IsKeyJustPressed(ek) {
			return true
		}
	}
	return false
}

func (liveInput) Wheel() float64 {
	_, dy := eb.Wheel()
	return dy
}

func (liveInput) Drag() (int, int, bool) {
	if !eb.IsMouseButtonPressed(eb.MouseButtonLeft) {
		return 0, 0, false
	}
	x, y := eb.CursorPosition()
	return x, y, true
}

// game adapts the display-free [gameState] to Ebiten's game loop.
type game struct {
	state  *gameState
	canvas *canvas.Canvas
}

func (g *game) Update() error {
	if err := g.state.step(); err != nil {
		if errors.Is(err, errQuit) {
			return eb.Termination
		}
		return err
	}
	return nil
}

func (g *game) Draw(screen *eb.Image) {
	s := g.state
	if g.canvas == nil || func() bool { w, h := g.canvas.Size(); return w != s.w || h != s.h }() {
		g.canvas = canvas.New(s.w, s.h)
	}
	DrawScene(g.canvas, s.scene())
	screen.WritePixels(g.canvas.Pixels())
}

func (g *game) Layout(outsideW, outsideH int) (int, int) {
	if outsideW > 0 && outsideH > 0 {
		g.state.w, g.state.h = outsideW, outsideH
	}
	return g.state.w, g.state.h
}

// fitMonitor shrinks a window that would not fit on the monitor.
func fitMonitor(w, h int) (int, int) {
	mw, mh := eb.Monitor().Size()
	if mw <= 0 || mh <= 0 {
		return w, h
	}
	maxW, maxH := mw*9/10, mh*9/10
	s := 1.0
	if w > maxW {
		s = math.Min(s, float64(maxW)/float64(w))
	}
	if h > maxH {
		s = math.Min(s, float64(maxH)/float64(h))
	}
	if s < 1 {
		w, h = int(float64(w)*s), int(float64(h)*s)
	}
	return w, h
}

func Run(cfg Config) error {
	g := newState(cfg, liveInput{})
	g.resize = func(cols, rows int) (int, int) {
		w, h := fitMonitor(windowSize(cols, rows))
		eb.SetWindowSize(w, h)
		return w, h
	}

	ctrl := cfg.Controller
	title := ctrl.Title
	if title == "" {
		title = "rubix"
	}
	// The user may resize the window; Layout reflows the scene to the new size.
	window.Configure(window.Options{Title: title, Width: g.w, Height: g.h, MinWidth: winW / 2, MinHeight: winH / 2})
	if ctrl.OffsetIndex > 0 {
		// Cascade child windows so they don't open exactly on top of the leader.
		eb.SetWindowPosition(60+ctrl.OffsetIndex*36, 60+ctrl.OffsetIndex*36)
	}
	if cfg.Link != nil {
		// Coordinated windows must keep updating while unfocused, so background windows
		// stay in sync, keep animating, and notice the leader closing (their Update must
		// run to drain the closed link and terminate).
		eb.SetRunnableOnUnfocused(true)
	}
	if err := eb.RunGame(&game{state: g}); err != nil {
		return fmt.Errorf("run window: %w", err)
	}
	return nil
}
