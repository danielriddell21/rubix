package solver

import "github.com/danielriddell21/rubix/pkg/cube"

type idaStarSolver struct{}

func (idaStarSolver) Name() string     { return "idastar" }
func (idaStarSolver) Describe() string { return "domino two-phase via IDA* with a lower-bound prune" }

func (idaStarSolver) Solve(c cube.Cube) (Result, error) {
	return timed("idastar", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		return solveTwoPhase(c, modeIDAStar, false)
	})
}
