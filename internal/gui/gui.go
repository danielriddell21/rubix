//go:build ebiten

// Package gui is an Ebiten visualizer of the live cube state. It renders the shared
// cube.Cube as a real 3D cube you can orbit, animates a solution move by move, and
// can unfold into a flat net and back. Built only with the `ebiten` tag.
package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/danielriddell21/rubix/internal/cube"
)

const (
	winW          = 640
	winH          = 640
	scale         = 42 // world units → pixels
	framesPerMove = 20 // solve animation pacing
	orbitSpeed    = 0.035
	unfoldSpeed   = 0.04
)

var palette = [6]color.RGBA{
	cube.ColU: {245, 245, 245, 255}, // white
	cube.ColR: {200, 30, 30, 255},   // red
	cube.ColF: {30, 170, 60, 255},   // green
	cube.ColD: {235, 210, 40, 255},  // yellow
	cube.ColL: {235, 140, 30, 255},  // orange
	cube.ColB: {40, 90, 210, 255},   // blue
}

var outline = color.RGBA{12, 12, 14, 255}

// whiteSub is a 1px white source used to fill vector triangles with a flat colour.
var whiteSub *ebiten.Image

func init() {
	w := ebiten.NewImage(3, 3)
	w.Fill(color.White)
	whiteSub = w.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}

// Available reports whether the visualizer is compiled in.
func Available() bool { return true }

type vec3 struct{ x, y, z float32 }

func (a vec3) add(b vec3) vec3      { return vec3{a.x + b.x, a.y + b.y, a.z + b.z} }
func (a vec3) sub(b vec3) vec3      { return vec3{a.x - b.x, a.y - b.y, a.z - b.z} }
func (a vec3) scale(s float32) vec3 { return vec3{a.x * s, a.y * s, a.z * s} }
func lerp(a, b vec3, t float32) vec3 {
	return vec3{a.x + (b.x-a.x)*t, a.y + (b.y-a.y)*t, a.z + (b.z-a.z)*t}
}

// quad holds a sticker's 3D corners (in both cube and net layouts), its face and the
// cell within that face, and the cube-space normal used for lighting.
type quad struct {
	cube, net [4]vec3
	normal    vec3
	face      cube.Color
	cell      int
}

// stickers builds the 54 sticker quads once: their positions on the solid cube, the
// same stickers laid out flat in an unfolded cross, and a per-face normal.
func stickers() []quad {
	// face basis in cube space: origin = top-left corner (viewed from outside),
	// u = one column step (right), v = one row step (down). Cube spans [-1.5,1.5].
	type basis struct {
		origin, u, v, n vec3
	}
	b := [6]basis{
		cube.ColU: {vec3{-1.5, 1.5, -1.5}, vec3{1, 0, 0}, vec3{0, 0, 1}, vec3{0, 1, 0}},
		cube.ColR: {vec3{1.5, 1.5, 1.5}, vec3{0, 0, -1}, vec3{0, -1, 0}, vec3{1, 0, 0}},
		cube.ColF: {vec3{-1.5, 1.5, 1.5}, vec3{1, 0, 0}, vec3{0, -1, 0}, vec3{0, 0, 1}},
		cube.ColD: {vec3{-1.5, -1.5, 1.5}, vec3{1, 0, 0}, vec3{0, 0, -1}, vec3{0, -1, 0}},
		cube.ColL: {vec3{-1.5, 1.5, -1.5}, vec3{0, 0, 1}, vec3{0, -1, 0}, vec3{-1, 0, 0}},
		cube.ColB: {vec3{1.5, 1.5, -1.5}, vec3{-1, 0, 0}, vec3{0, -1, 0}, vec3{0, 0, -1}},
	}
	// net layout: top-left of each face in the flat cross (x right, y up), centred.
	netTL := [6]vec3{
		cube.ColU: {-1.5, 4.5, 0},
		cube.ColR: {1.5, 1.5, 0},
		cube.ColF: {-1.5, 1.5, 0},
		cube.ColD: {-1.5, -1.5, 0},
		cube.ColL: {-4.5, 1.5, 0},
		cube.ColB: {4.5, 1.5, 0},
	}
	netCentre := vec3{1.5, 1.5, 0}
	nu, nv := vec3{1, 0, 0}, vec3{0, -1, 0}

	var out []quad
	for face := range 6 {
		f := cube.Color(face)
		bs := b[f]
		for r := range 3 {
			for c := range 3 {
				co := bs.origin.add(bs.u.scale(float32(c))).add(bs.v.scale(float32(r)))
				cq := [4]vec3{co, co.add(bs.u), co.add(bs.u).add(bs.v), co.add(bs.v)}
				no := netTL[f].sub(netCentre).add(nu.scale(float32(c))).add(nv.scale(float32(r)))
				nq := [4]vec3{no, no.add(nu), no.add(nu).add(nv), no.add(nv)}
				out = append(out, quad{cube: cq, net: nq, normal: bs.n, face: f, cell: r*3 + c})
			}
		}
	}
	return out
}

type gameState struct {
	c        cube.Cube
	moves    []cube.Move
	idx      int
	frame    int
	quads    []quad
	yaw      float32
	pitch    float32
	unfold   float32 // 0 = cube, 1 = flat net
	unfoldTo float32
	dragging bool
	lastX    int
	lastY    int
}

