package solver

import "github.com/danielriddell21/rubix/internal/cube"

// pruneSolver replaces the search's estimate with exact prune tables (edge orientation,
// corner orientation, middle-slice and their combinations), giving the domino solver an
// exact lower bound — the big speedup from video 2, fast and 100% reliable.
type pruneSolver struct{}

func (pruneSolver) Name() string     { return "prune" }
func (pruneSolver) Describe() string { return "domino two-phase via IDA* with exact prune tables" }

func (pruneSolver) Solve(c cube.Cube) (Result, error) {
	return timed("prune", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		return solveTwoPhase(c, modePrune, false)
	})
}
