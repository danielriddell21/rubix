package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/danielriddell21/rubix/pkg/cube"
)

// runJSON runs a command with -format json -output <temp file>, asserts a zero exit code,
// and decodes the written JSON into v.
func runJSON(t *testing.T, v any, args ...string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "out.json")
	full := append(append([]string{}, args...), "-format", "json", "-output", out)
	if code := Run("test", full); code != 0 {
		t.Fatalf("Run(%v) exit = %d, want 0", full, code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("unmarshal %s: %v", data, err)
	}
}

func TestScrambleDeterministic(t *testing.T) {
	var a, b scrambleOutput
	runJSON(t, &a, "scramble", "-seed", "1")
	runJSON(t, &b, "scramble", "-seed", "1")
	if a.Scramble != b.Scramble || a.Facelets != b.Facelets {
		t.Errorf("same seed gave different scrambles:\n%+v\n%+v", a, b)
	}
	if a.Seed != 1 || a.N != 25 {
		t.Errorf("got seed=%d n=%d, want seed=1 n=25", a.Seed, a.N)
	}
}

func TestSolversList(t *testing.T) {
	var list []solverInfo
	runJSON(t, &list, "solvers")
	want := []string{"greedy", "sandwich", "oriented", "cfop", "domino", "iddfs", "idastar", "prune", "multi"}
	if len(list) != len(want) {
		t.Fatalf("got %d solvers, want %d", len(list), len(want))
	}
	for i, w := range want {
		if list[i].Name != w {
			t.Errorf("solver %d: got %q want %q", i, list[i].Name, w)
		}
		if list[i].Describe == "" {
			t.Errorf("solver %q has no description", list[i].Name)
		}
	}
}

func TestSolveSolvesAndSimplifies(t *testing.T) {
	c := cube.ScrambledCube(25, 7)
	facelets := c.ToFacelets().String()

	var out solveOutput
	runJSON(t, &out, "solve", "-input", facelets, "-strategy", "multi")

	if !out.Solved {
		t.Fatalf("multi failed to solve a 25-move scramble")
	}
	moves, err := cube.ParseMoves(out.Moves)
	if err != nil {
		t.Fatalf("parse reported moves %q: %v", out.Moves, err)
	}
	if len(moves) != out.MoveCount || len(out.MoveList) != out.MoveCount {
		t.Errorf("move_count %d disagrees with moves (%d) / move_list (%d)", out.MoveCount, len(moves), len(out.MoveList))
	}
	if !c.Applied(moves...).IsSolved() {
		t.Errorf("reported solution does not solve the cube")
	}
	// Simplify runs in the solver, so the reported sequence must already be minimal.
	if got := cube.Simplify(moves); len(got) != len(moves) {
		t.Errorf("reported solution not simplified: %d moves, simplifies to %d", len(moves), len(got))
	}
}

func TestReplicaSummary(t *testing.T) {
	var out replicaOutput
	runJSON(t, &out, "replica", "-count", "3", "-seed", "1")
	if out.Summary.Total != 3 {
		t.Errorf("summary total = %d, want 3", out.Summary.Total)
	}
	if out.Summary.Solved != 3 { // default strategy is prune (always solves)
		t.Errorf("summary solved = %d, want 3", out.Summary.Solved)
	}
	if len(out.Results) != 3 {
		t.Fatalf("got %d result rows, want 3", len(out.Results))
	}
	for _, r := range out.Results {
		if !r.Solved || r.MoveCount == 0 {
			t.Errorf("row %d: solved=%v moves=%d", r.Index, r.Solved, r.MoveCount)
		}
	}
}

func TestSolveBadInputFails(t *testing.T) {
	if code := Run("test", []string{"solve", "-input", "not-a-cube"}); code == 0 {
		t.Errorf("solve with bad input should fail, got exit 0")
	}
}

func TestInvalidFormatFails(t *testing.T) {
	if code := Run("test", []string{"scramble", "-format", "xml"}); code == 0 {
		t.Errorf("invalid -format should fail, got exit 0")
	}
}
