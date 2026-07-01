package solver

import (
	"testing"

	"github.com/danielriddell21/rubix/pkg/cube"
)

var completeSolvers = []string{"cfop", "prune", "multi"}

var allSolverNames = []string{"greedy", "sandwich", "oriented", "cfop", "domino", "iddfs", "idastar", "prune", "multi"}

func TestCompleteSolvers(t *testing.T) {
	n := int64(15)
	if testing.Short() {
		n = 4
	}
	for _, name := range completeSolvers {
		s, err := Get(name)
		if err != nil {
			t.Fatal(err)
		}
		for seed := int64(0); seed < n; seed++ {
			c := cube.ScrambledCube(25, seed)
			res, err := s.Solve(c)
			if err != nil {
				t.Errorf("%s seed %d: %v", name, seed, err)
				continue
			}
			if !res.Solved {
				t.Errorf("%s seed %d: failed to solve a full scramble", name, seed)
				continue
			}
			if !c.Applied(res.Moves...).IsSolved() {
				t.Errorf("%s seed %d: solution does not solve the cube", name, seed)
			}
		}
	}
}

func TestSolverValidity(t *testing.T) {
	for _, name := range allSolverNames {
		s, err := Get(name)
		if err != nil {
			t.Fatal(err)
		}
		solved := 0
		const trials = 6
		for seed := int64(0); seed < trials; seed++ {
			c := cube.ScrambledCube(5, seed)
			res, err := s.Solve(c)
			if err != nil {
				t.Errorf("%s seed %d: %v", name, seed, err)
				continue
			}
			if res.Solved {
				if !c.Applied(res.Moves...).IsSolved() {
					t.Errorf("%s seed %d: claims solved but solution is invalid", name, seed)
					continue
				}
				solved++
			}
		}
		if solved == 0 {
			t.Errorf("%s solved none of the shallow scrambles", name)
		}
	}
}

func TestMultiShorterThanCfop(t *testing.T) {
	if testing.Short() {
		t.Skip("skip in short mode")
	}
	cfop, _ := Get("cfop")
	multi, _ := Get("multi")
	var cfopMoves, multiMoves int
	const n = 8
	for seed := int64(0); seed < n; seed++ {
		c := cube.ScrambledCube(25, seed)
		rc, _ := cfop.Solve(c)
		rm, _ := multi.Solve(c)
		cfopMoves += len(rc.Moves)
		multiMoves += len(rm.Moves)
	}
	if multiMoves >= cfopMoves {
		t.Errorf("expected multi (%d) shorter than cfop (%d) total moves", multiMoves, cfopMoves)
	}
}

func TestSolveSolvedCube(t *testing.T) {
	for _, s := range All() {
		res, err := s.Solve(cube.Solved())
		if err != nil {
			t.Errorf("%s: solving solved cube: %v", s.Name(), err)
			continue
		}
		if !res.Solved || len(res.Moves) != 0 {
			t.Errorf("%s: solved cube should need 0 moves, got solved=%v moves=%d", s.Name(), res.Solved, len(res.Moves))
		}
	}
}

func TestRegistryOrder(t *testing.T) {
	got := Names()
	if len(got) != len(allSolverNames) {
		t.Fatalf("got %d solvers, want %d", len(got), len(allSolverNames))
	}
	for i := range allSolverNames {
		if got[i] != allSolverNames[i] {
			t.Errorf("position %d: got %q want %q", i, got[i], allSolverNames[i])
		}
	}
}
