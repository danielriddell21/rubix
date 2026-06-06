package solver

import (
	"runtime"
	"sync"

	"github.com/danielriddell21/rubix/internal/cube"
)

// descend_fast.go is the greedy best-first search from video 1, running on the packed
// state so it is fast enough to search deep. Each step explores every move path up to a
// depth (the first moves searched in parallel), scoring positions as
// evaluation*scoreScale − depth (moves only a tie-breaker), and plays the first move of
// the highest-scoring path — escalating the depth when it can't improve, giving up if
// even the deepest search can't.

const (
	scoreScale = 1000
	goalBonus  = 1 << 30
)

// stateEval evaluates a packed position; higher is closer to the goal.
type stateEval func(state) int

// terminal reports whether the search should stop descending at a position — used for
// dictionary hits, where the path home is already known so there is no point searching
// deeper. noTerminal never stops.
type terminal func(state) bool

func noTerminal(state) bool { return false }

// inLookup returns a terminal predicate that stops at any position in the dictionary.
func inLookup(dict map[state]int) terminal {
	return func(s state) bool { _, ok := dict[s]; return ok }
}

// descend plays moves until isGoal holds or the search is stuck. moves is the move set
// (all 18 or a restricted subset). It returns the moves played and positions visited.
func descend(start state, eval stateEval, isGoal func(state) bool, stop terminal, moves []cube.Move, minDepth, maxDepth, stepLimit int) ([]cube.Move, uint64) {
	cur := start
	var sol []cube.Move
	var nodes uint64
	for step := 0; step < stepLimit; step++ {
		if isGoal(cur) {
			return sol, nodes
		}
		base := eval(cur) * scoreScale
		var best int
		var bestMove cube.Move
		improved := false
		for d := minDepth; d <= maxDepth; d++ {
			var n uint64
			best, bestMove = bestFirst(cur, eval, isGoal, stop, moves, d, &n)
			nodes += n
			if best > base {
				improved = true
				break
			}
		}
		if !improved {
			return sol, nodes
		}
		cur = cur.apply(bestMove)
		sol = append(sol, bestMove)
	}
	return sol, nodes
}

// bestFirst searches each legal first move in parallel and returns the highest subtree
// score and the first move achieving it.
func bestFirst(start state, eval stateEval, isGoal func(state) bool, stop terminal, moves []cube.Move, depth int, nodes *uint64) (int, cube.Move) {
	type res struct {
		score int
		move  cube.Move
		nodes uint64
	}
	results := make([]res, len(moves))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for i, m := range moves {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, m cube.Move) {
			defer wg.Done()
			defer func() { <-sem }()
			var n uint64
			s := dfsMaxState(start.apply(m), eval, isGoal, stop, moves, depth-1, 1, m, &n)
			results[i] = res{s, m, n}
		}(i, m)
	}
	wg.Wait()

	best, bestMove := -1<<62, cube.Move(0)
	for _, r := range results {
		*nodes += r.nodes
		if r.score > best {
			best, bestMove = r.score, r.move
		}
	}
	return best, bestMove
}

func dfsMaxState(s state, eval stateEval, isGoal func(state) bool, stop terminal, moves []cube.Move, depthRemaining, depth int, prev cube.Move, nodes *uint64) int {
	*nodes++
	if isGoal(s) {
		return goalBonus - depth
	}
	best := eval(s)*scoreScale - depth
	if depthRemaining == 0 || stop(s) {
		return best // a dictionary hit is a known path home — stop here
	}
	for _, m := range moves {
		if redundantMove(prev, m) {
			continue
		}
		if v := dfsMaxState(s.apply(m), eval, isGoal, stop, moves, depthRemaining-1, depth+1, m, nodes); v > best {
			best = v
		}
	}
	return best
}
