package solver

import (
	"fmt"
	"time"

	"github.com/danielriddell21/rubix/pkg/cube"
)

type Result struct {
	Moves    []cube.Move
	Strategy string
	Nodes    uint64
	Elapsed  time.Duration
	Solved   bool
}

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

func All() []Solver { return ordered }

func Get(name string) (Solver, error) {
	s, ok := byName[name]
	if !ok {
		return nil, fmt.Errorf("unknown solver %q", name)
	}
	return s, nil
}

func Names() []string {
	out := make([]string, len(ordered))
	for i, s := range ordered {
		out[i] = s.Name()
	}
	return out
}

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
