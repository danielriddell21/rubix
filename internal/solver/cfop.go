package solver

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/danielriddell21/rubix/internal/solver/search"
	"github.com/danielriddell21/rubix/pkg/cube"
)

type cfopSolver struct{}

func (cfopSolver) Name() string     { return "cfop" }
func (cfopSolver) Describe() string { return "CFOP: Cross, F2L, OLL, PLL (table-guided staged search)" }

func (cfopSolver) Solve(c cube.Cube) (Result, error) {
	return timed("cfop", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		moves, nodes, err := cfopSolve(c)
		if err != nil {
			// A stage that exceeds its depth is a give-up, not a hard error.
			return moves, false, nodes, nil //nolint:nilerr // depth give-up is a non-result, not an error
		}
		return moves, c.Applied(moves...).IsSolved(), nodes, nil
	})
}

func cfopStage(cur *cube.Cube, sol *[]cube.Move, nodes *uint64, goal search.Goal, h search.Heuristic, maxDepth int, name string) error {
	start := time.Now()
	moves, n, ok := search.IDAStar(*cur, goal, search.AllMoves, h, maxDepth)
	*nodes += n
	if cfopDebug {
		fmt.Fprintf(os.Stderr, "  cfop stage %-7s ok=%v len=%d nodes=%d elapsed=%s\n", name, ok, len(moves), n, time.Since(start))
	}
	if !ok {
		return fmt.Errorf("cfop: %s stage failed within depth %d", name, maxDepth)
	}
	cur.ApplySeq(moves)
	*sol = append(*sol, moves...)
	return nil
}

var cfopDebug = os.Getenv("CFOPDEBUG") != ""

func cfopSolve(c cube.Cube) ([]cube.Move, uint64, error) {
	cross := crossTable()
	crossDist := func(c cube.Cube) int { return int(cross[crossCoord(c)]) }
	pairDist := make([]func(cube.Cube) int, 4)
	pairCoords := make([]func(cube.Cube) int, 4)
	for k := range 4 {
		t := pairTable(k)
		co := pairCoord(k)
		pairCoords[k] = co
		pairDist[k] = func(c cube.Cube) int { return int(t[co(c)]) }
	}
	co := cornerOriTable()
	eo := edgeOriTable()
	cp := cornerPermTable()

	cur := c
	var sol []cube.Move
	var nodes uint64

	// Cross.
	if err := cfopStage(&cur, &sol, &nodes, crossSolved, crossDist, 8, "cross"); err != nil {
		return nil, nodes, err
	}
	// F2L pairs: an additive (greedy) heuristic over the cross plus every pair solved
	// so far and this one. It overestimates, so it is not optimal, but it prunes hard
	// and keeps each pair search fast while penalising disturbing earlier work.
	for k := range 4 {
		dists := []func(cube.Cube) int{crossDist}
		for j := 0; j <= k; j++ {
			dists = append(dists, pairDist[j])
		}
		h := sumDist(dists...)
		if err := cfopStage(&cur, &sol, &nodes, f2lGoal(k), h, 22, fmt.Sprintf("f2l-%d", k+1)); err != nil {
			return nil, nodes, err
		}
	}
	// OLL: orient the last layer, keeping the first two layers.
	ollDists := []func(cube.Cube) int{
		crossDist, pairDist[0], pairDist[1], pairDist[2], pairDist[3],
		func(c cube.Cube) int { return int(co[twistCoord(c)]) },
		func(c cube.Cube) int { return int(eo[flipCoord(c)]) },
	}
	if err := cfopStage(&cur, &sol, &nodes, ollGoal, sumDist(ollDists...), 22, "oll"); err != nil {
		return nil, nodes, err
	}
	// PLL: permute the last layer to finish.
	pllDists := slices.Concat(ollDists, []func(cube.Cube) int{
		func(c cube.Cube) int { return int(cp[cornPermCoord(c)]) },
	})
	if err := cfopStage(&cur, &sol, &nodes, cube.Cube.IsSolved, sumDist(pllDists...), 22, "pll"); err != nil {
		return nil, nodes, err
	}
	return sol, nodes, nil
}

func sumDist(fns ...func(cube.Cube) int) search.Heuristic {
	return func(c cube.Cube) int {
		s := 0
		for _, f := range fns {
			s += f(c)
		}
		return s
	}
}

var crossSlots = []int{4, 5, 6, 7}

func crossSolved(c cube.Cube) bool {
	for _, i := range crossSlots {
		if c.EdgePos[i] != uint8(i) || c.EdgeOri[i] != 0 {
			return false
		}
	}
	return true
}

func f2lGoal(k int) search.Goal {
	return func(c cube.Cube) bool {
		if !crossSolved(c) {
			return false
		}
		for j := 0; j <= k; j++ {
			if c.CornerPos[4+j] != uint8(4+j) || c.CornerOri[4+j] != 0 {
				return false
			}
			if c.EdgePos[8+j] != uint8(8+j) || c.EdgeOri[8+j] != 0 {
				return false
			}
		}
		return true
	}
}

func ollGoal(c cube.Cube) bool {
	if !f2lGoal(3)(c) {
		return false
	}
	for _, o := range c.CornerOri {
		if o != 0 {
			return false
		}
	}
	for _, o := range c.EdgeOri {
		if o != 0 {
			return false
		}
	}
	return true
}
