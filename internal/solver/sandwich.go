package solver

import "github.com/danielriddell21/rubix/internal/cube"

// sandwichSolver is the greedy solver plus a backward "meet in the middle" dictionary:
// a breadth-first search outward from solved records every nearby position's distance
// home, and the evaluation uses it so the descent knows when it is on a winning path.
// The video lifts the success rate from ~10% to ~72% this way.
type sandwichSolver struct{}

func (sandwichSolver) Name() string { return "sandwich" }
func (sandwichSolver) Describe() string {
	return "greedy search + backward meet-in-the-middle dictionary"
}

func (sandwichSolver) Solve(c cube.Cube) (Result, error) {
	return timed("sandwich", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		dict := fullLookup()
		eval := withLookup(dict, state.solvedCount)
		moves, nodes := descend(pack(c), eval, state.isSolved, inLookup(dict), allMoves, 6, 7, stepLimit)
		return moves, c.Applied(moves...).IsSolved(), nodes, nil
	})
}
