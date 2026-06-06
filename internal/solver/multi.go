package solver

import "github.com/danielriddell21/rubix/pkg/cube"

// multiSolver is the final solver from video 2: the prune-table two-phase engine run
// in multi-search mode, which enumerates many phase-1 reductions of increasing length
// and keeps the shortest overall solution. It trades time for noticeably shorter
// solutions (the video reaches a ~20-move average).
type multiSolver struct{}

func (multiSolver) Name() string { return "multi" }
func (multiSolver) Describe() string {
	return "domino multi-search: try many reductions, keep the shortest"
}

func (multiSolver) Solve(c cube.Cube) (Result, error) {
	return timed("multi", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		moves, solved, nodes, err := solveTwoPhase(c, modePrune, true)
		return moves, solved, nodes, err
	})
}
