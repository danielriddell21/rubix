package solver

import (
	"github.com/danielriddell21/rubix/internal/cube"
	"github.com/danielriddell21/rubix/internal/solver/search"
)

// orientationHeuristic is a light admissible heuristic from the corner- and
// edge-orientation pattern tables. Used by the uninformed-leaning idastar solver.
func orientationHeuristic() search.Heuristic {
	co := cornerOriTable()
	eo := edgeOriTable()
	return func(c cube.Cube) int {
		return int(max(co[twistCoord(c)], eo[flipCoord(c)]))
	}
}

// patternHeuristic adds the corner-permutation pattern table on top of the
// orientation tables, giving the stronger bound the prune solver relies on.
func patternHeuristic() search.Heuristic {
	co := cornerOriTable()
	eo := edgeOriTable()
	cp := cornerPermTable()
	return func(c cube.Cube) int {
		return int(max(co[twistCoord(c)], eo[flipCoord(c)], cp[cornPermCoord(c)]))
	}
}
