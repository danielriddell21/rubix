package solver

import "github.com/danielriddell21/rubix/pkg/cube"

// orientedSolver first orients all twelve edges (only front/back quarter turns can flip
// an edge, so once they are all oriented those turns are no longer needed). It then
// finishes with the sandwich approach but with front and back turns removed from the
// move set — a smaller set that lets it search and look up two moves deeper. The video
// reaches ~99.8% this way.
type orientedSolver struct{}

func (orientedSolver) Name() string { return "oriented" }
func (orientedSolver) Describe() string {
	return "orient all edges, then sandwich with front/back turns removed"
}

func (orientedSolver) Solve(c cube.Cube) (Result, error) {
	return timed("oriented", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		ps := pack(c)
		// Phase 1: orient every edge.
		p1, n1 := descend(ps, state.orientedEdgeCount, edgesAllOriented, noTerminal, allMoves, 6, 7, stepLimit)
		mid := ps.applyAll(p1)
		if !edgesAllOriented(mid) {
			return p1, false, n1, nil
		}
		// Phase 2: with front/back turns removed, search toward the backward dictionary
		// (the same one sandwich uses), then follow the dictionary home — it may use the
		// front/back turns the forward search no longer can.
		dict := fullLookup()
		reach := inLookup(dict)
		eval := withLookup(dict, state.solvedCount)
		p2, n2 := descend(mid, eval, func(s state) bool { return reach(s) }, reach, orientedMoves, 6, 7, stepLimit)
		hit := mid.applyAll(p2)
		if _, ok := dict[hit]; !ok {
			return append(p1, p2...), false, n1 + n2, nil // never reached the dictionary
		}
		sol := append(append(p1, p2...), dictSolve(hit, dict)...)
		return sol, c.Applied(sol...).IsSolved(), n1 + n2, nil
	})
}
