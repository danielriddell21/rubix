//go:build ebiten

// Package gui is an Ebiten visualizer of the live cube state. It renders the cube as a
// real 3D object with raised plastic tiles, animates each turn, and is self-driving:
// it scrambles and solves on its own, lets you orbit/zoom, x-ray, unfold to a flat net,
// re-scramble ("r") and switch solver ("s"). Built only with the `ebiten` tag.
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
	winW        = 640
	winH        = 640
	scale       = 40 // world units → pixels (before zoom)
	turnFrames  = 9  // frames per animated layer turn
	netFrames   = 8  // frames per move while unfolded flat
	orbitSpeed  = 0.035
	zoomSpeed   = 0.04
	unfoldSpeed = 0.04
	tileInset   = 0.12 // gap around each tile (plastic shows through)
	tileRaise   = 0.16 // how far each coloured tile stands proud of the body
)

var palette = [6]color.RGBA{
	cube.ColU: {245, 245, 245, 255}, // white
	cube.ColR: {210, 40, 40, 255},   // red
	cube.ColF: {40, 185, 75, 255},   // green
	cube.ColD: {245, 215, 50, 255},  // yellow
	cube.ColL: {245, 150, 35, 255},  // orange
	cube.ColB: {45, 105, 225, 255},  // blue
}

var (
	plastic    = color.RGBA{34, 34, 40, 255}
	outline    = color.RGBA{12, 12, 14, 255}
	background = color.RGBA{30, 30, 36, 255}
)

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

type pkind int

const (
	kBody pkind = iota // dark plastic cell base
	kTop               // coloured sticker top (gets the live facelet colour)
	kSide              // plastic bevel wall of a raised tile
)

// poly is one drawable quad, with matching corners on the solid cube and in the flat
// net so it can morph between them. tops carry a face/cell for their live colour.
type poly struct {
	cube, net [4]vec3
	normal    vec3
	center    vec3 // cube-space centre, for turn-layer membership
	kind      pkind
	face      cube.Color
	cell      int
}

type basis struct{ origin, u, v, n vec3 }

type shape struct {
	pts  [4]vec3
	kind pkind
}

// cellShapes builds the six quads of one raised tile: a plastic base filling the cell,
// a coloured top inset and raised along the normal, and four plastic bevel walls.
func cellShapes(b basis, c, r int) [6]shape {
	A := b.origin.add(b.u.scale(float32(c))).add(b.v.scale(float32(r)))
	B := A.add(b.u)
	C := B.add(b.v)
	D := A.add(b.v)
	iA := A.add(b.u.scale(tileInset)).add(b.v.scale(tileInset))
	iB := B.sub(b.u.scale(tileInset)).add(b.v.scale(tileInset))
	iC := C.sub(b.u.scale(tileInset)).sub(b.v.scale(tileInset))
	iD := D.add(b.u.scale(tileInset)).sub(b.v.scale(tileInset))
	hn := b.n.scale(tileRaise)
	tA, tB, tC, tD := iA.add(hn), iB.add(hn), iC.add(hn), iD.add(hn)
	return [6]shape{
		{[4]vec3{A, B, C, D}, kBody},
		{[4]vec3{tA, tB, tC, tD}, kTop},
		{[4]vec3{iA, iB, tB, tA}, kSide},
		{[4]vec3{iB, iC, tC, tB}, kSide},
		{[4]vec3{iC, iD, tD, tC}, kSide},
		{[4]vec3{iD, iA, tA, tD}, kSide},
	}
}

