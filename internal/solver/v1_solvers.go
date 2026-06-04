package solver

import (
	"fmt"

	"github.com/danielriddell21/rubix/internal/cube"
	"github.com/danielriddell21/rubix/internal/solver/search"
)

// This file holds the three solvers from Sebastian Lague's video (video 1):
// the Greedy Solver, the Sandwich Solver and the Oriented Solver.

// greedySolver does greedy best-first search guided by the pattern heuristic. It is
// fast but non-optimal and gives up on deep scrambles — as the video demonstrates.
type greedySolver struct{}

func (greedySolver) Name() string     { return "greedy" }
func (greedySolver) Describe() string { return "greedy best-first search (fast, non-optimal, shallow)" }

func (greedySolver) Solve(c cube.Cube) (Result, error) {
	return timed("greedy", c, func(c cube.Cube) ([]cube.Move, uint64, error) {
		h := patternHeuristic()
		moves, nodes, ok := search.Greedy(c, cube.Cube.IsSolved, search.AllMoves, h, 1_000_000)
		if !ok {
			return nil, nodes, fmt.Errorf("greedy: stuck (try a strong solver such as domino)")
		}
		return moves, nodes, nil
	})
}

// sandwichSolver does meet-in-the-middle (bidirectional) search, squeezing from the
// scrambled and solved ends until the frontiers meet.
type sandwichSolver struct{}

func (sandwichSolver) Name() string     { return "sandwich" }
func (sandwichSolver) Describe() string { return "meet-in-the-middle bidirectional search" }

func (sandwichSolver) Solve(c cube.Cube) (Result, error) {
	return timed("sandwich", c, func(c cube.Cube) ([]cube.Move, uint64, error) {
		moves, nodes, ok := search.Bidirectional(c, cube.Solved(), search.AllMoves, 12)
		if !ok {
			return nil, nodes, fmt.Errorf("sandwich: no solution within the meet-in-the-middle depth")
		}
		return moves, nodes, nil
	})
}

// orientedSolver reduces the cube to the fully-oriented domino group and then solves
// it, taking the first (shortest) reduction it finds.
type orientedSolver struct{}

func (orientedSolver) Name() string     { return "oriented" }
func (orientedSolver) Describe() string { return "orient-then-permute two-phase reduction" }

func (orientedSolver) Solve(c cube.Cube) (Result, error) {
	return timed("oriented", c, func(c cube.Cube) ([]cube.Move, uint64, error) {
		moves, nodes, err := orientThenSolve(c)
		return moves, nodes, err
	})
}

// orientThenSolve performs a single shortest domino reduction followed by a phase-2
// solve. Shared by orientedSolver and the multi solver.
func orientThenSolve(c cube.Cube) ([]cube.Move, uint64, error) {
	h1 := phase1Heuristic()
	h2 := phase2Heuristic()
	p1, n1, ok := search.IDAStar(c, inDomino, search.AllMoves, h1, 13)
	if !ok {
		return nil, n1, fmt.Errorf("oriented: could not reduce to the domino group")
	}
	mid := c.Applied(p1...)
	p2, n2, ok := search.IDAStar(mid, cube.Cube.IsSolved, search.DominoMoves, h2, 18)
	if !ok {
		return nil, n1 + n2, fmt.Errorf("oriented: could not solve the domino group")
	}
	return append(p1, p2...), n1 + n2, nil
}
