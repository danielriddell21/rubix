package solver

import (
	"fmt"

	"github.com/danielriddell21/rubix/internal/cube"
	"github.com/danielriddell21/rubix/internal/solver/search"
)

// dominoSolver implements Kociemba-style two-phase solving (video 2: "Domino
// Reduction"). Phase 1 reduces the cube to the domino group ⟨U,D,R2,L2,F2,B2⟩;
// phase 2 solves it within that group. Both phases are IDA* searches bounded by
// precomputed pruning tables. It tries successively longer phase-1 reductions and
// keeps the shortest overall solution.
type dominoSolver struct{}

func (dominoSolver) Name() string { return "domino" }
func (dominoSolver) Describe() string {
	return "domino reduction two-phase (Kociemba-style), near-optimal and fast"
}

func (dominoSolver) Solve(c cube.Cube) (Result, error) {
	return timed("domino", c, func(c cube.Cube) ([]cube.Move, uint64, error) {
		moves, nodes, err := twoPhase(c, 5)
		return moves, nodes, err
	})
}

// twoPhase solves c via domino reduction, exploring up to maxAttempts distinct
// phase-1 reductions (of non-decreasing length) and returning the shortest total.
func twoPhase(c cube.Cube, maxAttempts int) ([]cube.Move, uint64, error) {
	h1 := phase1Heuristic()
	h2 := phase2Heuristic()

	var best []cube.Move
	var nodes uint64
	bestLen := 1 << 30

	attempts := 0
	for d1 := h1(c); d1 <= 20 && attempts < maxAttempts; d1++ {
		// Enumerate phase-1 reductions of exactly length d1.
		phase1Each(c, d1, h1, &nodes, func(p1 []cube.Move) bool {
			attempts++
			mid := c.Applied(p1...)
			limit := bestLen - len(p1) - 1
			if limit < 0 {
				limit = 18
			}
			p2, n2, ok := search.IDAStar(mid, cube.Cube.IsSolved, search.DominoMoves, h2, min(limit, 18))
			nodes += n2
			if ok && len(p1)+len(p2) < bestLen {
				best = append(append([]cube.Move(nil), p1...), p2...)
				bestLen = len(best)
			}
			return attempts >= maxAttempts
		})
		if best != nil {
			break
		}
	}
	if best == nil {
		return nil, nodes, fmt.Errorf("domino: no solution found")
	}
	return best, nodes, nil
}

// phase1Each invokes fn for each phase-1 reduction of exactly length depth. fn
// returns true to stop the enumeration early.
func phase1Each(c cube.Cube, depth int, h search.Heuristic, nodes *uint64, fn func([]cube.Move) bool) {
	path := make([]cube.Move, 0, depth)
	var dfs func(cur cube.Cube, g int, hasPrev bool, prev cube.Move) bool
	dfs = func(cur cube.Cube, g int, hasPrev bool, prev cube.Move) bool {
		*nodes++
		if g == depth {
			if inDomino(cur) {
				cp := append([]cube.Move(nil), path...)
				return fn(cp)
			}
			return false
		}
		if g+h(cur) > depth {
			return false
		}
		for _, m := range search.AllMoves {
			if hasPrev && redundantMove(prev, m) {
				continue
			}
			nc := cur
			nc.Apply(m)
			path = append(path, m)
			stop := dfs(nc, g+1, true, m)
			path = path[:len(path)-1]
			if stop {
				return true
			}
		}
		return false
	}
	dfs(c, 0, false, 0)
}

// redundantMove mirrors the move-pruning used by the search engine.
func redundantMove(prev, cur cube.Move) bool {
	pf, cf := prev.Face(), cur.Face()
	if pf == cf {
		return true
	}
	if pf%3 == cf%3 && pf > cf {
		return true
	}
	return false
}
