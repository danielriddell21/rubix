package cube

import (
	"math/rand/v2"
	"testing"
)

func mustMoves(t *testing.T, s string) []Move {
	t.Helper()
	if s == "" {
		return nil
	}
	ms, err := ParseMoves(s)
	if err != nil {
		t.Fatalf("ParseMoves(%q): %v", s, err)
	}
	return ms
}

func TestSimplifyTable(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"R R'", ""},               // inverse cancels
		{"R' R", ""},               // inverse cancels (other order)
		{"U U", "U2"},              // two quarters merge to a half
		{"U2 U2", ""},              // two halves cancel
		{"U U U", "U'"},            // three quarters = one prime
		{"U U U U", ""},            // four quarters = identity
		{"U U2", "U'"},             // quarter + half = prime
		{"U F F' U", "U2"},         // cancel the inner pair, then merge the U's
		{"R L R", "R2 L"},          // same-axis commuting merge
		{"R L R'", "L"},            // R and R' cancel across the L
		{"R U R' U'", "R U R' U'"}, // already minimal: unchanged
		{"U D U D", "U2 D2"},       // independent merges on one axis
	}
	for _, c := range cases {
		got := FormatMoves(Simplify(mustMoves(t, c.in)))
		if got != c.want {
			t.Errorf("Simplify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestSimplifyPreservesTransform checks the core invariant on random sequences: the
// simplified sequence has the same effect as the original and is never longer.
func TestSimplifyPreservesTransform(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for iter := 0; iter < 2000; iter++ {
		n := rng.IntN(40)
		seq := make([]Move, n)
		for i := range seq {
			seq[i] = Move(rng.IntN(int(NumMoves)))
		}
		simplified := Simplify(seq)
		if len(simplified) > len(seq) {
			t.Fatalf("Simplify lengthened %v -> %v", seq, simplified)
		}
		if Solved().Applied(seq...) != Solved().Applied(simplified...) {
			t.Fatalf("Simplify changed the transform for %v -> %v", seq, simplified)
		}
	}
}

// TestSimplifyIdempotent: simplifying an already-simplified sequence is a no-op.
func TestSimplifyIdempotent(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	for iter := 0; iter < 500; iter++ {
		seq := make([]Move, rng.IntN(30))
		for i := range seq {
			seq[i] = Move(rng.IntN(int(NumMoves)))
		}
		once := Simplify(seq)
		twice := Simplify(once)
		if FormatMoves(once) != FormatMoves(twice) {
			t.Fatalf("Simplify not idempotent: %q -> %q", FormatMoves(once), FormatMoves(twice))
		}
	}
}
