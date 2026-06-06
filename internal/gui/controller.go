package gui

import "github.com/danielriddell21/rubix/pkg/cube"

// Controller is how the CLI drives the visualizer without coupling the gui package to
// the solver: it supplies a strategy list, a way to make a fresh scramble, and a way
// to solve a cube with a named strategy. The visualizer owns the interaction (scramble
// on load, "r" for a new scramble, "s" to cycle strategy).
type Controller struct {
	Strategies []string         // selectable solver names, in order
	Start      int              // index of the strategy to begin with
	Initial    *cube.Cube       // optional starting cube (else a scramble)
	Scramble   func() cube.Cube // a fresh random scramble
	// Solve returns the moves a strategy played and whether they actually solved the
	// cube. The moves are returned even when solved is false, so the viewer can play
	// the attempt and show it getting stuck (faithful to the videos).
	Solve func(strategy string, c cube.Cube) (moves []cube.Move, solved bool)

	// Cells, when non-empty, drives a grid of independent cubes (replica mode) instead
	// of the single self-driving cube. Each cell carries its own scramble and strategy.
	Cells []Cell
}

// Cell is one cube in a replica grid: a starting scramble, the strategy that solves it,
// and a short label shown on the cell.
type Cell struct {
	Strategy string
	Initial  cube.Cube
	Label    string
}