// buildPolys generates every drawable quad once: each face's nine raised tiles, in
// both the solid-cube layout and the flat unfolded-net layout.
func buildPolys() []poly {
	cubeB := [6]basis{
		cube.ColU: {vec3{-1.5, 1.5, -1.5}, vec3{1, 0, 0}, vec3{0, 0, 1}, vec3{0, 1, 0}},
		cube.ColR: {vec3{1.5, 1.5, 1.5}, vec3{0, 0, -1}, vec3{0, -1, 0}, vec3{1, 0, 0}},
		cube.ColF: {vec3{-1.5, 1.5, 1.5}, vec3{1, 0, 0}, vec3{0, -1, 0}, vec3{0, 0, 1}},
		cube.ColD: {vec3{-1.5, -1.5, 1.5}, vec3{1, 0, 0}, vec3{0, 0, -1}, vec3{0, -1, 0}},
		cube.ColL: {vec3{-1.5, 1.5, -1.5}, vec3{0, 0, 1}, vec3{0, -1, 0}, vec3{-1, 0, 0}},
		cube.ColB: {vec3{1.5, 1.5, -1.5}, vec3{-1, 0, 0}, vec3{0, -1, 0}, vec3{0, 0, -1}},
	}
	netTL := [6]vec3{
		cube.ColU: {-1.5, 4.5, 0},
		cube.ColR: {1.5, 1.5, 0},
		cube.ColF: {-1.5, 1.5, 0},
		cube.ColD: {-1.5, -1.5, 0},
		cube.ColL: {-4.5, 1.5, 0},
		cube.ColB: {4.5, 1.5, 0},
	}
	netCentre := vec3{1.5, 1.5, 0}
	nu, nv, nn := vec3{1, 0, 0}, vec3{0, -1, 0}, vec3{0, 0, 1}

	var out []poly
	for face := range 6 {
		f := cube.Color(face)
		cb := cubeB[f]
		nb := basis{origin: netTL[f].sub(netCentre), u: nu, v: nv, n: nn}
		for r := range 3 {
			for c := range 3 {
				cs := cellShapes(cb, c, r)
				ns := cellShapes(nb, c, r)
				for i := range cs {
					ctr := cs[i].pts[0].add(cs[i].pts[1]).add(cs[i].pts[2]).add(cs[i].pts[3]).scale(0.25)
					out = append(out, poly{
						cube: cs[i].pts, net: ns[i].pts,
						normal: quadNormal(cs[i].pts), center: ctr,
						kind: cs[i].kind, face: f, cell: r*3 + c,
					})
				}
			}
		}
	}
	return out
}

type solveOut struct {
	gen   int
	moves []cube.Move
}

type gameState struct {
	ctrl     Controller
	start    cube.Cube // the current scramble (re-solved when the strategy changes)
	stratIdx int
	solving  bool
	gen      int
	solveCh  chan solveOut

	c     cube.Cube
	moves []cube.Move
	idx   int
	frame int // net-mode move pacing

	turning   bool
	turnFrame int
	turnMove  cube.Move

	polys []poly

	yaw, pitch float32
	zoom       float32
	unfold     float32 // 0 = cube, 1 = flat net
	unfoldTo   float32
	xray       bool
	dragging   bool
	lastX      int
	lastY      int
}

// kickSolve restarts the animation from the current scramble and solves it (with the
// current strategy) on a background goroutine so the window never blocks.
func (g *gameState) kickSolve() {
	g.c = g.start
	g.idx, g.frame, g.turnFrame = 0, 0, 0
	g.turning = false
	g.moves = nil
	g.solving = true
	g.gen++
	gen, strat, start, ch, solve := g.gen, g.ctrl.Strategies[g.stratIdx], g.start, g.solveCh, g.ctrl.Solve
	go func() { ch <- solveOut{gen, solve(strat, start)} }()
}

func (g *gameState) newScramble() {
	g.start = g.ctrl.Scramble()
	g.kickSolve()
}

