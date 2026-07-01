package solver

import "github.com/danielriddell21/rubix/pkg/cube"

func twistCoord(c cube.Cube) int {
	t := 0
	for i := range 7 {
		t = t*3 + int(c.CornerOri[i])
	}
	return t
}

func flipCoord(c cube.Cube) int {
	f := 0
	for i := range 11 {
		f = f*2 + int(c.EdgeOri[i])
	}
	return f
}

var binom [13][5]int

func init() {
	for n := range 13 {
		binom[n][0] = 1
		for k := 1; k < 5 && k <= n; k++ {
			binom[n][k] = binom[n-1][k-1] + binom[n-1][k]
		}
	}
}

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

func cornPermCoord(c cube.Cube) int {
	return permCoord(c.CornerPos[:])
}

func edge8PermCoord(c cube.Cube) int {
	var p [8]uint8
	copy(p[:], c.EdgePos[:8])
	return permCoord(p[:])
}

func slicePermCoord(c cube.Cube) int {
	var p [4]uint8
	for i := range 4 {
		p[i] = c.EdgePos[8+i] - 8
	}
	return permCoord(p[:])
}

func edgeSlotOri(c cube.Cube, cubie uint8) (slot, ori int) {
	for j := range 12 {
		if c.EdgePos[j] == cubie {
			return j, int(c.EdgeOri[j])
		}
	}
	return 0, 0
}

func cornerSlotOri(c cube.Cube, cubie uint8) (slot, ori int) {
	for j := range 8 {
		if c.CornerPos[j] == cubie {
			return j, int(c.CornerOri[j])
		}
	}
	return 0, 0
}

func crossCoord(c cube.Cube) int {
	idx := 0
	for cubie := uint8(4); cubie < 8; cubie++ {
		s, o := edgeSlotOri(c, cubie)
		idx = idx*24 + s*2 + o
	}
	return idx
}

const crossCoordSize = 24 * 24 * 24 * 24

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
