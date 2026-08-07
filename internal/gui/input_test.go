package gui

import (
	"testing"

	"github.com/danielriddell21/rubix/pkg/cube"
)

// testController drives the viewer without a solver: it scrambles from a fixed
// seed and returns no moves, which is enough to step the state machine.
func testController() Controller {
	return Controller{
		Strategies: []string{"test"},
		Scramble:   func() cube.Cube { return cube.ScrambledCube(25, 1) },
		Solve:      func(string, cube.Cube) ([]cube.Move, bool) { return nil, false },
	}
}

func TestBuildScriptEmpty(t *testing.T) {
	s := buildScript("  ,  ", 120)
	if s == nil {
		t.Fatal("buildScript returned nil; an empty script must still be usable")
	}
	for f := range 130 {
		s.frame = f
		if s.Down(keyArrowLeft) || s.Tapped(keySpace) {
			t.Fatalf("frame %d: empty script pressed something", f)
		}
	}
}

func TestBuildScriptToggleFiresTwice(t *testing.T) {
	// A lone toggle shows both states: once after the intro, once halfway.
	s := buildScript("space", 120)
	want := map[int]bool{12: true, 60: true}
	for f := range 130 {
		s.frame = f
		if got := s.Tapped(keySpace); got != want[f] {
			t.Errorf("frame %d: Tapped(space) = %v, want %v", f, got, want[f])
		}
	}
}

func TestBuildScriptTapsAreSpaced(t *testing.T) {
	// Successive taps land one gap apart, in the order given.
	s := buildScript("plus,plus,tab", 120)
	at := func(f int, k key) bool { s.frame = f; return s.Tapped(k) }
	for _, f := range []int{12, 30} {
		if !at(f, keyPlus) {
			t.Errorf("frame %d: want a plus tap", f)
		}
	}
	if !at(48, keyTab) {
		t.Error("frame 48: want a tab tap")
	}
	if at(48, keyPlus) {
		t.Error("frame 48: plus tapped a third time")
	}
	// A repeated toggle is not lone, so it does not fire again at the midpoint.
	if at(60, keyPlus) || at(60, keyTab) {
		t.Error("frame 60: a multi-key script must not fire at the midpoint")
	}
}

func TestBuildScriptHoldsForTheWholeClip(t *testing.T) {
	// A modified hold presses both keys, from the intro to the last frame.
	s := buildScript("shift+up", 120)
	cases := []struct {
		frame int
		want  bool
	}{{0, false}, {11, false}, {12, true}, {60, true}, {120, true}, {121, false}}
	for _, c := range cases {
		s.frame = c.frame
		for _, k := range []key{keyShift, keyArrowUp} {
			if got := s.Down(k); got != c.want {
				t.Errorf("frame %d: Down(%v) = %v, want %v", c.frame, k, got, c.want)
			}
		}
	}
}

func TestScriptDrivesTheViewer(t *testing.T) {
	// The point of the abstraction: a script steps the real controller with no
	// display. Pressing space unfolds the cube to its flat net.
	s := buildScript("space", 60)
	g := newState(Config{Controller: testController()}, s)
	if g.unfoldTo != 0 {
		t.Fatalf("unfoldTo = %v before the script runs, want 0", g.unfoldTo)
	}
	for range 20 {
		s.advance()
		if err := g.step(); err != nil {
			t.Fatalf("step: %v", err)
		}
	}
	if g.unfoldTo != 1 {
		t.Errorf("unfoldTo = %v after the space tap, want 1", g.unfoldTo)
	}
	if g.unfold <= 0 {
		t.Errorf("unfold = %v, want the net to have started opening", g.unfold)
	}
}

func TestScriptIgnoresWheelAndDrag(t *testing.T) {
	s := buildScript("left", 60)
	if dy := s.Wheel(); dy != 0 {
		t.Errorf("Wheel() = %v, want 0", dy)
	}
	if _, _, ok := s.Drag(); ok {
		t.Error("Drag() reported a drag; a script has no pointer")
	}
}
