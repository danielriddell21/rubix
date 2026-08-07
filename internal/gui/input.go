package gui

import "strings"

// key names a control the viewer reacts to. The controller reads keys by these
// names rather than the display's own, which is what lets a scripted run drive
// exactly the same controller with no window and no display.
type key int

const (
	keyArrowLeft key = iota
	keyArrowRight
	keyArrowUp
	keyArrowDown
	keyShift
	keySpace
	keyX
	keyEnter
	keyR
	keyS
	keyTab
	keyM
	keyPlus
	keyMinus
	// The manual-turn face keys. R doubles as the rescramble key, which is
	// unambiguous because manual mode disables rescrambling.
	keyU
	keyF
	keyD
	keyL
	keyB
)

// input is where the controller reads the player from: held keys, taps, the
// scroll wheel, and a pointer drag. The window feeds it live input; a recording
// feeds it a script.
type input interface {
	// Down reports whether k is held this frame.
	Down(k key) bool
	// Tapped reports whether k went down this frame.
	Tapped(k key) bool
	// Wheel is this frame's vertical scroll, in notches.
	Wheel() float64
	// Drag reports the pointer position while the primary button is held.
	Drag() (x, y int, ok bool)
}

// frameRange is an inclusive span of frames a scripted key stays held for.
type frameRange struct{ lo, hi int }

// keyScript replays a fixed keybind sequence against frame numbers, so a
// recording performs the same actions every time it runs. It carries its own
// frame counter, which [keyScript.advance] steps once per rendered frame.
type keyScript struct {
	taps  map[key][]int
	holds map[key][]frameRange
	frame int
}

// advance moves the script on to the next frame. Frames are one-based: the
// script's first opportunity to act is the first frame the viewer draws.
func (s *keyScript) advance() { s.frame++ }

func (s *keyScript) Down(k key) bool {
	for _, r := range s.holds[k] {
		if s.frame >= r.lo && s.frame <= r.hi {
			return true
		}
	}
	return false
}

func (s *keyScript) Tapped(k key) bool {
	for _, f := range s.taps[k] {
		if f == s.frame {
			return true
		}
	}
	return false
}

// Wheel and Drag are always idle: a script drives the viewer by keybind alone,
// which keeps a recording reproducible.
func (s *keyScript) Wheel() float64         { return 0 }
func (s *keyScript) Drag() (int, int, bool) { return 0, 0, false }

// buildScript turns a comma-separated keybind list into a schedule: toggles and
// taps fire one after another with a gap between them, while a hold (an orbit
// or zoom direction) stays down for the whole clip. A single toggle also fires
// again halfway through, so the clip shows both states.
func buildScript(keys string, recFrames int) *keyScript {
	const (
		intro = 12 // brief pause so the opening state is visible before acting
		gap   = 18 // frames between successive tap actions
	)
	s := &keyScript{taps: map[key][]int{}, holds: map[key][]frameRange{}}

	var toks []string
	for _, raw := range strings.Split(keys, ",") {
		if tok := strings.ToLower(strings.TrimSpace(raw)); tok != "" {
			toks = append(toks, tok)
		}
	}
	if len(toks) == 0 {
		return s
	}

	lone := len(toks) == 1
	tap := func(k key, at int) { s.taps[k] = append(s.taps[k], at) }
	hold := func(k key) { s.holds[k] = append(s.holds[k], frameRange{intro, recFrames}) }
	toggle := func(k key, at int) {
		tap(k, at)
		if lone {
			tap(k, recFrames/2) // show the off state for a single-toggle demo
		}
	}

	slot := intro
	for _, tok := range toks {
		switch {
		case scriptToggleKeys[tok] != nil:
			toggle(*scriptToggleKeys[tok], slot)
		case scriptTapKeys[tok] != nil:
			tap(*scriptTapKeys[tok], slot)
		default:
			for _, k := range scriptHoldKeys[tok] {
				hold(k)
			}
		}
		slot += gap
	}
	return s
}

var (
	scriptToggleKeys = map[string]*key{
		"space": keyPtr(keySpace),
		"x":     keyPtr(keyX),
		"m":     keyPtr(keyM),
	}
	scriptTapKeys = map[string]*key{
		"r":     keyPtr(keyR),
		"s":     keyPtr(keyS),
		"tab":   keyPtr(keyTab),
		"+":     keyPtr(keyPlus),
		"plus":  keyPtr(keyPlus),
		"-":     keyPtr(keyMinus),
		"minus": keyPtr(keyMinus),
	}
	scriptHoldKeys = map[string][]key{
		"left":       {keyArrowLeft},
		"right":      {keyArrowRight},
		"up":         {keyArrowUp},
		"down":       {keyArrowDown},
		"shift+up":   {keyShift, keyArrowUp},
		"shift+down": {keyShift, keyArrowDown},
	}
)

func keyPtr(k key) *key { return &k }