func (g *gameState) Update() error {
	// Collect a finished solve (ignoring stale ones from a superseded request).
	select {
	case out := <-g.solveCh:
		if out.gen == g.gen {
			g.moves, g.solving = out.moves, false
		}
	default:
	}

	shift := ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)

	// Orbit (only while folded up); the view locks and faces the net once unfolded.
	if g.unfold < 0.5 {
		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
			g.yaw -= orbitSpeed
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
			g.yaw += orbitSpeed
		}
		if !shift {
			if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
				g.pitch -= orbitSpeed
			}
			if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
				g.pitch += orbitSpeed
			}
		}
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
	} else {
		g.dragging = false
		g.yaw += clamp(-g.yaw, -0.08, 0.08)
		g.pitch += clamp(-g.pitch, -0.08, 0.08)
	}

	// Zoom: mouse wheel, or Shift+Up/Down.
	if _, dy := ebiten.Wheel(); dy != 0 {
		g.zoom *= float32(math.Pow(1.12, dy))
	}
	if shift {
		if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
			g.zoom *= 1 + zoomSpeed
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
			g.zoom *= 1 - zoomSpeed
		}
	}
	g.zoom = clamp(g.zoom, 0.5, 3)

	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if g.unfoldTo == 0 {
			g.unfoldTo = 1
		} else {
			g.unfoldTo = 0
		}
	}
	g.unfold += clamp(g.unfoldTo-g.unfold, -unfoldSpeed, unfoldSpeed)

	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		g.xray = !g.xray
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.newScramble()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && len(g.ctrl.Strategies) > 0 {
		g.stratIdx = (g.stratIdx + 1) % len(g.ctrl.Strategies)
		g.kickSolve()
	}

	g.advanceSolve()
	return nil
}

// advanceSolve plays the solution: animated turns while folded, instant snaps while flat.
func (g *gameState) advanceSolve() {
	if g.unfold >= 0.5 {
		g.turning = false
		if g.idx < len(g.moves) {
			if g.frame++; g.frame >= netFrames {
				g.frame = 0
				g.c.Apply(g.moves[g.idx])
				g.idx++
			}
		}
		return
	}
	if g.turning {
		if g.turnFrame++; g.turnFrame >= turnFrames {
			g.turning = false
			g.c.Apply(g.turnMove)
			g.idx++
		}
		return
	}
	if g.idx < len(g.moves) {
		g.turning, g.turnFrame, g.turnMove = true, 0, g.moves[g.idx]
	}
}

func (g *gameState) Draw(screen *ebiten.Image) {
	screen.Fill(background)
	f := g.c.ToFacelets()
	sinY, cosY := fsincos(g.yaw)
	sinX, cosX := fsincos(g.pitch)
	key := normalize(vec3{0.5, 0.8, 0.9})
	fill := normalize(vec3{-0.5, 0.3, 0.4})
	px := scale * g.zoom

	// Current animated turn angle, applied to the moving layer.
	turnAxis, turnAng := g.turnAngle()
	tsin, tcos := fsincos(turnAng)

	type face2d struct {
		pts    [4][2]float32
		depth  float32
		col    color.RGBA
		stroke bool
	}
	faces := make([]face2d, 0, len(g.polys))
	for _, p := range g.polys {
		if g.xray && p.kind != kTop {
			continue // x-ray: drop the plastic body so every side shows through
		}
		spin := g.turning && g.inTurnLayer(p)
		var pts [4][2]float32
		var depth float32
		for i := range 4 {
			v := lerp(p.cube[i], p.net[i], g.unfold)
			if spin {
				v = rotateAxis(v, turnAxis, tsin, tcos)
			}
			v = rotate(v, sinY, cosY, sinX, cosX)
			pts[i] = [2]float32{winW/2 + v.x*px, winH/2 - v.y*px}
			depth += v.z
		}
		n := p.normal
		if spin {
			n = rotateAxis(n, turnAxis, tsin, tcos)
		}
		n = rotate(n, sinY, cosY, sinX, cosX)
		lit := min(1, 0.6+0.32*max(0, dot(n, key))+0.18*max(0, dot(n, fill)))
		base := plastic
		if p.kind == kTop {
			base = palette[f[int(p.face)*9+p.cell]]
		}
		col := shade(base, lit)
		if g.xray {
			col.A = 125
		}
		faces = append(faces, face2d{pts: pts, depth: depth / 4, col: col, stroke: p.kind == kTop && !g.xray})
	}
	sort.Slice(faces, func(i, j int) bool { return faces[i].depth < faces[j].depth })
	for _, fc := range faces {
		fillQuad(screen, fc.pts, fc.col)
		if fc.stroke {
			strokeQuad(screen, fc.pts)
		}
	}

	ebitenutil.DebugPrintAt(screen, g.status(), 12, 12)
	ebitenutil.DebugPrintAt(screen, "drag/arrows: orbit   shift+up/down or wheel: zoom   space: unfold   x: x-ray   r: scramble   s: solver", 12, winH-22)
}

