package solver

import "github.com/danielriddell21/rubix/pkg/cube"

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
