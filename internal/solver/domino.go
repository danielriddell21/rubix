package solver

import "github.com/danielriddell21/rubix/pkg/cube"

// dominoSolver reduces the cube to the "domino" state — all edges and corners oriented
// and the four middle edges in the middle layer — and then solves it using only up/down
// quarter turns and half turns of the other faces, which keep that state. Phase 1 is the
// greedy descent toward the domino state; phase 2 finishes with the backward dictionary,
// as in the video.
type dominoSolver struct{}

func (dominoSolver) Name() string     { return "domino" }
func (dominoSolver) Describe() string { return "reduce to the domino state, then solve it" }

func (dominoSolver) Solve(c cube.Cube) (Result, error) {
	return timed("domino", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		ps := pack(c)
		// Phase 1: reduce to the domino state.
		p1, n1 := descend(ps, state.dominoScore, inDominoState, noTerminal, allMoves, 6, 7, stepLimit)
		mid := ps.applyAll(p1)
		if !inDominoState(mid) {
			return p1, false, n1, nil
		}
		// Phase 2: solve within the domino move set, finishing via the dictionary.
		dict := dominoLookup()
		eval := withLookup(dict, state.solvedCount)
		p2, n2 := descend(mid, eval, state.isSolved, inLookup(dict), dominoMoves, 6, 8, stepLimit)
		sol := append(p1, p2...)
		return sol, c.Applied(sol...).IsSolved(), n1 + n2, nil
	})
}
