package solver

import (
	"testing"

	"github.com/danielriddell21/rubix/internal/cube"
)

// strong solvers must solve arbitrary (deep) scrambles.
var strongSolvers = []string{"oriented", "cfop", "domino", "multi"}

// shallow solvers are demonstration steps; test them on shallow scrambles only.
var shallowSolvers = []string{"greedy", "sandwich", "iddfs", "idastar", "prune"}

func TestStrongSolvers(t *testing.T) {
	for _, name := range strongSolvers {
		s, err := Get(name)
		if err != nil {
			t.Fatal(err)
		}
		for seed := int64(0); seed < 8; seed++ {
			c := cube.ScrambledCube(25, seed)
			res, err := s.Solve(c)
			if err != nil {
				t.Errorf("%s seed %d: %v", name, seed, err)
				continue
			}
			if !c.Applied(res.Moves...).IsSolved() {
				t.Errorf("%s seed %d: solution does not solve the cube", name, seed)
			}
		}
	}
}

func TestShallowSolvers(t *testing.T) {
	for _, name := range shallowSolvers {
		s, err := Get(name)
		if err != nil {
			t.Fatal(err)
		}
		for seed := int64(0); seed < 8; seed++ {
			c := cube.ScrambledCube(6, seed)
			res, err := s.Solve(c)
			if err != nil {
				t.Errorf("%s seed %d: %v", name, seed, err)
				continue
			}
			if !c.Applied(res.Moves...).IsSolved() {
				t.Errorf("%s seed %d: solution does not solve the cube", name, seed)
			}
		}
	}
}

func TestSolveSolvedCube(t *testing.T) {
	for _, s := range All() {
		res, err := s.Solve(cube.Solved())
		if err != nil {
			t.Errorf("%s: solving solved cube: %v", s.Name(), err)
			continue
		}
		if len(res.Moves) != 0 {
			t.Errorf("%s: solved cube should need 0 moves, got %d", s.Name(), len(res.Moves))
		}
	}
}

func TestRegistryOrder(t *testing.T) {
	want := []string{"greedy", "sandwich", "oriented", "cfop", "domino", "iddfs", "idastar", "prune", "multi"}
	got := Names()
	if len(got) != len(want) {
		t.Fatalf("got %d solvers, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d: got %q want %q", i, got[i], want[i])
		}
	}
}
