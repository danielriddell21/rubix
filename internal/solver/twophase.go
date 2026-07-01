package solver

import (
	"github.com/danielriddell21/rubix/internal/solver/search"
	"github.com/danielriddell21/rubix/pkg/cube"
)

type searchMode int

const (
	modeIDDFS searchMode = iota
	modeIDAStar
	modePrune
)

func solveTwoPhase(c cube.Cube, mode searchMode, multi bool) ([]cube.Move, bool, uint64, error) {
	h1 := phase1Bound(mode)
	h2 := phase2Bound(mode)
	const d1, d2 = 13, 18

	if multi {
		sol, found, nodes := multiSearch(c, h1, h2, d1, d2)
		return sol, found, nodes, nil
	}

	var nodes uint64
	p1, n1, ok := idaSearch(c, inDomino, allMoves, h1, d1)
	nodes += n1
	if !ok {
		return nil, false, nodes, nil // gave up reducing to the domino group
	}
	mid := c.Applied(p1...)
	p2, n2, ok := idaSearch(mid, cube.Cube.IsSolved, search.DominoMoves, h2, d2)
	nodes += n2
	if !ok {
		return p1, false, nodes, nil
	}
	return append(p1, p2...), true, nodes, nil
}

func phase1Bound(mode searchMode) search.Heuristic {
	if mode == modePrune {
		return phase1Heuristic() // exact maximum of both pattern databases
	}
	// iddfs / idastar bound only the edge-orientation + slice subproblem.
	_, fs := phase1Tables()
	edgeSlice := func(c cube.Cube) int { return int(fs[flipCoord(c)*nUDSlice+udSliceCoord(c)]) }
	if mode == modeIDAStar {
		// add a cheap corner-orientation lower bound on top of the edge table.
		return func(c cube.Cube) int { return max(edgeSlice(c), ceilDiv(badCorners(c), 4)) }
	}
	return edgeSlice // modeIDDFS
}

func phase2Bound(searchMode) search.Heuristic {
	return phase2Heuristic()
}

func idaSearch(start cube.Cube, goal search.Goal, moves []cube.Move, h search.Heuristic, maxDepth int) ([]cube.Move, uint64, bool) {
	s := &idaState{
		moves: moves,
		goal:  goal,
		h:     h,
		path:  make([]cube.Move, 0, maxDepth),
	}
	for bound := h(start); bound <= maxDepth; {
		found, t := s.dfs(start, 0, bound, false, 0)
		if found {
			return append([]cube.Move(nil), s.path...), s.nodes, true
		}
		if t > maxDepth {
			break
		}
		bound = t
	}
	return nil, s.nodes, false
}

type idaState struct {
	moves []cube.Move
	goal  search.Goal
	h     search.Heuristic
	nodes uint64
	path  []cube.Move
}

func (s *idaState) dfs(c cube.Cube, g, bound int, hasPrev bool, prev cube.Move) (bool, int) {
	s.nodes++
	f := g + s.h(c)
	if f > bound {
		return false, f
	}
	if s.goal(c) {
		return true, f
	}
	next := 1 << 30
	for _, m := range s.moves {
		if hasPrev && redundantMove(prev, m) {
			continue
		}
		nc := c
		nc.Apply(m)
		s.path = append(s.path, m)
		found, t := s.dfs(nc, g+1, bound, true, m)
		if found {
			return true, t
		}
		s.path = s.path[:len(s.path)-1]
		next = min(next, t)
	}
	return false, next
}

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
			p2, n2, ok := idaSearch(mid, cube.Cube.IsSolved, search.DominoMoves, h2, cutoff)
			nodes += n2
			if ok && len(p1)+len(p2) < bestLen {
				best = append(make([]cube.Move, 0, len(p1)+len(p2)), p1...)
				best = append(best, p2...)
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

func redundantMove(prev, cur cube.Move) bool {
	pf, cf := prev.Face(), cur.Face()
	if pf == cf {
		return true
	}
	return pf%3 == cf%3 && pf > cf
}

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
