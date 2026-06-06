package solver

import "github.com/danielriddell21/rubix/pkg/cube"

// This file defines the coordinate projections of a cube used both as pruning-table
// indices and as the goal tests for the two-phase (domino) solver. Each coordinate
// is a well-defined quotient under the relevant move set, so pruning tables built by
// breadth-first search are valid admissible heuristics.

// twistCoord encodes the orientation of the first seven corners in base 3 (the
// eighth is determined). Range 0..2186; solved = 0.
func twistCoord(c cube.Cube) int {
	t := 0
	for i := range 7 {
		t = t*3 + int(c.CornerOri[i])
	}
	return t
}

// flipCoord encodes the orientation of the first eleven edges in base 2 (the
// twelfth is determined). Range 0..2047; solved = 0.
func flipCoord(c cube.Cube) int {
	f := 0
	for i := range 11 {
		f = f*2 + int(c.EdgeOri[i])
	}
	return f
}

// binom is Pascal's triangle for the combinatorial number system used by udSlice.
var binom [13][5]int

func init() {
	for n := range 13 {
		binom[n][0] = 1
		for k := 1; k < 5 && k <= n; k++ {
			binom[n][k] = binom[n-1][k-1] + binom[n-1][k]
		}
	}
}

// udSliceCoord encodes which four slots hold the UD-slice edges (cubies 8..11),
// ignoring their order. Range 0..494.
func udSliceCoord(c cube.Cube) int {
	idx, k := 0, 0
	for j := range 12 {
		if c.EdgePos[j] >= 8 {
			k++
			idx += binom[j][k]
		}
	}
	return idx
}

// permCoord returns the Lehmer-code index of a permutation. Identity = 0.
func permCoord(p []uint8) int {
	idx := 0
	n := len(p)
	for i := range n {
		idx *= n - i
		for j := i + 1; j < n; j++ {
			if p[j] < p[i] {
				idx++
			}
		}
	}
	return idx
}

// cornPermCoord indexes the corner permutation. Range 0..40319; solved = 0.
func cornPermCoord(c cube.Cube) int {
	return permCoord(c.CornerPos[:])
}

// edge8PermCoord indexes the permutation of the eight U/D-layer edges among slots
// 0..7. Only meaningful inside the domino group. Range 0..40319; solved = 0.
func edge8PermCoord(c cube.Cube) int {
	var p [8]uint8
	copy(p[:], c.EdgePos[:8])
	return permCoord(p[:])
}

// slicePermCoord indexes the permutation of the four UD-slice edges among slots
// 8..11. Only meaningful inside the domino group. Range 0..23; solved = 0.
func slicePermCoord(c cube.Cube) int {
	var p [4]uint8
	for i := range 4 {
		p[i] = c.EdgePos[8+i] - 8
	}
	return permCoord(p[:])
}

// edgeSlotOri returns the slot holding the given edge cubie and its orientation.
func edgeSlotOri(c cube.Cube, cubie uint8) (slot, ori int) {
	for j := range 12 {
		if c.EdgePos[j] == cubie {
			return j, int(c.EdgeOri[j])
		}
	}
	return 0, 0
}

// cornerSlotOri returns the slot holding the given corner cubie and its orientation.
func cornerSlotOri(c cube.Cube, cubie uint8) (slot, ori int) {
	for j := range 8 {
		if c.CornerPos[j] == cubie {
			return j, int(c.CornerOri[j])
		}
	}
	return 0, 0
}

// crossCoord encodes the four D-layer cross edges (cubies 4..7): each as slot*2+ori
// (0..23), packed in base 24. Range 0..24^4-1; injective on reachable states.
func crossCoord(c cube.Cube) int {
	idx := 0
	for cubie := uint8(4); cubie < 8; cubie++ {
		s, o := edgeSlotOri(c, cubie)
		idx = idx*24 + s*2 + o
	}
	return idx
}

const crossCoordSize = 24 * 24 * 24 * 24

// pairCoord encodes F2L pair k: its corner (cubie 4+k) as slot*3+ori (0..23) and its
// edge (cubie 8+k) as slot*2+ori (0..23), packed as cornerVal*24+edgeVal. Range 0..575.
func pairCoord(k int) func(cube.Cube) int {
	corner := uint8(4 + k)
	edge := uint8(8 + k)
	return func(c cube.Cube) int {
		cs, co := cornerSlotOri(c, corner)
		es, eo := edgeSlotOri(c, edge)
		return (cs*3+co)*24 + es*2 + eo
	}
}

const pairCoordSize = 24 * 24

// inDomino reports whether the cube lies in the domino group ⟨U,D,R2,L2,F2,B2⟩:
// all pieces oriented and the UD-slice edges confined to the middle slots.
func inDomino(c cube.Cube) bool {
	for _, o := range c.CornerOri {
		if o != 0 {
			return false
		}
	}
	for _, o := range c.EdgeOri {
		if o != 0 {
			return false
		}
	}
	for i := 8; i < 12; i++ {
		if c.EdgePos[i] < 8 {
			return false
		}
	}
	return true
}
