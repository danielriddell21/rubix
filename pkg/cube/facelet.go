package cube

import (
	"fmt"
	"strings"
)

// Color identifies a sticker colour by the face it belongs to on a solved cube:
// 0=U, 1=R, 2=F, 3=D, 4=L, 5=B.
type Color uint8

// The colours, one per cube face on a solved cube.
const (
	ColU Color = iota
	ColR
	ColF
	ColD
	ColL
	ColB
)

// faceLetters maps a Color to its facelet letter, also used as the default colour
// scheme (U=white, R=red, F=green, D=yellow, L=orange, B=blue).
var faceLetters = [6]byte{'U', 'R', 'F', 'D', 'L', 'B'}

// String returns the face letter (U, R, F, D, L or B) for the colour.
func (c Color) String() string { return string(faceLetters[c]) }

// Facelets is the 54-sticker view of a cube, ordered U0..8, R9..17, F18..26,
// D27..35, L36..44, B45..53 (each face row-major).
type Facelets [54]Color

// Facelet slot indices for each corner cubie's three stickers, in Kociemba order.
var cornerFacelet = [8][3]int{
	{8, 9, 20},   // URF: U9, R1, F3
	{6, 18, 38},  // UFL: U7, F1, L3
	{0, 36, 47},  // ULB: U1, L1, B3
	{2, 45, 11},  // UBR: U3, B1, R3
	{29, 26, 15}, // DFR: D3, F9, R7
	{27, 44, 24}, // DLF: D1, L9, F7
	{33, 53, 42}, // DBL: D7, B9, L7
	{35, 17, 51}, // DRB: D9, R9, B7
}

// Facelet slot indices for each edge cubie's two stickers, in Kociemba order.
var edgeFacelet = [12][2]int{
	{5, 10},  // UR: U6, R2
	{7, 19},  // UF: U8, F2
	{3, 37},  // UL: U4, L2
	{1, 46},  // UB: U2, B2
	{32, 16}, // DR: D6, R8
	{28, 25}, // DF: D2, F8
	{30, 43}, // DL: D4, L8
	{34, 52}, // DB: D8, B8
	{23, 12}, // FR: F6, R4
	{21, 41}, // FL: F4, L6
	{50, 39}, // BL: B6, L4
	{48, 14}, // BR: B4, R6
}

// cornerColor[c][n] is the solved-state colour of corner c's n-th sticker.
var cornerColor [8][3]Color

// edgeColor[e][n] is the solved-state colour of edge e's n-th sticker.
var edgeColor [12][2]Color

func init() {
	for c := range cornerColor {
		for n := range cornerColor[c] {
			cornerColor[c][n] = Color(cornerFacelet[c][n] / 9)
		}
	}
	for e := range edgeColor {
		for n := range edgeColor[e] {
			edgeColor[e][n] = Color(edgeFacelet[e][n] / 9)
		}
	}
}

// ToFacelets renders the cube as its 54-sticker colour view.
func (c Cube) ToFacelets() Facelets {
	var f Facelets
	// Centres.
	for face := range 6 {
		f[face*9+4] = Color(face)
	}
	for i := range 8 {
		j := c.CornerPos[i]
		ori := c.CornerOri[i]
		for n := range 3 {
			f[cornerFacelet[i][(n+int(ori))%3]] = cornerColor[j][n]
		}
	}
	for i := range 12 {
		j := c.EdgePos[i]
		ori := c.EdgeOri[i]
		for n := range 2 {
			f[edgeFacelet[i][(n+int(ori))%2]] = edgeColor[j][n]
		}
	}
	return f
}

// String renders the facelets as a 54-character URFDLB string.
func (f Facelets) String() string {
	var b strings.Builder
	b.Grow(54)
	for _, col := range f {
		b.WriteByte(faceLetters[col])
	}
	return b.String()
}

// FromFacelets builds a cube from a 54-sticker colour view, returning an error if
// the stickers do not describe a legal cube.
func FromFacelets(f Facelets) (Cube, error) {
	var c Cube
	for i := range 8 {
		pos, ori, ok := matchCorner(f, i)
		if !ok {
			return c, fmt.Errorf("corner slot %d has no matching cubie", i)
		}
		c.CornerPos[i], c.CornerOri[i] = pos, ori
	}
	for i := range 12 {
		pos, ori, ok := matchEdge(f, i)
		if !ok {
			return c, fmt.Errorf("edge slot %d has no matching cubie", i)
		}
		c.EdgePos[i], c.EdgeOri[i] = pos, ori
	}
	if err := c.Validate(); err != nil {
		return c, err
	}
	return c, nil
}

// matchCorner resolves corner slot i to its cubie index and orientation from the
// facelets (orientation is the index of the U/D sticker), reporting false if no
// cubie matches.
func matchCorner(f Facelets, i int) (pos, ori uint8, ok bool) {
	var o int
	for o = range 3 {
		col := f[cornerFacelet[i][o]]
		if col == ColU || col == ColD {
			break
		}
	}
	col1 := f[cornerFacelet[i][(o+1)%3]]
	col2 := f[cornerFacelet[i][(o+2)%3]]
	for j := range 8 {
		if col1 == cornerColor[j][1] && col2 == cornerColor[j][2] {
			return uint8(j), uint8(o), true
		}
	}
	return 0, 0, false
}

// matchEdge resolves edge slot i to its cubie index and orientation from the
// facelets, reporting false if no cubie matches.
func matchEdge(f Facelets, i int) (pos, ori uint8, ok bool) {
	a := f[edgeFacelet[i][0]]
	b := f[edgeFacelet[i][1]]
	for j := range 12 {
		switch {
		case a == edgeColor[j][0] && b == edgeColor[j][1]:
			return uint8(j), 0, true
		case a == edgeColor[j][1] && b == edgeColor[j][0]:
			return uint8(j), 1, true
		}
	}
	return 0, 0, false
}

// ParseFacelets parses a 54-character string in the URFDLB scheme. Each facelet
// may be given as a face letter (U, R, F, D, L, B) or the equivalent default
// colour (W, R, G, Y, O, B); both are accepted case-insensitively. Whitespace is
// ignored.
func ParseFacelets(s string) (Facelets, error) {
	var f Facelets
	clean := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			return -1
		}
		return r
	}, s)
	if len(clean) != 54 {
		return f, fmt.Errorf("expected 54 facelets, got %d", len(clean))
	}
	for i := range 54 {
		col, ok := parseColor(clean[i])
		if !ok {
			return f, fmt.Errorf("invalid facelet %q at index %d", string(clean[i]), i)
		}
		f[i] = col
	}
	return f, nil
}

func parseColor(b byte) (Color, bool) {
	switch b {
	case 'U', 'u', 'W', 'w':
		return ColU, true
	case 'R', 'r':
		return ColR, true
	case 'F', 'f', 'G', 'g':
		return ColF, true
	case 'D', 'd', 'Y', 'y':
		return ColD, true
	case 'L', 'l', 'O', 'o':
		return ColL, true
	case 'B', 'b':
		return ColB, true
	default:
		return 0, false
	}
}
