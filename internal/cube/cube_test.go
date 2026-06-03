package cube

import "testing"

func TestMoveOrderFour(t *testing.T) {
	// Every quarter turn applied four times is the identity.
	for _, m := range []Move{U, R, F, D, L, B, Up, Rp, Fp, Dp, Lp, Bp} {
		c := Solved()
		for range 4 {
			c.Apply(m)
		}
		if !c.IsSolved() {
			t.Errorf("%s applied 4x is not identity", m)
		}
	}
}

func TestMoveInverse(t *testing.T) {
	for m := Move(0); m < NumMoves; m++ {
		c := Solved()
		c.Apply(m)
		c.Apply(m.Inverse())
		if !c.IsSolved() {
			t.Errorf("%s then %s is not identity", m, m.Inverse())
		}
	}
}

func TestDoubleEqualsTwoQuarters(t *testing.T) {
	pairs := []struct {
		dbl   Move
		quart Move
	}{{U2, U}, {R2, R}, {F2, F}, {D2, D}, {L2, L}, {B2, B}}
	for _, p := range pairs {
		a := Solved()
		a.Apply(p.dbl)
		b := Solved()
		b.Apply(p.quart)
		b.Apply(p.quart)
		if a != b {
			t.Errorf("%s != %s %s", p.dbl, p.quart, p.quart)
		}
	}
}

func TestSexyMoveSixTimes(t *testing.T) {
	// (R U R' U') repeated 6 times returns to solved.
	seq, err := ParseMoves("R U R' U'")
	if err != nil {
		t.Fatal(err)
	}
	c := Solved()
	for range 6 {
		c.ApplySeq(seq)
	}
	if !c.IsSolved() {
		t.Error("(R U R' U')x6 is not identity")
	}
}

func TestApplyThenInverseSeq(t *testing.T) {
	seq := Scramble(25, 42)
	c := Solved()
	c.ApplySeq(seq)
	c.ApplySeq(InverseSeq(seq))
	if !c.IsSolved() {
		t.Error("seq then InverseSeq(seq) is not identity")
	}
}

func TestNotationRoundTrip(t *testing.T) {
	in := "R U2 R' D' F2 L B'"
	ms, err := ParseMoves(in)
	if err != nil {
		t.Fatal(err)
	}
	if got := FormatMoves(ms); got != in {
		t.Errorf("round trip got %q want %q", got, in)
	}
}

func TestParseRejectsUnsupported(t *testing.T) {
	for _, bad := range []string{"x", "M", "Rw", "U3", "Z"} {
		if _, err := ParseMove(bad); err == nil {
			t.Errorf("expected error parsing %q", bad)
		}
	}
}

func TestFaceletRoundTrip(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		c := ScrambledCube(20, seed)
		f := c.ToFacelets()
		got, err := FromFacelets(f)
		if err != nil {
			t.Fatalf("seed %d: FromFacelets: %v", seed, err)
		}
		if got != c {
			t.Fatalf("seed %d: cube round trip mismatch", seed)
		}
	}
}

func TestSolvedFaceletString(t *testing.T) {
	want := "UUUUUUUUU" + "RRRRRRRRR" + "FFFFFFFFF" + "DDDDDDDDD" + "LLLLLLLLL" + "BBBBBBBBB"
	if got := Solved().ToFacelets().String(); got != want {
		t.Errorf("solved facelets = %q want %q", got, want)
	}
}

func TestParseFaceletsRoundTrip(t *testing.T) {
	c := ScrambledCube(15, 7)
	s := c.ToFacelets().String()
	f, err := ParseFacelets(s)
	if err != nil {
		t.Fatal(err)
	}
	got, err := FromFacelets(f)
	if err != nil {
		t.Fatal(err)
	}
	if got != c {
		t.Error("ParseFacelets round trip mismatch")
	}
}

func TestValidateAcceptsScrambles(t *testing.T) {
	for seed := int64(0); seed < 100; seed++ {
		if err := ScrambledCube(30, seed).Validate(); err != nil {
			t.Errorf("seed %d: valid scramble rejected: %v", seed, err)
		}
	}
}

func TestValidateRejectsIllegal(t *testing.T) {
	// Single twisted corner.
	c := Solved()
	c.CornerOri[0] = 1
	if err := c.Validate(); err == nil {
		t.Error("expected twisted-corner rejection")
	}
	// Single flipped edge.
	c = Solved()
	c.EdgeOri[0] = 1
	if err := c.Validate(); err == nil {
		t.Error("expected flipped-edge rejection")
	}
	// Swapped pair of edges (parity violation).
	c = Solved()
	c.EdgePos[0], c.EdgePos[1] = c.EdgePos[1], c.EdgePos[0]
	if err := c.Validate(); err == nil {
		t.Error("expected swapped-pair parity rejection")
	}
}
