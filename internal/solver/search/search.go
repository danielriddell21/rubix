package search

import (
	"container/heap"

	"github.com/danielriddell21/rubix/pkg/cube"
)

type Goal func(cube.Cube) bool

type Heuristic func(cube.Cube) int

const unbounded = 1 << 30

var AllMoves = func() []cube.Move {
	ms := make([]cube.Move, cube.NumMoves)
	for i := range ms {
		ms[i] = cube.Move(i)
	}
	return ms
}()

var DominoMoves = []cube.Move{
	cube.U, cube.U2, cube.Up,
	cube.D, cube.D2, cube.Dp,
	cube.R2, cube.L2, cube.F2, cube.B2,
}

func redundant(prev, cur cube.Move, hasPrev bool) bool {
	if !hasPrev {
		return false
	}
	pf, cf := prev.Face(), cur.Face()
	if pf == cf {
		return true
	}
	// Opposite faces commute; only allow the lower-numbered face first.
	if pf%3 == cf%3 && pf > cf {
		return true
	}
	return false
}

func IDDFS(start cube.Cube, goal Goal, moves []cube.Move, maxDepth int) ([]cube.Move, uint64, bool) {
	return IDAStar(start, goal, moves, func(cube.Cube) int { return 0 }, maxDepth)
}

func IDAStar(start cube.Cube, goal Goal, moves []cube.Move, h Heuristic, maxDepth int) ([]cube.Move, uint64, bool) {
	var nodes uint64
	path := make([]cube.Move, 0, maxDepth)

	var dfs func(c cube.Cube, g, bound int, hasPrev bool, prev cube.Move) (bool, int)
	dfs = func(c cube.Cube, g, bound int, hasPrev bool, prev cube.Move) (bool, int) {
		nodes++
		f := g + h(c)
		if f > bound {
			return false, f
		}
		if goal(c) {
			return true, f
		}
		// Track the smallest f that exceeded the current bound; seed it above
		// any real g+h so the first exceeding child wins the min.
		next := unbounded
		for _, m := range moves {
			if redundant(prev, m, hasPrev) {
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
		}
		return false, next
	}

	bound := h(start)
	for bound <= maxDepth {
		found, t := dfs(start, 0, bound, false, 0)
		if found {
			sol := make([]cube.Move, len(path))
			copy(sol, path)
			return sol, nodes, true
		}
		if t > maxDepth {
			break
		}
		bound = t
	}
	return nil, nodes, false
}

type gNode struct {
	c    cube.Cube
	path []cube.Move
	pri  int
}

type pq []gNode

func (p pq) Len() int           { return len(p) }
func (p pq) Less(i, j int) bool { return p[i].pri < p[j].pri }
func (p pq) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p *pq) Push(x any)        { *p = append(*p, x.(gNode)) }
func (p *pq) Pop() any          { old := *p; n := len(old); x := old[n-1]; *p = old[:n-1]; return x }

func Greedy(start cube.Cube, goal Goal, moves []cube.Move, h Heuristic, nodeLimit uint64) ([]cube.Move, uint64, bool) {
	if goal(start) {
		return nil, 0, true
	}
	open := &pq{{c: start, pri: h(start)}}
	visited := map[cube.Cube]bool{start: true}
	var nodes uint64
	for open.Len() > 0 && nodes < nodeLimit {
		n := heap.Pop(open).(gNode)
		nodes++
		var prev cube.Move
		hasPrev := len(n.path) > 0
		if hasPrev {
			prev = n.path[len(n.path)-1]
		}
		for _, m := range moves {
			if redundant(prev, m, hasPrev) {
				continue
			}
			nc := n.c
			nc.Apply(m)
			if visited[nc] {
				continue
			}
			np := append(append([]cube.Move(nil), n.path...), m)
			if goal(nc) {
				return np, nodes, true
			}
			visited[nc] = true
			heap.Push(open, gNode{c: nc, path: np, pri: h(nc)})
		}
	}
	return nil, nodes, false
}

func Bidirectional(start, target cube.Cube, moves []cube.Move, maxDepth int) ([]cube.Move, uint64, bool) {
	if start == target {
		return nil, 0, true
	}
	fwd := map[cube.Cube]biEntry{start: {}}
	bwd := map[cube.Cube]biEntry{target: {}}
	fFrontier := []cube.Cube{start}
	bFrontier := []cube.Cube{target}
	var nodes uint64

	for depth := 0; depth < maxDepth; depth++ {
		// Expand the smaller frontier for efficiency.
		var sol []cube.Move
		var met bool
		if len(fFrontier) <= len(bFrontier) {
			fFrontier, sol, met = expandFrontier(fFrontier, fwd, bwd, moves, true, &nodes)
		} else {
			bFrontier, sol, met = expandFrontier(bFrontier, bwd, fwd, moves, false, &nodes)
		}
		if met {
			return sol, nodes, true
		}
		if len(fFrontier) == 0 || len(bFrontier) == 0 {
			break
		}
	}
	return nil, nodes, false
}

type biEntry struct {
	path []cube.Move
}

func expandFrontier(frontier []cube.Cube, own, other map[cube.Cube]biEntry, moves []cube.Move, forward bool, nodes *uint64) ([]cube.Cube, []cube.Move, bool) {
	var next []cube.Cube
	for _, c := range frontier {
		e := own[c]
		for _, m := range moves {
			nc := c
			nc.Apply(m)
			if _, ok := own[nc]; ok {
				continue
			}
			*nodes++
			np := append(append([]cube.Move(nil), e.path...), m)
			if oe, ok := other[nc]; ok {
				if forward {
					return next, joinPaths(np, oe.path), true
				}
				return next, joinPaths(oe.path, np), true
			}
			own[nc] = biEntry{path: np}
			next = append(next, nc)
		}
	}
	return next, nil, false
}

func joinPaths(fwdPath, bwdPath []cube.Move) []cube.Move {
	sol := append([]cube.Move(nil), fwdPath...)
	return append(sol, cube.InverseSeq(bwdPath)...)
}
