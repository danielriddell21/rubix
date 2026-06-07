package cube

// Move is one of the 18 outer-face quarter/half turns. Moves are grouped by face
// (face = m/3) and power (m%3: 0 = 90° CW, 1 = 180°, 2 = 90° CCW / prime).
type Move uint8

// The 18 face turns, three per face: 90° CW, 180°, and 90° CCW (prime). NumMoves is the
// count of real moves.
const (
	U Move = iota
	U2
	Up // U'
	R
	R2
	Rp
	F
	F2
	Fp
	D
	D2
	Dp
	L
	L2
	Lp
	B
	B2
	Bp
	NumMoves
)

// Face returns the face index (0=U,1=R,2=F,3=D,4=L,5=B) of the move.
func (m Move) Face() int { return int(m) / 3 }

// Inverse returns the move that undoes m.
func (m Move) Inverse() Move {
	switch m % 3 {
	case 0: // CW -> CCW
		return m + 2
	case 2: // CCW -> CW
		return m - 2
	default: // 180 is its own inverse
		return m
	}
}

// baseMove holds the six clockwise quarter-turn definitions (U,R,F,D,L,B) as the
// cubie permutation/orientation each move produces when applied to a solved cube.
// These are the canonical Kociemba move cubes.
type rawMove struct {
	cp [8]uint8
	co [8]uint8
	ep [12]uint8
	eo [12]uint8
}

var baseMoves = [6]rawMove{
	// U
	{
		cp: [8]uint8{UBR, URF, UFL, ULB, DFR, DLF, DBL, DRB},
		co: [8]uint8{0, 0, 0, 0, 0, 0, 0, 0},
		ep: [12]uint8{UB, UR, UF, UL, DR, DF, DL, DB, FR, FL, BL, BR},
		eo: [12]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// R
	{
		cp: [8]uint8{DFR, UFL, ULB, URF, DRB, DLF, DBL, UBR},
		co: [8]uint8{2, 0, 0, 1, 1, 0, 0, 2},
		ep: [12]uint8{FR, UF, UL, UB, BR, DF, DL, DB, DR, FL, BL, UR},
		eo: [12]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// F
	{
		cp: [8]uint8{UFL, DLF, ULB, UBR, URF, DFR, DBL, DRB},
		co: [8]uint8{1, 2, 0, 0, 2, 1, 0, 0},
		ep: [12]uint8{UR, FL, UL, UB, DR, FR, DL, DB, UF, DF, BL, BR},
		eo: [12]uint8{0, 1, 0, 0, 0, 1, 0, 0, 1, 1, 0, 0},
	},
	// D
	{
		cp: [8]uint8{URF, UFL, ULB, UBR, DLF, DBL, DRB, DFR},
		co: [8]uint8{0, 0, 0, 0, 0, 0, 0, 0},
		ep: [12]uint8{UR, UF, UL, UB, DF, DL, DB, DR, FR, FL, BL, BR},
		eo: [12]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// L
	{
		cp: [8]uint8{URF, ULB, DBL, UBR, DFR, UFL, DLF, DRB},
		co: [8]uint8{0, 1, 2, 0, 0, 2, 1, 0},
		ep: [12]uint8{UR, UF, BL, UB, DR, DF, FL, DB, FR, UL, DL, BR},
		eo: [12]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// B
	{
		cp: [8]uint8{URF, UFL, UBR, DRB, DFR, DLF, ULB, DBL},
		co: [8]uint8{0, 0, 1, 2, 0, 0, 2, 1},
		ep: [12]uint8{UR, UF, UL, BR, DR, DF, DL, BL, FR, FL, UB, DB},
		eo: [12]uint8{0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 1, 1},
	},
}

// moveCubes holds the precomputed Cube for each of the 18 moves.
var moveCubes [NumMoves]Cube

func init() {
	for face := 0; face < 6; face++ {
		var q Cube
		rm := baseMoves[face]
		q.CornerPos = rm.cp
		q.CornerOri = rm.co
		q.EdgePos = rm.ep
		q.EdgeOri = rm.eo

		q2 := compose(q, q)
		q3 := compose(q2, q)
		moveCubes[face*3+0] = q
		moveCubes[face*3+1] = q2
		moveCubes[face*3+2] = q3
	}
}

// compose returns the state of applying move b to a cube already in state a.
func compose(a, b Cube) Cube {
	var r Cube
	for i := range r.CornerPos {
		p := b.CornerPos[i]
		r.CornerPos[i] = a.CornerPos[p]
		r.CornerOri[i] = (a.CornerOri[p] + b.CornerOri[i]) % 3
	}
	for i := range r.EdgePos {
		p := b.EdgePos[i]
		r.EdgePos[i] = a.EdgePos[p]
		r.EdgeOri[i] = (a.EdgeOri[p] + b.EdgeOri[i]) % 2
	}
	return r
}

// Apply applies a single move to the cube.
func (c *Cube) Apply(m Move) { *c = compose(*c, moveCubes[m]) }

// ApplySeq applies a sequence of moves in order.
func (c *Cube) ApplySeq(ms []Move) {
	for _, m := range ms {
		c.Apply(m)
	}
}

// Applied returns a copy of c with the moves applied, leaving c unchanged.
func (c Cube) Applied(ms ...Move) Cube {
	c.ApplySeq(ms)
	return c
}

// InverseSeq returns the moves that undo ms (reversed and each inverted).
func InverseSeq(ms []Move) []Move {
	out := make([]Move, len(ms))
	for i, m := range ms {
		out[len(ms)-1-i] = m.Inverse()
	}
	return out
}
