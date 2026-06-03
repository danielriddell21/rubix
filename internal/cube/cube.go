// Package cube provides the core 3x3x3 Rubik's cube data model: a cubie-level
// representation (piece permutation + orientation), move application, Singmaster
// notation, facelet conversion for I/O, validation and scrambling.
//
// This is the single shared state model used by the headless solver, the Ebiten
// visualizer and the LEGO EV3 robot.
package cube

// Corner slots, in the standard Kociemba order.
const (
	URF = iota
	UFL
	ULB
	UBR
	DFR
	DLF
	DBL
	DRB
)

// Edge slots, in the standard Kociemba order.
const (
	UR = iota
	UF
	UL
	UB
	DR
	DF
	DL
	DB
	FR
	FL
	BL
	BR
)

// Cube is a cubie-level cube state.
//
//   - CornerPos[i] is the corner cubie currently in slot i (a permutation of 0..7).
//   - CornerOri[i] is that corner's twist: 0, 1 or 2.
//   - EdgePos[i] is the edge cubie currently in slot i (a permutation of 0..11).
//   - EdgeOri[i] is that edge's flip: 0 or 1.
//
// Centres are fixed, so the cube frame is fixed and whole-cube rotations are not
// represented.
type Cube struct {
	CornerPos [8]uint8
	CornerOri [8]uint8
	EdgePos   [12]uint8
	EdgeOri   [12]uint8
}

// Solved returns a solved cube.
func Solved() Cube {
	var c Cube
	for i := range c.CornerPos {
		c.CornerPos[i] = uint8(i)
	}
	for i := range c.EdgePos {
		c.EdgePos[i] = uint8(i)
	}
	return c
}

// IsSolved reports whether the cube is solved.
func (c Cube) IsSolved() bool {
	return c == Solved()
}

// Clone returns a copy of the cube. Cube is a value type with no pointers, so a
// plain assignment copies it; Clone exists for call-site clarity.
func (c Cube) Clone() Cube { return c }
