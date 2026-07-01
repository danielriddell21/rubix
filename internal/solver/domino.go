package solver

import (
	"slices"

	"github.com/danielriddell21/rubix/pkg/cube"
)

type dominoSolver struct{}

func (dominoSolver) Name() string     { return "domino" }
func (dominoSolver) Describe() string { return "reduce to the domino state, then solve it" }

func (dominoSolver) Solve(c cube.Cube) (Result, error) {
	return timed("domino", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		ps := pack(c)
		// Phase 1: reduce to the domino state.
		p1, n1 := descend(ps, state.dominoScore, inDominoState, noTerminal, allMoves, 7)
		mid := ps.applyAll(p1)
		if !inDominoState(mid) {
			return p1, false, n1, nil
		}
		// Phase 2: solve within the domino move set, finishing via the dictionary.
		dict := dominoLookup()
		eval := withLookup(dict, state.solvedCount)
		p2, n2 := descend(mid, eval, state.isSolved, inLookup(dict), dominoMoves, 8)
		sol := slices.Concat(p1, p2)
		return sol, c.Applied(sol...).IsSolved(), n1 + n2, nil
	})
}
