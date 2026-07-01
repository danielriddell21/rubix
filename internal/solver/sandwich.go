package solver

import "github.com/danielriddell21/rubix/pkg/cube"

type sandwichSolver struct{}

func (sandwichSolver) Name() string { return "sandwich" }
func (sandwichSolver) Describe() string {
	return "greedy search + backward meet-in-the-middle dictionary"
}

func (sandwichSolver) Solve(c cube.Cube) (Result, error) {
	return timed("sandwich", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		dict := fullLookup()
		eval := withLookup(dict, state.solvedCount)
		moves, nodes := descend(pack(c), eval, state.isSolved, inLookup(dict), allMoves, 7)
		return moves, c.Applied(moves...).IsSolved(), nodes, nil
	})
}
