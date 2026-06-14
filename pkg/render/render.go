// Package render turns a Rubik's cube into depth-sorted 2D polygons, independent of any
// graphics library. A caller projects the cube with [Project] and draws the returned
// [Quad] values back-to-front with its own filled-polygon renderer.
//
// The package is pure: it imports only [github.com/danielriddell21/rubix/pkg/cube] and the
// standard library, holds no global state, and every exported function is deterministic.
package render

import (
	"image/color"
	"math"
	"sort"

	"github.com/danielriddell21/rubix/pkg/cube"
)

// Tile geometry: each sticker is a raised tile inset from its cell, standing proud of the
// dark plastic body.
const (
	tileInset = 0.12 // gap around each tile (plastic shows through)
	tileRaise = 0.16 // how far each coloured tile stands proud of the body
)

// Vec3 is a point or vector in cube space.
type Vec3 struct{ X, Y, Z float32 }

// Add returns a+b.
func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

// Sub returns a-b.
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

// Scale returns a scaled by s.
func (a Vec3) Scale(s float32) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

// Lerp linearly interpolates from a to b at t in [0,1].
func Lerp(a, b Vec3, t float32) Vec3 {
	return Vec3{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t, a.Z + (b.Z-a.Z)*t}
}

// Dot returns the dot product of a and b.
func Dot(a, b Vec3) float32 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

// Cross returns the cross product of a and b.
func Cross(a, b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}

// Normalize returns v scaled to unit length (or v unchanged if it is the zero vector).
func Normalize(v Vec3) Vec3 {
	l := float32(math.Sqrt(float64(Dot(v, v))))
	if l == 0 {
		return v
	}
	return v.Scale(1 / l)
}

// QuadNormal returns the unit normal of a quad from three of its corners.
func QuadNormal(p [4]Vec3) Vec3 {
	return Normalize(Cross(p[1].Sub(p[0]), p[3].Sub(p[0])))
}

// PKind classifies the parts of a raised tile: [Body], [Top] and [Side].
type PKind int

// The parts of a raised tile, in draw order from the base outward.
const (
	Body PKind = iota // dark plastic cell base
	Top               // coloured sticker top (carries the live facelet colour)
	Side              // plastic bevel wall of a raised tile
)

// Poly is one drawable quad, with matching corners on the solid cube ([Poly.Cube]) and in
// the flat net ([Poly.Net]) so it can morph between them. A [Top] poly carries a face/cell
// index ([Poly.Face], [Poly.Cell]) for its live sticker colour.
type Poly struct {
	Cube, Net [4]Vec3
	Normal    Vec3
	Center    Vec3 // cube-space centre, for turn-layer membership
	Kind      PKind
	Face      cube.Color
	Cell      int
}

type basis struct{ origin, u, v, n Vec3 }

type shape struct {
	pts  [4]Vec3
	kind PKind
}

// cellShapes builds the six quads of one raised tile: a plastic base filling the cell, a
// coloured top inset and raised along the normal, and four plastic bevel walls.
func cellShapes(b basis, c, r int) [6]shape {
	A := b.origin.Add(b.u.Scale(float32(c))).Add(b.v.Scale(float32(r)))
	B := A.Add(b.u)
	C := B.Add(b.v)
	D := A.Add(b.v)
	iA := A.Add(b.u.Scale(tileInset)).Add(b.v.Scale(tileInset))
	iB := B.Sub(b.u.Scale(tileInset)).Add(b.v.Scale(tileInset))
	iC := C.Sub(b.u.Scale(tileInset)).Sub(b.v.Scale(tileInset))
	iD := D.Add(b.u.Scale(tileInset)).Sub(b.v.Scale(tileInset))
	hn := b.n.Scale(tileRaise)
	tA, tB, tC, tD := iA.Add(hn), iB.Add(hn), iC.Add(hn), iD.Add(hn)
	return [6]shape{
		{[4]Vec3{A, B, C, D}, Body},
		{[4]Vec3{tA, tB, tC, tD}, Top},
		{[4]Vec3{iA, iB, tB, tA}, Side},
		{[4]Vec3{iB, iC, tC, tB}, Side},
		{[4]Vec3{iC, iD, tD, tC}, Side},
		{[4]Vec3{iD, iA, tA, tD}, Side},
	}
}

