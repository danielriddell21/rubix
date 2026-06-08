// Package solver holds the cube-solving strategies, exposed through a common
// interface and an ordered registry that mirrors the two source videos' narrative:
// greedy → sandwich → oriented → cfop → domino → iddfs → idastar → prune → multi.
package solver

import (
	"fmt"
	"time"

	"github.com/danielriddell21/rubix/pkg/cube"
)

// Result is the outcome of a solve.
type Result struct {
	Moves    []cube.Move   // moves played; when Solved, applying them solves the cube
	Strategy string        // solver name
	Nodes    uint64        // search nodes expanded (0 if not search-based)
	Elapsed  time.Duration // wall-clock solve time
	Solved   bool          // false when the solver legitimately gave up (faithful to the videos)
}

// Solver turns a scrambled cube into a sequence of moves that solves it.
type Solver interface {
	Name() string
	Describe() string
	Solve(cube.Cube) (Result, error)
}

var (
	ordered []Solver
	byName  = map[string]Solver{}
)

func register(s Solver) {
	if _, dup := byName[s.Name()]; dup {
		panic("duplicate solver: " + s.Name())
	}
	ordered = append(ordered, s)
	byName[s.Name()] = s
}

// Registration order is the on-screen order across both videos.
func init() {
	register(greedySolver{})
	register(sandwichSolver{})
	register(orientedSolver{})
	register(cfopSolver{})
	register(dominoSolver{})
	register(iddfsSolver{})
	register(idaStarSolver{})
	register(pruneSolver{})
	register(multiSolver{})
}

// All returns every solver in video order.
func All() []Solver { return ordered }

// Get returns the named solver.
func Get(name string) (Solver, error) {
	s, ok := byName[name]
	if !ok {
		return nil, fmt.Errorf("unknown solver %q", name)
	}
	return s, nil
}

// Names returns every solver name in video order.
func Names() []string {
	out := make([]string, len(ordered))
	for i, s := range ordered {
		out[i] = s.Name()
	}
	return out
}

// timed runs solve and stamps the result's strategy and elapsed time. The solve
// function reports whether it actually solved the cube; a solver returning solved=false
// has legitimately given up (not an error). Any solution it claims is sanity-checked.
func timed(name string, c cube.Cube, solve func(cube.Cube) (moves []cube.Move, solved bool, nodes uint64, err error)) (Result, error) {
	start := time.Now()
	moves, solved, nodes, err := solve(c)
	elapsed := time.Since(start)
	if err != nil {
		return Result{}, err
	}
	if solved {
		// Cancel redundant turns so the reported solution is as short as possible, then
		// confirm the trimmed sequence still solves the cube. Unsolved attempts are left
		// as-is so the incomplete solvers' "getting stuck" playback stays faithful.
		moves = cube.Simplify(moves)
		if !c.Applied(moves...).IsSolved() {
			return Result{}, fmt.Errorf("%s: produced an invalid solution", name)
		}
	}
	return Result{Moves: moves, Strategy: name, Nodes: nodes, Elapsed: elapsed, Solved: solved}, nil
}
