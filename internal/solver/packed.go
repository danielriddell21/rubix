package solver

import "github.com/danielriddell21/rubix/pkg/cube"

type state struct {
	edges   uint64
	corners uint64
}

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

type slotMove struct{ dst, src, delta uint8 }

var (
	edgeKeep    [cube.NumMoves]uint64
	edgeMoves   [cube.NumMoves][4]slotMove
	cornerKeep  [cube.NumMoves]uint64
	cornerMoves [cube.NumMoves][4]slotMove
)

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

func (s state) orientedEdgeCount() int {
	n := 0
	for i := range 12 {
		if (s.edges>>(5*i+4))&1 == 0 {
			n++
		}
	}
	return n
}

func (s state) orientedCornerCount() int {
	n := 0
	for i := range 8 {
		if (s.corners>>(5*i+3))&3 == 0 {
			n++
		}
	}
	return n
}

func (s state) middleInMiddle() int {
	n := 0
	for i := 8; i < 12; i++ {
		if (s.edges>>(5*i))&0xF >= 8 {
			n++
		}
	}
	return n
}

func (s state) dominoScore() int {
	return s.orientedEdgeCount() + s.middleInMiddle() + s.orientedCornerCount()
}

func inDominoState(s state) bool {
	return s.orientedEdgeCount() == 12 && s.orientedCornerCount() == 8 && s.middleInMiddle() == 4
}

func (s state) applyAll(ms []cube.Move) state {
	for _, m := range ms {
		s = s.apply(m)
	}
	return s
}

func edgesAllOriented(s state) bool { return s.orientedEdgeCount() == 12 }
