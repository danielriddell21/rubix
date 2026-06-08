package cube

import "testing"

// FuzzParseFacelets ensures arbitrary input never panics, and that facelet decoding is a
// stable canonical round-trip. FromFacelets reconstructs a cube from its cubie colour pairs
// (ignoring redundant stickers), so it is many-to-one; the invariant that holds is
// idempotency: a cube's canonical facelets re-parse to the same cube and are a fixed point.
func FuzzParseFacelets(f *testing.F) {
	f.Add(Solved().ToFacelets().String())
	f.Add(Solved().Applied(R, U, Rp, Up).ToFacelets().String())
	f.Add("")
	f.Add("not a cube")
	f.Fuzz(func(t *testing.T, s string) {
		fl, err := ParseFacelets(s)
		if err != nil {
			return
		}
		c, err := FromFacelets(fl)
		if err != nil {
			return // syntactically valid but not a real cube state
		}
		canon := c.ToFacelets()
		c2, err := FromFacelets(canon)
		if err != nil {
			t.Fatalf("canonical facelets failed to re-parse: %s: %v", canon.String(), err)
		}
		if c2 != c {
			t.Fatalf("canonical round-trip changed the cube:\n %s", canon.String())
		}
		if c2.ToFacelets() != canon {
			t.Fatalf("facelets not a fixed point:\n %s\n %s", canon.String(), c2.ToFacelets().String())
		}
	})
}

// FuzzSimplify checks the two invariants on arbitrary move sequences: simplification
// preserves the transform and never lengthens the sequence.
func FuzzSimplify(f *testing.F) {
	f.Add([]byte{0, 0, 3, 5})
	f.Add([]byte{0, 8, 7, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		seq := make([]Move, len(data))
		for i, b := range data {
			seq[i] = Move(int(b) % int(NumMoves))
		}
		got := Simplify(seq)
		if len(got) > len(seq) {
			t.Fatalf("Simplify lengthened %v -> %v", seq, got)
		}
		if Solved().Applied(seq...) != Solved().Applied(got...) {
			t.Fatalf("Simplify changed transform for %v -> %v", seq, got)
		}
	})
}
