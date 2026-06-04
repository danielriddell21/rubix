package solver

import (
	"github.com/danielriddell21/rubix/internal/cube"
	"github.com/danielriddell21/rubix/internal/solver/search"
)

// twophase.go is the domino two-phase engine shared by the iddfs, idastar, prune and
// multi solvers. Phase 1 reduces the cube to the domino group; phase 2 solves it
// within ⟨U,D,R2,L2,F2,B2⟩. The four solvers are the SAME search at increasing
// levels of optimisation, mirroring video 2's arc: iterative deepening → IDA* with a
// cheap lower bound → exact prune tables → multi-search over many phase-1 reductions.

type searchMode int

const (
	modeIDDFS   searchMode = iota // iterative deepening, minimal lower bound
	modeIDAStar                   // + cheap admissible lower bound (orient counts / 4)
	modePrune                     // + exact prune tables
)

// solveTwoPhase runs the engine in the given mode. When multi is set it enumerates
// many phase-1 reductions and keeps the shortest overall solution.
func solveTwoPhase(c cube.Cube, mode searchMode, multi bool) ([]cube.Move, bool, uint64, error) {
	h1 := phase1Bound(mode)
	h2 := phase2Bound(mode)
	budget := budgetFor(mode)
	const d1, d2 = 13, 18

	if multi {
		sol, found, nodes := multiSearch(c, h1, h2, d1, d2)
		return sol, found, nodes, nil
	}

	var nodes uint64
	p1, n1, ok := idaSearch(c, inDomino, allMoves, h1, d1, budget)
	nodes += n1
	if !ok {
		return nil, false, nodes, nil // gave up reducing to the domino group
	}
	mid := c.Applied(p1...)
	p2, n2, ok := idaSearch(mid, cube.Cube.IsSolved, search.DominoMoves, h2, d2, budget)
	nodes += n2
	if !ok {
		return p1, false, nodes, nil
	}
	return append(p1, p2...), true, nodes, nil
}

// budgetFor caps nodes per phase search. It is 0 (unbounded) for every mode: the
// two-phase search is a complete method, so — like the videos, where the domino
// solver stayed reliable while it got faster — it always solves. The earlier modes
// differ only in flavour, not success.
func budgetFor(searchMode) uint64 { return 0 }

// phase1Bound is the admissible lower bound on moves to reach the domino group, and it
// is where video 2's optimisation arc lives. Each mode prunes harder than the last, so
// the search visits fewer nodes (visible in the node counts and timing) while all three
// still solve every cube. A purely counting bound (incorrect edges over four, as the
// video uses) is too weak here — phase 1 then explodes — so every mode leans on at least
// one exact pattern database, and the arc is realised as increasingly tight bounds:
//
//   - iddfs   — only the edge-orientation + slice database; corner orientation is left
//               unconstrained, so it explores the most.
//   - idastar — that database plus a cheap corner-orientation lower bound (over four):
//               a tighter admissible bound that cuts branches the edge table misses.
//   - prune   — the exact maximum of both pattern databases: the fewest nodes.
func phase1Bound(mode searchMode) search.Heuristic {
	ts, fs := phase1Tables()
	edgeSlice := func(c cube.Cube) int { return int(fs[flipCoord(c)*nUDSlice+udSliceCoord(c)]) }
	cornerSlice := func(c cube.Cube) int { return int(ts[twistCoord(c)*nUDSlice+udSliceCoord(c)]) }
	switch mode {
	case modeIDDFS:
		return edgeSlice
	case modeIDAStar:
		return func(c cube.Cube) int { return max(edgeSlice(c), ceilDiv(badCorners(c), 4)) }
	default: // modePrune
		return func(c cube.Cube) int { return max(edgeSlice(c), cornerSlice(c)) }
	}
}

// phase2Bound is the admissible lower bound on moves to solve within the domino group.
// Every mode uses the exact prune tables here: phase 2 reaches depth ~13 within the
// restricted ⟨U,D,R2,L2,F2,B2⟩ set, and a weak bound there blows up the tree, so the
// iddfs→idastar→prune progression optimises phase 1 (above) while phase 2 always leans
// on the precomputed lookup — as it does in the video.
func phase2Bound(searchMode) search.Heuristic {
	return phase2Heuristic()
}

