package cube

// Simplify rewrites a move sequence into a shorter one with the same overall effect on
// the cube. It cancels and merges turns of the same face — directly adjacent or separated
// only by turns of the opposite face, since moves on the same axis commute (U/D, R/L,
// F/B). For example "R R'" disappears, "U U" becomes "U2", and "U F F' U" becomes "U2".
//
// It is transform-preserving: applying Simplify(ms) to any cube yields the same result as
// applying ms. It never returns a longer sequence than its input.
func Simplify(ms []Move) []Move {
	out := ms
	for {
		next := simplifyOnce(out)
		if len(next) == len(out) {
			return next
		}
		out = next
	}
}

// simplifyOnce makes a single left-to-right pass, collapsing each maximal run of moves
// that share an axis. Repeating it to a fixpoint (see Simplify) catches runs that only
// become adjacent after an intervening run cancels to nothing.
func simplifyOnce(ms []Move) []Move {
	out := make([]Move, 0, len(ms))
	for i := 0; i < len(ms); {
		axis := ms[i].Face() % 3
		// quarter counts (mod 4) per face, indexed by the face's slot on this axis.
		var quarters [6]int
		j := i
		for j < len(ms) && ms[j].Face()%3 == axis {
			m := ms[j]
			quarters[m.Face()] += int(m%3) + 1 // CW=1, 180=2, CCW=3 quarter turns
			j++
		}
		// Emit at most one move per face on the axis, in face order for stable output.
		for face := 0; face < 6; face++ {
			switch quarters[face] % 4 {
			case 1:
				out = append(out, Move(face*3)) // CW
			case 2:
				out = append(out, Move(face*3+1)) // 180
			case 3:
				out = append(out, Move(face*3+2)) // CCW
			}
		}
		i = j
	}
	return out
}
