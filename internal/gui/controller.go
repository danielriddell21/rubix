package gui

import "github.com/danielriddell21/rubix/internal/cube"

// Controller is how the CLI drives the visualizer without coupling the gui package to
// the solver: it supplies a strategy list, a way to make a fresh scramble, and a way
// to solve a cube with a named strategy. The visualizer owns the interaction (scramble
// on load, "r" for a new scramble, "s" to cycle strategy).
type Controller struct {
	Strategies []string                                       // selectable solver names, in order
	Start      int                                            // index of the strategy to begin with
	Initial    *cube.Cube                                     // optional starting cube (else a scramble)
	Scramble   func() cube.Cube                               // a fresh random scramble
	Solve      func(strategy string, c cube.Cube) []cube.Move // solution moves (nil if it gave up)
}