// BuildPolys generates every drawable [Poly] once: each face's nine raised tiles, in both
// the solid-cube layout and the flat unfolded-net layout.
func BuildPolys() []Poly {
	cubeB := [6]basis{
		cube.ColU: {Vec3{-1.5, 1.5, -1.5}, Vec3{1, 0, 0}, Vec3{0, 0, 1}, Vec3{0, 1, 0}},
		cube.ColR: {Vec3{1.5, 1.5, 1.5}, Vec3{0, 0, -1}, Vec3{0, -1, 0}, Vec3{1, 0, 0}},
		cube.ColF: {Vec3{-1.5, 1.5, 1.5}, Vec3{1, 0, 0}, Vec3{0, -1, 0}, Vec3{0, 0, 1}},
		cube.ColD: {Vec3{-1.5, -1.5, 1.5}, Vec3{1, 0, 0}, Vec3{0, 0, -1}, Vec3{0, -1, 0}},
		cube.ColL: {Vec3{-1.5, 1.5, -1.5}, Vec3{0, 0, 1}, Vec3{0, -1, 0}, Vec3{-1, 0, 0}},
		cube.ColB: {Vec3{1.5, 1.5, -1.5}, Vec3{-1, 0, 0}, Vec3{0, -1, 0}, Vec3{0, 0, -1}},
	}
	netTL := [6]Vec3{
		cube.ColU: {-1.5, 4.5, 0},
		cube.ColR: {1.5, 1.5, 0},
		cube.ColF: {-1.5, 1.5, 0},
		cube.ColD: {-1.5, -1.5, 0},
		cube.ColL: {-4.5, 1.5, 0},
		cube.ColB: {4.5, 1.5, 0},
	}
	netCentre := Vec3{1.5, 1.5, 0}
	nu, nv, nn := Vec3{1, 0, 0}, Vec3{0, -1, 0}, Vec3{0, 0, 1}

	var out []Poly
	for face := range 6 {
		f := cube.Color(face)
		cb := cubeB[f]
		nb := basis{origin: netTL[f].Sub(netCentre), u: nu, v: nv, n: nn}
		for r := range 3 {
			for c := range 3 {
				cs := cellShapes(cb, c, r)
				ns := cellShapes(nb, c, r)
				for i := range cs {
					ctr := cs[i].pts[0].Add(cs[i].pts[1]).Add(cs[i].pts[2]).Add(cs[i].pts[3]).Scale(0.25)
					out = append(out, Poly{
						Cube: cs[i].pts, Net: ns[i].pts,
						Normal: QuadNormal(cs[i].pts), Center: ctr,
						Kind: cs[i].kind, Face: f, Cell: r*3 + c,
					})
				}
			}
		}
	}
	return out
}

// FaceAxis maps a face index to its rotation axis (0=x,1=y,2=z) and the sign of its
// outward normal along that axis.
func FaceAxis(face int) (axis, sign int) {
	switch face {
	case 0:
		return 1, 1 // U +y
	case 3:
		return 1, -1 // D -y
	case 1:
		return 0, 1 // R +x
	case 4:
		return 0, -1 // L -x
	case 2:
		return 2, 1 // F +z
	default:
		return 2, -1 // B -z
	}
}

// RotateAxis rotates p by the given sine and cosine about axis 0=x, 1=y or 2=z.
func RotateAxis(p Vec3, axis int, s, c float32) Vec3 {
	switch axis {
	case 0: // x
		return Vec3{p.X, p.Y*c - p.Z*s, p.Y*s + p.Z*c}
	case 1: // y
		return Vec3{p.X*c + p.Z*s, p.Y, -p.X*s + p.Z*c}
	default: // z
		return Vec3{p.X*c - p.Y*s, p.X*s + p.Y*c, p.Z}
	}
}

// RotateCamera applies the camera orbit (yaw then pitch) to p, given the precomputed sines
// and cosines of the yaw and pitch angles.
func RotateCamera(p Vec3, sinY, cosY, sinX, cosX float32) Vec3 {
	x := p.X*cosY + p.Z*sinY
	z := -p.X*sinY + p.Z*cosY
	y := p.Y*cosX - z*sinX
	z = p.Y*sinX + z*cosX
	return Vec3{x, y, z}
}

func easeInOut(t float32) float32 { return t * t * (3 - 2*t) }