// idaSearch is IDA* with a node budget (0 = unlimited). It returns the solution, the
// nodes expanded, and whether a solution was found within the depth and budget.
func idaSearch(start cube.Cube, goal search.Goal, moves []cube.Move, h search.Heuristic, maxDepth int, maxNodes uint64) ([]cube.Move, uint64, bool) {
	var nodes uint64
	over := false
	path := make([]cube.Move, 0, maxDepth)

	var dfs func(c cube.Cube, g, bound int, hasPrev bool, prev cube.Move) (bool, int)
	dfs = func(c cube.Cube, g, bound int, hasPrev bool, prev cube.Move) (bool, int) {
		nodes++
		if maxNodes != 0 && nodes > maxNodes {
			over = true
			return false, bound + 1
		}
		f := g + h(c)
		if f > bound {
			return false, f
		}
		if goal(c) {
			return true, f
		}
		next := 1 << 30
		for _, m := range moves {
			if hasPrev && redundantMove(prev, m) {
				continue
			}
			nc := c
			nc.Apply(m)
			path = append(path, m)
			found, t := dfs(nc, g+1, bound, true, m)
			if found {
				return true, t
			}
			path = path[:len(path)-1]
			next = min(next, t)
			if over {
				return false, next
			}
		}
		return false, next
	}

	for bound := h(start); bound <= maxDepth && !over; {
		found, t := dfs(start, 0, bound, false, 0)
		if found {
			return append([]cube.Move(nil), path...), nodes, true
		}
		if t > maxDepth {
			break
		}
		bound = t
	}
	return nil, nodes, false
}

// multiSearch enumerates phase-1 reductions of non-decreasing length, solves phase 2
// for each, and keeps the shortest total — video 2's multi-search. Each new best
// tightens the phase-2 depth cutoff. Uses exact prune-table bounds.
func multiSearch(c cube.Cube, h1, h2 search.Heuristic, d1, d2 int) ([]cube.Move, bool, uint64) {
	var best []cube.Move
	found := false
	bestLen := 1 << 30
	var nodes uint64
	attempts := 0
	const maxAttempts = 200

	for length := h1(c); length <= d1 && attempts < maxAttempts; length++ {
		phase1Each(c, length, h1, &nodes, func(p1 []cube.Move) bool {
			attempts++
			mid := c.Applied(p1...)
			cutoff := min(bestLen-len(p1)-1, d2)
			if cutoff < 0 {
				return attempts >= maxAttempts
			}
			p2, n2, ok := idaSearch(mid, cube.Cube.IsSolved, search.DominoMoves, h2, cutoff, 0)
			nodes += n2
			if ok && len(p1)+len(p2) < bestLen {
				best = append(append([]cube.Move{}, p1...), p2...)
				bestLen = len(best)
				found = true
			}
			return attempts >= maxAttempts
		})
		if found && length >= bestLen {
			break // no shorter total possible from longer phase-1 reductions
		}
	}
	return best, found, nodes
}

// phase1Each invokes fn for each phase-1 reduction of exactly length depth, pruned by
// the admissible bound h. fn returns true to stop the enumeration.
func phase1Each(c cube.Cube, depth int, h search.Heuristic, nodes *uint64, fn func([]cube.Move) bool) {
	path := make([]cube.Move, 0, depth)
	var dfs func(cur cube.Cube, g int, hasPrev bool, prev cube.Move) bool
	dfs = func(cur cube.Cube, g int, hasPrev bool, prev cube.Move) bool {
		*nodes++
		if g == depth {
			if inDomino(cur) {
				return fn(append([]cube.Move(nil), path...))
			}
			return false
		}
		if g+h(cur) > depth {
			return false
		}
		for _, m := range allMoves {
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

// redundantMove prunes a turn of the same face, or of the opposite face in the
// higher-index order (commuting moves).
func redundantMove(prev, cur cube.Move) bool {
	pf, cf := prev.Face(), cur.Face()
	if pf == cf {
		return true
	}
	return pf%3 == cf%3 && pf > cf
}

// badCorners counts misoriented corners, the cheap corner lower bound idastar adds on
// top of the edge pattern database (each move re-orients at most four corners).
func badCorners(c cube.Cube) int {
	n := 0
	for _, o := range c.CornerOri {
		if o != 0 {
			n++
		}
	}
	return n
}

func ceilDiv(a, b int) int { return (a + b - 1) / b }