func (g *gameState) Update() error {
	// Orbit with arrow keys.
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.yaw -= orbitSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.yaw += orbitSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.pitch -= orbitSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.pitch += orbitSpeed
	}
	// Orbit with mouse drag.
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if g.dragging {
			g.yaw += float32(x-g.lastX) * 0.01
			g.pitch += float32(y-g.lastY) * 0.01
		}
		g.lastX, g.lastY, g.dragging = x, y, true
	} else {
		g.dragging = false
	}
	g.pitch = clamp(g.pitch, -1.45, 1.45)

	// Toggle the unfold-to-net animation with space.
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if g.unfoldTo == 0 {
			g.unfoldTo = 1
		} else {
			g.unfoldTo = 0
		}
	}
	g.unfold += clamp(g.unfoldTo-g.unfold, -unfoldSpeed, unfoldSpeed)

	// Step the solve.
	if g.idx < len(g.moves) {
		g.frame++
		if g.frame >= framesPerMove {
			g.frame = 0
			g.c.Apply(g.moves[g.idx])
			g.idx++
		}
	}
	return nil
}

func (g *gameState) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{22, 22, 26, 255})
	f := g.c.ToFacelets()
	sinY, cosY := fsincos(g.yaw)
	sinX, cosX := fsincos(g.pitch)
	light := normalize(vec3{0.4, 0.7, 1})

	type face2d struct {
		pts   [4][2]float32
		depth float32
		col   color.RGBA
	}
	faces := make([]face2d, 0, len(g.quads))
	for _, q := range g.quads {
		var pts [4][2]float32
		var depth float32
		for i := range 4 {
			p := rotate(lerp(q.cube[i], q.net[i], g.unfold), sinY, cosY, sinX, cosX)
			pts[i] = [2]float32{winW/2 + p.x*scale, winH/2 - p.y*scale}
			depth += p.z
		}
		// Lighting from the (rotated) normal; when flat, light it head-on.
		n := rotate(q.normal, sinY, cosY, sinX, cosX)
		lit := 0.55 + 0.45*max(0, dot(n, light))
		lit = lit + (1-lit)*g.unfold // wash out shading as it flattens
		faces = append(faces, face2d{pts: pts, depth: depth / 4, col: shade(palette[f[int(q.face)*9+q.cell]], lit)})
	}
	sort.Slice(faces, func(i, j int) bool { return faces[i].depth < faces[j].depth })
	for _, fc := range faces {
		fillQuad(screen, fc.pts, fc.col)
		strokeQuad(screen, fc.pts)
	}

	status := "solved"
	if g.idx < len(g.moves) {
		status = fmt.Sprintf("move %d/%d: %s", g.idx+1, len(g.moves), g.moves[g.idx])
	} else if len(g.moves) > 0 {
		status = fmt.Sprintf("done in %d moves", len(g.moves))
	}
	ebitenutil.DebugPrintAt(screen, status, 12, 12)
	ebitenutil.DebugPrintAt(screen, "drag / arrows: orbit    space: unfold", 12, winH-22)
}

func (g *gameState) Layout(int, int) (int, int) { return winW, winH }

func rotate(p vec3, sinY, cosY, sinX, cosX float32) vec3 {
	x := p.x*cosY + p.z*sinY
	z := -p.x*sinY + p.z*cosY
	y := p.y*cosX - z*sinX
	z = p.y*sinX + z*cosX
	return vec3{x, y, z}
}

func fillQuad(screen *ebiten.Image, q [4][2]float32, col color.RGBA) {
	var path vector.Path
	path.MoveTo(q[0][0], q[0][1])
	path.LineTo(q[1][0], q[1][1])
	path.LineTo(q[2][0], q[2][1])
	path.LineTo(q[3][0], q[3][1])
	path.Close()
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	r, gg, b, a := float32(col.R)/255, float32(col.G)/255, float32(col.B)/255, float32(col.A)/255
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = r, gg, b, a
	}
	screen.DrawTriangles(vs, is, whiteSub, &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

func strokeQuad(screen *ebiten.Image, q [4][2]float32) {
	for i := range 4 {
		j := (i + 1) % 4
		vector.StrokeLine(screen, q[i][0], q[i][1], q[j][0], q[j][1], 1.5, outline, true)
	}
}

func shade(c color.RGBA, f float32) color.RGBA {
	f = clamp(f, 0, 1)
	return color.RGBA{uint8(float32(c.R) * f), uint8(float32(c.G) * f), uint8(float32(c.B) * f), c.A}
}

func clamp(v, lo, hi float32) float32 { return max(lo, min(hi, v)) }
func dot(a, b vec3) float32           { return a.x*b.x + a.y*b.y + a.z*b.z }
func normalize(v vec3) vec3 {
	l := float32(math.Sqrt(float64(dot(v, v))))
	if l == 0 {
		return v
	}
	return v.scale(1 / l)
}
func fsincos(a float32) (float32, float32) {
	s, c := math.Sincos(float64(a))
	return float32(s), float32(c)
}

// Play opens the visualizer on the start cube and animates the moves.
func Play(start cube.Cube, moves []cube.Move) error {
	g := &gameState{c: start, moves: moves, quads: stickers(), yaw: 0.6, pitch: 0.5}
	ebiten.SetWindowSize(winW, winH)
	ebiten.SetWindowTitle("rubix")
	return ebiten.RunGame(g)
}
