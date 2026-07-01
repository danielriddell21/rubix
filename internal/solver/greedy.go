package solver

import "github.com/danielriddell21/rubix/pkg/cube"

const stepLimit = 300

type greedySolver struct{}

func (greedySolver) Name() string     { return "greedy" }
func (greedySolver) Describe() string { return "greedy best-first by solved-cubie count" }

func (greedySolver) Solve(c cube.Cube) (Result, error) {
	return timed("greedy", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		moves, nodes := descend(pack(c), state.solvedCount, state.isSolved, noTerminal, allMoves, 7)
		return moves, c.Applied(moves...).IsSolved(), nodes, nil
	})
}
