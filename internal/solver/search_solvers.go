package solver

import (
	"fmt"

	"github.com/danielriddell21/rubix/internal/cube"
	"github.com/danielriddell21/rubix/internal/solver/search"
)

// This file holds the incremental search solvers from video 2's optimization arc:
// Iterative Deepening, IDA* and IDA* with prune tables.

// iddfsSolver is uninformed iterative-deepening DFS: complete and low-memory, but
// only practical for shallow scrambles.
type iddfsSolver struct{}

func (iddfsSolver) Name() string     { return "iddfs" }
func (iddfsSolver) Describe() string { return "iterative-deepening DFS (uninformed, shallow)" }

func (iddfsSolver) Solve(c cube.Cube) (Result, error) {
	return timed("iddfs", c, func(c cube.Cube) ([]cube.Move, uint64, error) {
		moves, nodes, ok := search.IDDFS(c, cube.Cube.IsSolved, search.AllMoves, 8)
		if !ok {
			return nil, nodes, fmt.Errorf("iddfs: no solution within the depth limit")
		}
		return moves, nodes, nil
	})
}

// idaStarSolver is IDA* with the light orientation heuristic.
type idaStarSolver struct{}

func (idaStarSolver) Name() string     { return "idastar" }
func (idaStarSolver) Describe() string { return "IDA* with an orientation heuristic" }

func (idaStarSolver) Solve(c cube.Cube) (Result, error) {
	return timed("idastar", c, func(c cube.Cube) ([]cube.Move, uint64, error) {
		moves, nodes, ok := search.IDAStar(c, cube.Cube.IsSolved, search.AllMoves, orientationHeuristic(), 11)
		if !ok {
			return nil, nodes, fmt.Errorf("idastar: no solution within the depth limit")
		}
		return moves, nodes, nil
	})
}

// pruneSolver is IDA* with the stronger pattern (prune-table) heuristic.
type pruneSolver struct{}

func (pruneSolver) Name() string     { return "prune" }
func (pruneSolver) Describe() string { return "IDA* powered by pattern prune tables" }

func (pruneSolver) Solve(c cube.Cube) (Result, error) {
	return timed("prune", c, func(c cube.Cube) ([]cube.Move, uint64, error) {
		moves, nodes, ok := search.IDAStar(c, cube.Cube.IsSolved, search.AllMoves, patternHeuristic(), 12)
		if !ok {
			return nil, nodes, fmt.Errorf("prune: no solution within the depth limit")
		}
		return moves, nodes, nil
	})
}
