package solver

import "github.com/danielriddell21/rubix/internal/cube"

// packed.go is the "Binary State" from video 1: the cube packed into two 64-bit
// integers so the greedy search can apply moves and score positions with register-only
// bit operations instead of array shuffling. cube.Cube stays the I/O model; this is
// used only inside the descent solvers' hot loop.
//
// Layout: each location is a 5-bit slot. Edges (12 slots in `edges`): bits 0–3 = edge
// id (0–11), bit 4 = orientation (0/1). Corners (8 slots in `corners`): bits 0–2 =
// corner id (0–7), bits 3–4 = orientation (0–2).
type state struct {
	edges   uint64
	corners uint64
}

// solved is the packed solved cube.
var solved = func() state {
	var e, c uint64
	for i := range 12 {
		e |= uint64(i) << (5 * i)
	}
	for i := range 8 {
		c |= uint64(i) << (5 * i)
	}
	return state{e, c}
}()

// pack converts a cube.Cube to its packed form.
func pack(cu cube.Cube) state {
	var e, c uint64
	for i := range 12 {
		e |= (uint64(cu.EdgePos[i]) | uint64(cu.EdgeOri[i])<<4) << (5 * i)
	}
	for i := range 8 {
		c |= (uint64(cu.CornerPos[i]) | uint64(cu.CornerOri[i])<<3) << (5 * i)
	}
	return state{e, c}
}

func (s state) isSolved() bool { return s == solved }

// A turn moves exactly four edges and four corners; the rest stay. For each move we
// keep a mask of the unchanged slots and the (dest, src, orientation-delta) of the four
// that move, so apply is a mask plus four slot shuffles per ring.
type slotMove struct{ dst, src, delta uint8 }

var (
	edgeKeep    [cube.NumMoves]uint64
	edgeMoves   [cube.NumMoves][4]slotMove
	cornerKeep  [cube.NumMoves]uint64
	cornerMoves [cube.NumMoves][4]slotMove
)

// co3[x] = x mod 3 for x in 0..4 (a corner orientation 0..2 plus a delta 0..2).
var co3 = [5]uint64{0, 1, 2, 0, 1}

func init() {
	for m := range cube.NumMoves {
		mc := cube.Solved().Applied(m)
		var ek uint64
		ec := 0
		for i := range 12 {
			ek |= uint64(0x1F) << (5 * i)
			if int(mc.EdgePos[i]) == i && mc.EdgeOri[i] == 0 {
				continue
			}
			ek &^= uint64(0x1F) << (5 * i)
			edgeMoves[m][ec] = slotMove{uint8(i), mc.EdgePos[i], mc.EdgeOri[i]}
			ec++
		}
		edgeKeep[m] = ek

		var ck uint64
		cc := 0
		for i := range 8 {
			ck |= uint64(0x1F) << (5 * i)
			if int(mc.CornerPos[i]) == i && mc.CornerOri[i] == 0 {
				continue
			}
			ck &^= uint64(0x1F) << (5 * i)
			cornerMoves[m][cc] = slotMove{uint8(i), mc.CornerPos[i], mc.CornerOri[i]}
			cc++
		}
		cornerKeep[m] = ck
	}
}

// apply returns the state after move m.
func (s state) apply(m cube.Move) state {
	ne := s.edges & edgeKeep[m]
	for _, e := range &edgeMoves[m] {
		slot := ((s.edges >> (5 * e.src)) & 0x1F) ^ (uint64(e.delta) << 4)
		ne |= slot << (5 * e.dst)
	}
	nc := s.corners & cornerKeep[m]
	for _, c := range &cornerMoves[m] {
		slot := (s.corners >> (5 * c.src)) & 0x1F
		ori := co3[(slot>>3)+uint64(c.delta)]
		nc |= ((slot & 0x7) | ori<<3) << (5 * c.dst)
	}
	return state{ne, nc}
}

// solvedCount is the number of cubies in their solved location AND orientation — a
// slot equals its location index exactly when both id and orientation are correct.
func (s state) solvedCount() int {
	n := 0
	for i := range 12 {
		if (s.edges>>(5*i))&0x1F == uint64(i) {
			n++
		}
	}
	for i := range 8 {
		if (s.corners>>(5*i))&0x1F == uint64(i) {
			n++
		}
	}
	return n
}

// orientedEdgeCount is the number of correctly-oriented edges (orientation bit clear).
func (s state) orientedEdgeCount() int {
	n := 0
	for i := range 12 {
		if (s.edges>>(5*i+4))&1 == 0 {
			n++
		}
	}
	return n
}

// orientedCornerCount is the number of correctly-oriented corners (orientation 0).
func (s state) orientedCornerCount() int {
	n := 0
	for i := range 8 {
		if (s.corners>>(5*i+3))&3 == 0 {
			n++
		}
	}
	return n
}

// middleInMiddle is the number of middle-layer edges (ids 8–11) currently in the four
// middle-layer slots (8–11) — they need not be in the right slot, just the layer.
func (s state) middleInMiddle() int {
	n := 0
	for i := 8; i < 12; i++ {
		if (s.edges>>(5*i))&0xF >= 8 {
			n++
		}
	}
	return n
}

// dominoScore rewards the three "domino" requirements: oriented edges, the four
// middle edges in the middle layer, and oriented corners (maxes at 24).
func (s state) dominoScore() int {
	return s.orientedEdgeCount() + s.middleInMiddle() + s.orientedCornerCount()
}

// inDomino reports whether the cube is in the domino state: all edges and corners
// oriented and the middle edges confined to the middle layer.
func inDominoState(s state) bool {
	return s.orientedEdgeCount() == 12 && s.orientedCornerCount() == 8 && s.middleInMiddle() == 4
}

// applyAll returns the state after a sequence of moves.
func (s state) applyAll(ms []cube.Move) state {
	for _, m := range ms {
		s = s.apply(m)
	}
	return s
}

// edgesAllOriented reports whether every edge is correctly oriented.
func edgesAllOriented(s state) bool { return s.orientedEdgeCount() == 12 }
