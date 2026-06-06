// Package search provides the shared solving engine used by the cube solvers:
// iterative-deepening DFS, IDA*, greedy best-first and bidirectional (meet in the
// middle) search over cube states, plus pruning-table construction and caching.
package search

import (
	"container/heap"

	"github.com/danielriddell21/rubix/pkg/cube"
)

// Goal reports whether a cube state satisfies the search target.
type Goal func(cube.Cube) bool

// Heuristic estimates the number of moves remaining from a state. It must never
// overestimate (be admissible) for IDA* to return optimal solutions.
type Heuristic func(cube.Cube) int

// AllMoves is the full set of 18 outer-face turns.
var AllMoves = func() []cube.Move {
	ms := make([]cube.Move, cube.NumMoves)
	for i := range ms {
		ms[i] = cube.Move(i)
	}
	return ms
}()

// DominoMoves is the phase-2 move set: ⟨U, D, R2, L2, F2, B2⟩.
var DominoMoves = []cube.Move{
	cube.U, cube.U2, cube.Up,
	cube.D, cube.D2, cube.Dp,
	cube.R2, cube.L2, cube.F2, cube.B2,
}

// redundant reports whether playing cur right after prev is wasteful: another turn
// of the same face, or a turn of the opposite face in the "wrong" order (to break
// the symmetry of commuting moves like R and L).
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

// IDDFS performs uninformed iterative-deepening DFS up to maxDepth. It returns the
// solution moves, the number of nodes expanded, and whether a solution was found.
func IDDFS(start cube.Cube, goal Goal, moves []cube.Move, maxDepth int) ([]cube.Move, uint64, bool) {
	return IDAStar(start, goal, moves, func(cube.Cube) int { return 0 }, maxDepth)
}

// IDAStar performs iterative-deepening A* search.
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
		next := bound + 1<<30
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

// node for the greedy best-first priority queue.
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

// Greedy performs greedy best-first search: it always expands the open state with
// the lowest heuristic value, ignoring path cost. It is fast but non-optimal and
// gives up after nodeLimit expansions. Returns the solution, nodes expanded, ok.
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

// Bidirectional performs meet-in-the-middle BFS from start (forward) and target
// (backward), expanding both frontiers until they intersect. maxDepth bounds the
// total solution length. Returns the solution moves, states explored, ok.
func Bidirectional(start, target cube.Cube, moves []cube.Move, maxDepth int) ([]cube.Move, uint64, bool) {
	if start == target {
		return nil, 0, true
	}
	type entry struct {
		path []cube.Move
	}
	fwd := map[cube.Cube]entry{start: {}}
	bwd := map[cube.Cube]entry{target: {}}
	fFrontier := []cube.Cube{start}
	bFrontier := []cube.Cube{target}
	var nodes uint64

	join := func(fwdPath, bwdPath []cube.Move) []cube.Move {
		sol := append([]cube.Move(nil), fwdPath...)
		// Backward path was built from target; reverse and invert to walk to target.
		sol = append(sol, cube.InverseSeq(bwdPath)...)
		return sol
	}

	for depth := 0; depth < maxDepth; depth++ {
		// Expand the smaller frontier for efficiency.
		expandFwd := len(fFrontier) <= len(bFrontier)
		if expandFwd {
			var nextF []cube.Cube
			for _, c := range fFrontier {
				e := fwd[c]
				for _, m := range moves {
					nc := c
					nc.Apply(m)
					if _, ok := fwd[nc]; ok {
						continue
					}
					nodes++
					np := append(append([]cube.Move(nil), e.path...), m)
					if be, ok := bwd[nc]; ok {
						return join(np, be.path), nodes, true
					}
					fwd[nc] = entry{path: np}
					nextF = append(nextF, nc)
				}
			}
			fFrontier = nextF
		} else {
			var nextB []cube.Cube
			for _, c := range bFrontier {
				e := bwd[c]
				for _, m := range moves {
					nc := c
					nc.Apply(m)
					if _, ok := bwd[nc]; ok {
						continue
					}
					nodes++
					np := append(append([]cube.Move(nil), e.path...), m)
					if fe, ok := fwd[nc]; ok {
						return join(fe.path, np), nodes, true
					}
					bwd[nc] = entry{path: np}
					nextB = append(nextB, nc)
				}
			}
			bFrontier = nextB
		}
		if len(fFrontier) == 0 || len(bFrontier) == 0 {
			break
		}
	}
	return nil, nodes, false
}
