package solver

import "github.com/danielriddell21/rubix/internal/cube"

// idaStarSolver adds a lower-bound prune to the iterative-deepening domino search
// (count the incorrect edges — each move fixes at most four — and cut a branch that
// can't reach the goal in the moves left). Iterative-deepening A*.
type idaStarSolver struct{}

func (idaStarSolver) Name() string     { return "idastar" }
func (idaStarSolver) Describe() string { return "domino two-phase via IDA* with a lower-bound prune" }

func (idaStarSolver) Solve(c cube.Cube) (Result, error) {
	return timed("idastar", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		return solveTwoPhase(c, modeIDAStar, false)
	})
}