// TurnAngle returns the rotation axis (0=x,1=y,2=z) and the signed angle of [cube.Move] m
// animated at frac in [0,1] (eased). A clockwise turn, viewed from outside the face, is a
// negative rotation about the face's outward axis (see [FaceAxis]).
func TurnAngle(m cube.Move, frac float32) (axis int, angle float32) {
	var target float32
	switch m % 3 {
	case 0:
		target = -math.Pi / 2 // CW quarter
	case 1:
		target = math.Pi // half turn
	default:
		target = math.Pi / 2 // CCW quarter
	}
	a, sign := FaceAxis(m.Face())
	return a, float32(sign) * target * easeInOut(frac)
}

// InTurnLayer reports whether a [Poly] centred at center belongs to the layer that
// [cube.Move] m turns.
func InTurnLayer(m cube.Move, center Vec3) bool {
	switch m.Face() {
	case 0:
		return center.Y > 0.5
	case 3:
		return center.Y < -0.5
	case 1:
		return center.X > 0.5
	case 4:
		return center.X < -0.5
	case 2:
		return center.Z > 0.5
	default:
		return center.Z < -0.5
	}
}

// palette maps each sticker colour to its display RGBA (the default cube scheme).
var palette = [6]color.RGBA{
	cube.ColU: {245, 245, 245, 255}, // white
	cube.ColR: {210, 40, 40, 255},   // red
	cube.ColF: {40, 185, 75, 255},   // green
	cube.ColD: {245, 215, 50, 255},  // yellow
	cube.ColL: {245, 150, 35, 255},  // orange
	cube.ColB: {45, 105, 225, 255},  // blue
}

// FaceletColor returns the display colour for sticker colour c (the default cube scheme).
func FaceletColor(c cube.Color) color.RGBA { return palette[c] }

// Camera is the viewer used by [Project]: orbit angles Yaw and Pitch (radians) and Scale
// (pixels per world unit).
type Camera struct{ Yaw, Pitch, Scale float32 }

// Quad is one sticker projected to 2D by [Project]: four corners (relative to the cube's
// centre), its fill [color.RGBA], and a depth for back-to-front (painter's) ordering.
type Quad struct {
	Pts   [4][2]float32
	Color color.RGBA
	Depth float32
}

// Project renders cube c's stickers as 2D quads, ordered back-to-front by [Quad.Depth] so a
// caller can draw them in turn with simple filled polygons. Each [Quad]'s corners are
// relative to the cube's centre and already scaled by [Camera.Scale]; add your own centre
// offset. turn animates the moving layer from frac 0→1 (eased); pass [cube.NoMove] (or
// frac 0) for a static cube.
func Project(c cube.Cube, cam Camera, turn cube.Move, frac float32) []Quad {
	f := c.ToFacelets()
	sinY, cosY := sincos(cam.Yaw)
	sinX, cosX := sincos(cam.Pitch)

	animate := frac > 0 && turn < cube.NumMoves
	var turnAxis int
	var tsin, tcos float32
	if animate {
		a, ang := TurnAngle(turn, clamp01(frac))
		turnAxis = a
		tsin, tcos = sincos(ang)
	}

	quads := make([]Quad, 0, 54)
	for _, p := range BuildPolys() {
		if p.Kind != Top {
			continue // the coloured stickers are what a flat renderer draws
		}
		spin := animate && InTurnLayer(turn, p.Center)
		var pts [4][2]float32
		var depth float32
		for i := range 4 {
			pv := p.Cube[i]
			if spin {
				pv = RotateAxis(pv, turnAxis, tsin, tcos)
			}
			pv = RotateCamera(pv, sinY, cosY, sinX, cosX)
			pts[i] = [2]float32{pv.X * cam.Scale, -pv.Y * cam.Scale}
			depth += pv.Z
		}
		quads = append(quads, Quad{Pts: pts, Color: FaceletColor(f[int(p.Face)*9+p.Cell]), Depth: depth / 4})
	}
	sort.Slice(quads, func(i, j int) bool { return quads[i].Depth < quads[j].Depth })
	return quads
}

func sincos(a float32) (sin, cos float32) {
	s, c := math.Sincos(float64(a))
	return float32(s), float32(c)
}

func clamp01(f float32) float32 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}
