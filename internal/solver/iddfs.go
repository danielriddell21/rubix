package solver

import "github.com/danielriddell21/rubix/pkg/cube"

type iddfsSolver struct{}

func (iddfsSolver) Name() string     { return "iddfs" }
func (iddfsSolver) Describe() string { return "domino two-phase via iterative deepening" }

func (iddfsSolver) Solve(c cube.Cube) (Result, error) {
	return timed("iddfs", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		return solveTwoPhase(c, modeIDDFS, false)
	})
}