func (g *gameState) status() string {
	strat := ""
	if len(g.ctrl.Strategies) > 0 {
		strat = g.ctrl.Strategies[g.stratIdx]
	}
	switch {
	case g.solving:
		return "solving... (" + strat + ")"
	case len(g.moves) == 0:
		return "[" + strat + "] gave up"
	case g.idx < len(g.moves):
		return fmt.Sprintf("[%s] move %d/%d  %s", strat, g.idx+1, len(g.moves), g.moves[g.idx])
	default:
		return fmt.Sprintf("[%s] solved in %d moves", strat, len(g.moves))
	}
}

// turnAngle returns the axis (0=x,1=y,2=z) and current signed angle of the active turn.
// Clockwise (viewed from outside the face) is a negative rotation about the outward axis.
func (g *gameState) turnAngle() (int, float32) {
	if !g.turning {
		return 1, 0
	}
	var target float32
	switch g.turnMove % 3 {
	case 0:
		target = -math.Pi / 2 // CW quarter
	case 1:
		target = math.Pi // half turn
	default:
		target = math.Pi / 2 // CCW quarter
	}
	axis, sign := faceAxis(g.turnMove.Face())
	t := easeInOut(float32(g.turnFrame) / turnFrames)
	return axis, float32(sign) * target * t
}

func (g *gameState) inTurnLayer(p poly) bool {
	switch g.turnMove.Face() {
	case 0:
		return p.center.y > 0.5
	case 3:
		return p.center.y < -0.5
	case 1:
		return p.center.x > 0.5
	case 4:
		return p.center.x < -0.5
	case 2:
		return p.center.z > 0.5
	default:
		return p.center.z < -0.5
	}
}

func (g *gameState) Layout(int, int) (int, int) { return winW, winH }

// faceAxis maps a face index to its rotation axis (0=x,1=y,2=z) and the sign of its
// outward normal along that axis.
func faceAxis(face int) (axis, sign int) {
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

func rotateAxis(p vec3, axis int, s, c float32) vec3 {
	switch axis {
	case 0: // x
		return vec3{p.x, p.y*c - p.z*s, p.y*s + p.z*c}
	case 1: // y
		return vec3{p.x*c + p.z*s, p.y, -p.x*s + p.z*c}
	default: // z
		return vec3{p.x*c - p.y*s, p.x*s + p.y*c, p.z}
	}
}

func rotate(p vec3, sinY, cosY, sinX, cosX float32) vec3 {
	x := p.x*cosY + p.z*sinY
	z := -p.x*sinY + p.z*cosY
	y := p.y*cosX - z*sinX
	z = p.y*sinX + z*cosX
	return vec3{x, y, z}
}

func easeInOut(t float32) float32 { return t * t * (3 - 2*t) }

func quadNormal(p [4]vec3) vec3 {
	return normalize(cross(p[1].sub(p[0]), p[3].sub(p[0])))
}

func cross(a, b vec3) vec3 {
	return vec3{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x}
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
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = r*a, gg*a, b*a, a
	}
	screen.DrawTriangles(vs, is, whiteSub, &ebiten.DrawTrianglesOptions{AntiAlias: true})
}

func strokeQuad(screen *ebiten.Image, q [4][2]float32) {
	for i := range 4 {
		j := (i + 1) % 4
		vector.StrokeLine(screen, q[i][0], q[i][1], q[j][0], q[j][1], 1.2, outline, true)
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

// Play opens the self-driving visualizer driven by ctrl.
func Play(ctrl Controller) error {
	g := &gameState{
		ctrl: ctrl, polys: buildPolys(),
		yaw: 0.6, pitch: 0.5, zoom: 1,
		stratIdx: ctrl.Start, solveCh: make(chan solveOut, 8),
	}
	if ctrl.Initial != nil {
		g.start = *ctrl.Initial
	} else {
		g.start = ctrl.Scramble()
	}
	g.kickSolve()
	ebiten.SetWindowSize(winW, winH)
	ebiten.SetWindowTitle("rubix")
	return ebiten.RunGame(g)
}
