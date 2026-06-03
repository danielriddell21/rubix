package cube

import "math/rand/v2"

// Scramble returns n random outer-face moves, avoiding consecutive turns of the
// same face (and of the opposite face right after) so the scramble is non-trivial.
// A fixed seed yields a reproducible scramble.
func Scramble(n int, seed int64) []Move {
	rng := rand.New(rand.NewPCG(uint64(seed), 0x9e3779b97f4a7c15))
	moves := make([]Move, 0, n)
	prevFace, prevPrevFace := -1, -1
	for len(moves) < n {
		m := Move(rng.IntN(int(NumMoves)))
		f := m.Face()
		if f == prevFace {
			continue
		}
		// Avoid e.g. R L R, which is equivalent to a shorter sequence: skip a face
		// move sandwiched by its opposite face.
		if f == prevPrevFace && areOpposite(f, prevFace) {
			continue
		}
		moves = append(moves, m)
		prevPrevFace, prevFace = prevFace, f
	}
	return moves
}

// ScrambledCube returns a solved cube with n random moves applied.
func ScrambledCube(n int, seed int64) Cube {
	c := Solved()
	c.ApplySeq(Scramble(n, seed))
	return c
}

// areOpposite reports whether two faces are on opposite sides (U/D, R/L, F/B).
func areOpposite(a, b int) bool {
	// Faces: 0=U,1=R,2=F,3=D,4=L,5=B. Opposite pairs differ by 3.
	if a > b {
		a, b = b, a
	}
	return b-a == 3
}
