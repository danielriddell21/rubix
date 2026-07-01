package solver

import "github.com/danielriddell21/rubix/pkg/cube"

type pruneSolver struct{}

func (pruneSolver) Name() string     { return "prune" }
func (pruneSolver) Describe() string { return "domino two-phase via IDA* with exact prune tables" }

func (pruneSolver) Solve(c cube.Cube) (Result, error) {
	return timed("prune", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		return solveTwoPhase(c, modePrune, false)
	})
}
