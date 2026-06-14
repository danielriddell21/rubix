//go:build ebiten

// Package gui is an Ebiten visualizer of the live cube state. It renders the cube as a
// real 3D object with raised plastic tiles, animates each turn, and is self-driving:
// it scrambles and solves on its own, lets you orbit/zoom, x-ray, unfold to a flat net,
// re-scramble ("r"), switch solver ("s") and toggle the move list ("m"). In replica mode
// it shows a grid of independent cubes ("tab" focuses one). Built only with `ebiten`.
package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/danielriddell21/rubix/pkg/cube"
	"github.com/danielriddell21/rubix/pkg/render"
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

	cellSize = 320  // target pixels per grid cell
	maxWin   = 1280 // largest window edge in replica grid mode
	maxCells = 16   // cap on cubes drawn in the grid (the CLI table handles more)
)

var (
	plastic    = color.RGBA{34, 34, 40, 255}
	outline    = color.RGBA{12, 12, 14, 255}
	background = color.RGBA{30, 30, 36, 255}
	focusLine  = color.RGBA{200, 200, 80, 255}
	cellLine   = color.RGBA{70, 70, 80, 255}
	highlight  = color.RGBA{70, 70, 30, 255}
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

type solveOut struct {
	gen    int
	moves  []cube.Move
	solved bool
}

// cubeView is one animated cube: its scramble, the solution being played and the
// per-move animation state. The camera and geometry are shared and live on gameState.
type cubeView struct {
	ctrl     Controller
	start    cube.Cube // the current scramble (re-solved when the strategy changes)
	stratIdx int
	label    string
	solving  bool
	gen      int
	solveCh  chan solveOut

	c          cube.Cube
	moves      []cube.Move
	idx        int
	frame      int // net-mode move pacing
	lastSolved bool

	turning   bool
	turnFrame int
	turnMove  cube.Move
}

// kickSolve restarts the animation from the current scramble and solves it (with the
// current strategy) on a background goroutine so the window never blocks.
func (v *cubeView) kickSolve() {
	v.c = v.start
	v.idx, v.frame, v.turnFrame = 0, 0, 0
	v.turning = false
	v.moves = nil
	v.solving = true
	v.gen++
	gen, strategy, start, ch, solve := v.gen, v.ctrl.Strategies[v.stratIdx], v.start, v.solveCh, v.ctrl.Solve
	go func() {
		mv, ok := solve(strategy, start)
		ch <- solveOut{gen, mv, ok}
	}()
}

func (v *cubeView) rescramble(scramble func() cube.Cube) {
	v.start = scramble()
	v.kickSolve()
}

// drainSolve picks up a finished solve, ignoring stale results from a superseded kick.
func (v *cubeView) drainSolve() {
	select {
	case out := <-v.solveCh:
		if out.gen == v.gen {
			v.moves, v.solving, v.lastSolved = out.moves, false, out.solved
		}
	default:
	}
}

// advanceSolve plays the solution: animated turns while folded, instant snaps while flat.
func (v *cubeView) advanceSolve(unfold float32) {
	if unfold >= 0.5 {
		v.turning = false
		if v.idx < len(v.moves) {
			if v.frame++; v.frame >= netFrames {
				v.frame = 0
				v.c.Apply(v.moves[v.idx])
				v.idx++
			}
		}
		return
	}
	if v.turning {
		if v.turnFrame++; v.turnFrame >= turnFrames {
			v.turning = false
			v.c.Apply(v.turnMove)
			v.idx++
		}
		return
	}
	if v.idx < len(v.moves) {
		v.turning, v.turnFrame, v.turnMove = true, 0, v.moves[v.idx]
	}
}

func (v *cubeView) strategy() string {
	if len(v.ctrl.Strategies) > 0 {
		return v.ctrl.Strategies[v.stratIdx]
	}
	return ""
}

func (v *cubeView) status() string {
	strategy := v.strategy()
	switch {
	case v.solving:
		return "solving... (" + strategy + ")"
	case len(v.moves) == 0:
		if v.lastSolved {
			return "[" + strategy + "] already solved"
		}
		return "[" + strategy + "] no solution found"
	case v.idx < len(v.moves):
		return fmt.Sprintf("[%s] move %d/%d  %s", strategy, v.idx+1, len(v.moves), v.moves[v.idx])
	case v.lastSolved:
		return fmt.Sprintf("[%s] solved in %d moves", strategy, len(v.moves))
	default:
		return fmt.Sprintf("[%s] stuck after %d moves", strategy, len(v.moves))
	}
}

// shortStatus is the compact per-cell readout for the grid.
func (v *cubeView) shortStatus() string {
	switch {
	case v.solving:
		return "solving"
	case len(v.moves) == 0:
		if v.lastSolved {
			return "solved"
		}
		return "no sol"
	case v.idx < len(v.moves):
		return fmt.Sprintf("%d/%d", v.idx, len(v.moves))
	case v.lastSolved:
		return fmt.Sprintf("done %d", len(v.moves))
	default:
		return fmt.Sprintf("stuck %d", len(v.moves))
	}
}

// viewMode is the visualizer layout: one self-driving cube, a compare grid (one scramble
// across every solver) or a replica grid (several independent scrambles on one solver).
type viewMode int

const (
	modeSingle viewMode = iota
	modeCompare
	modeReplica

	replicaCount = 4 // cubes shown in replica mode (a tidy 2×2)
)

// gameState owns the shared camera, geometry and the set of cube views.
type gameState struct {
	ctrl       Controller
	mode       viewMode
	views      []*cubeView
	cols, rows int
	w, h       int

	polys []render.Poly

	yaw, pitch float32
	zoom       float32
	unfold     float32 // 0 = cube, 1 = flat net
	unfoldTo   float32
	xray       bool
	dragging   bool
	lastX      int
	lastY      int

	focus     int
	showMoves bool

	// Recording (set when ctrl.Record != ""): captures frames in Draw and, once enough
	// are gathered, saves the GIF and ends the run from Update. script, when set, injects
	// keybinds on a frame timeline so each demo records the same actions every run.
	rec       *recorder
	recPath   string
	recFrames int
	recSaved  bool
	frame     int
	script    *keyScript
}

func (g *gameState) grid() bool { return len(g.views) > 1 }

// defaultStrategy is the solver the single and replica views run.
func (g *gameState) defaultStrategy() string {
	if g.ctrl.Start >= 0 && g.ctrl.Start < len(g.ctrl.Strategies) {
		return g.ctrl.Strategies[g.ctrl.Start]
	}
	if len(g.ctrl.Strategies) > 0 {
		return g.ctrl.Strategies[0]
	}
	return ""
}

// buildViews constructs the cube views for a layout, always with fresh scrambles. Compare
// shares one scramble across every solver; replica gives each cube its own scramble solved
// by the default strategy; single is one self-driving cube.
func (g *gameState) buildViews(m viewMode) []*cubeView {
	switch m {
	case modeCompare:
		scramble := g.ctrl.Scramble()
		var vs []*cubeView
		for i, name := range g.ctrl.Strategies {
			if i >= maxCells {
				break
			}
			vs = append(vs, newGridView(g.ctrl, Cell{Strategy: name, Initial: scramble, Label: fmt.Sprintf("#%d %s", i, name)}))
		}
		return vs
	case modeReplica:
		name := g.defaultStrategy()
		vs := make([]*cubeView, 0, replicaCount)
		for i := 0; i < replicaCount; i++ {
			vs = append(vs, newGridView(g.ctrl, Cell{Strategy: name, Initial: g.ctrl.Scramble(), Label: fmt.Sprintf("#%d %s", i, name)}))
		}
		return vs
	default:
		return []*cubeView{newGridView(g.ctrl, Cell{Strategy: g.defaultStrategy(), Initial: g.ctrl.Scramble()})}
	}
}

// setMode switches the layout, rebuilds the views and starts their solves. The window is
// resized to fit the new grid (kept fixed while recording so the GIF frame size is stable).
func (g *gameState) setMode(m viewMode) {
	g.mode = m
	g.views = g.buildViews(m)
	g.cols, g.rows = gridDims(len(g.views))
	g.focus = 0
	if g.rec == nil {
		g.w, g.h = windowSize(g.cols, g.rows)
		ebiten.SetWindowSize(g.w, g.h)
	}
	for _, v := range g.views {
		v.kickSolve()
	}
}

// frameRange is an inclusive interval of frame indices during which a key is held.
type frameRange struct{ lo, hi int }

// keyScript injects keybinds on a fixed timeline so a recording performs the same actions
// every run: taps fire a single just-pressed at the given frame(s); holds keep a key down
// across a range of frames.
type keyScript struct {
	taps  map[ebiten.Key][]int
	holds map[ebiten.Key][]frameRange
}

func (s *keyScript) justPressed(k ebiten.Key, frame int) bool {
	if s == nil {
		return false
	}
	for _, f := range s.taps[k] {
		if f == frame {
			return true
		}
	}
	return false
}

func (s *keyScript) held(k ebiten.Key, frame int) bool {
	if s == nil {
		return false
	}
	for _, r := range s.holds[k] {
		if frame >= r.lo && frame <= r.hi {
			return true
		}
	}
	return false
}

// keyDown reports a key as held if the user is pressing it or the script holds it now.
func (g *gameState) keyDown(k ebiten.Key) bool {
	return ebiten.IsKeyPressed(k) || g.script.held(k, g.frame)
}

// keyTapped reports a fresh press from the user or a scripted tap on this frame.
func (g *gameState) keyTapped(k ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(k) || g.script.justPressed(k, g.frame)
}

// buildScript turns a comma-separated keybind list (e.g. "space", "left", "replica,tab")
// into a timeline over recFrames frames. Tap actions fire in order, each in its own slot,
// so multi-key demos work (e.g. switch to the replica grid, then cycle focus). A lone
// toggle (space/x/m) is tapped twice — on, then off — to show both states. Orbit/zoom
// keys are held for the whole clip. It returns nil when no keys are given.
func buildScript(keys string, recFrames int) *keyScript {
	const (
		intro = 12 // brief pause so the opening state is visible before acting
		gap   = 18 // frames between successive tap actions
	)
	var toks []string
	for _, raw := range strings.Split(keys, ",") {
		if tok := strings.ToLower(strings.TrimSpace(raw)); tok != "" {
			toks = append(toks, tok)
		}
	}
	if len(toks) == 0 {
		return nil
	}

	s := &keyScript{taps: map[ebiten.Key][]int{}, holds: map[ebiten.Key][]frameRange{}}
	lone := len(toks) == 1
	tap := func(k ebiten.Key, at int) { s.taps[k] = append(s.taps[k], at) }
	hold := func(k ebiten.Key) { s.holds[k] = append(s.holds[k], frameRange{intro, recFrames}) }
	toggle := func(k ebiten.Key, at int) {
		tap(k, at)
		if lone {
			tap(k, recFrames/2) // show the off state for a single-toggle demo
		}
	}

	slot := intro
	for _, tok := range toks {
		switch tok {
		case "space":
			toggle(ebiten.KeySpace, slot)
		case "x":
			toggle(ebiten.KeyX, slot)
		case "m":
			toggle(ebiten.KeyM, slot)
		case "r":
			tap(ebiten.KeyR, slot)
		case "s":
			tap(ebiten.KeyS, slot)
		case "tab":
			tap(ebiten.KeyTab, slot)
		case "1", "single":
			tap(ebiten.KeyDigit1, slot)
		case "2", "compare":
			tap(ebiten.KeyDigit2, slot)
		case "3", "replica":
			tap(ebiten.KeyDigit3, slot)
		case "left":
			hold(ebiten.KeyArrowLeft)
		case "right":
			hold(ebiten.KeyArrowRight)
		case "up":
			hold(ebiten.KeyArrowUp)
		case "down":
			hold(ebiten.KeyArrowDown)
		case "shift+up":
			hold(ebiten.KeyShiftLeft)
			hold(ebiten.KeyArrowUp)
		case "shift+down":
			hold(ebiten.KeyShiftLeft)
			hold(ebiten.KeyArrowDown)
		}
		slot += gap
	}
	return s
}

func (g *gameState) Update() error {
	if g.rec != nil {
		g.frame++
		if g.rec.len() >= g.recFrames {
			if !g.recSaved {
				if err := g.rec.save(g.recPath); err != nil {
					return err
				}
				g.recSaved = true
			}
			return ebiten.Termination
		}
	}

	for _, v := range g.views {
		v.drainSolve()
	}

	shift := g.keyDown(ebiten.KeyShiftLeft) || g.keyDown(ebiten.KeyShiftRight)

	// Orbit (only while folded up); the view locks and faces the net once unfolded.
	if g.unfold < 0.5 {
		if g.keyDown(ebiten.KeyArrowLeft) {
			g.yaw -= orbitSpeed
		}
		if g.keyDown(ebiten.KeyArrowRight) {
			g.yaw += orbitSpeed
		}
		if !shift {
			if g.keyDown(ebiten.KeyArrowUp) {
				g.pitch -= orbitSpeed
			}
			if g.keyDown(ebiten.KeyArrowDown) {
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
		if g.keyDown(ebiten.KeyArrowUp) {
			g.zoom *= 1 + zoomSpeed
		}
		if g.keyDown(ebiten.KeyArrowDown) {
			g.zoom *= 1 - zoomSpeed
		}
	}
	g.zoom = clamp(g.zoom, 0.5, 3)

	if g.keyTapped(ebiten.KeySpace) {
		if g.unfoldTo == 0 {
			g.unfoldTo = 1
		} else {
			g.unfoldTo = 0
		}
	}
	g.unfold += clamp(g.unfoldTo-g.unfold, -unfoldSpeed, unfoldSpeed)

	if g.keyTapped(ebiten.KeyX) {
		g.xray = !g.xray
	}
	if g.keyTapped(ebiten.KeyR) {
		for _, v := range g.views {
			v.rescramble(v.ctrl.Scramble)
		}
	}
	if g.keyTapped(ebiten.KeyS) {
		v := g.views[g.focus]
		if len(v.ctrl.Strategies) > 0 {
			v.stratIdx = (v.stratIdx + 1) % len(v.ctrl.Strategies)
			v.label = fmt.Sprintf("#%d %s", g.focus, v.strategy())
			v.kickSolve()
		}
	}
	if g.keyTapped(ebiten.KeyTab) && g.grid() {
		g.focus = (g.focus + 1) % len(g.views)
	}
	if g.keyTapped(ebiten.KeyM) {
		g.showMoves = !g.showMoves
	}
	// Layout: 1 single cube, 2 compare grid (all solvers, one scramble), 3 replica grid.
	// Re-pressing the current mode regenerates it with fresh scrambles.
	if g.keyTapped(ebiten.KeyDigit1) {
		g.setMode(modeSingle)
	}
	if g.keyTapped(ebiten.KeyDigit2) {
		g.setMode(modeCompare)
	}
	if g.keyTapped(ebiten.KeyDigit3) {
		g.setMode(modeReplica)
	}

	for _, v := range g.views {
		v.advanceSolve(g.unfold)
	}
	return nil
}

func (g *gameState) Draw(screen *ebiten.Image) {
	screen.Fill(background)
	cw := float32(g.w) / float32(g.cols)
	ch := float32(g.h) / float32(g.rows)
	px := scale * g.zoom
	if g.grid() {
		px = scale * g.zoom * min(cw, ch) / winW
	}

	for k, v := range g.views {
		col, row := k%g.cols, k/g.cols
		ox, oy := float32(col)*cw, float32(row)*ch
		g.drawView(screen, v, ox+cw/2, oy+ch/2, px)
		if g.grid() {
			line := cellLine
			if k == g.focus {
				line = focusLine
			}
			vector.StrokeRect(screen, ox, oy, cw, ch, 1, line, false)
			ebitenutil.DebugPrintAt(screen, v.label+"  "+v.shortStatus(), int(ox)+6, int(oy)+6)
		}
	}

	fv := g.views[g.focus]
	if !g.grid() {
		ebitenutil.DebugPrintAt(screen, fv.status(), 12, 12)
	}
	help := "drag/arrows: orbit   wheel: zoom   space: unfold   x: x-ray   r: scramble   s: solver   m: moves   1/2/3: single/compare/replica"
	if g.grid() {
		help += "   tab: focus"
	}
	ebitenutil.DebugPrintAt(screen, help, 12, g.h-22)

	if g.showMoves {
		g.drawMoveList(screen, fv)
	}

	if g.rec != nil && g.rec.len() < g.recFrames {
		w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
		buf := make([]byte, 4*w*h)
		screen.ReadPixels(buf)
		g.rec.add(&image.RGBA{Pix: buf, Stride: 4 * w, Rect: image.Rect(0, 0, w, h)})
	}
}

// drawView projects one cube into a cell centred at (cx,cy) with px pixels per world
// unit. Camera angles, unfold, x-ray and geometry are shared from gameState.
func (g *gameState) drawView(screen *ebiten.Image, v *cubeView, cx, cy, px float32) {
	f := v.c.ToFacelets()
	sinY, cosY := fsincos(g.yaw)
	sinX, cosX := fsincos(g.pitch)
	key := render.Normalize(render.Vec3{X: 0.5, Y: 0.8, Z: 0.9})
	fill := render.Normalize(render.Vec3{X: -0.5, Y: 0.3, Z: 0.4})

	// Current animated turn angle, applied to the moving layer.
	turnAxis, turnAng := render.TurnAngle(v.turnMove, float32(v.turnFrame)/turnFrames)
	tsin, tcos := fsincos(turnAng)

	type face2d struct {
		pts    [4][2]float32
		depth  float32
		col    color.RGBA
		stroke bool
	}
	faces := make([]face2d, 0, len(g.polys))
	for _, p := range g.polys {
		if g.xray && p.Kind != render.Top {
			continue // x-ray: drop the plastic body so every side shows through
		}
		spin := v.turning && render.InTurnLayer(v.turnMove, p.Center)
		var pts [4][2]float32
		var depth float32
		for i := range 4 {
			pv := render.Lerp(p.Cube[i], p.Net[i], g.unfold)
			if spin {
				pv = render.RotateAxis(pv, turnAxis, tsin, tcos)
			}
			pv = render.RotateCamera(pv, sinY, cosY, sinX, cosX)
			pts[i] = [2]float32{cx + pv.X*px, cy - pv.Y*px}
			depth += pv.Z
		}
		n := p.Normal
		if spin {
			n = render.RotateAxis(n, turnAxis, tsin, tcos)
		}
		n = render.RotateCamera(n, sinY, cosY, sinX, cosX)
		lit := min(1, 0.6+0.32*max(0, render.Dot(n, key))+0.18*max(0, render.Dot(n, fill)))
		base := plastic
		if p.Kind == render.Top {
			base = render.FaceletColor(f[int(p.Face)*9+p.Cell])
		}
		col := shade(base, lit)
		if g.xray {
			col.A = 125
		}
		faces = append(faces, face2d{pts: pts, depth: depth / 4, col: col, stroke: p.Kind == render.Top && !g.xray})
	}
	sort.Slice(faces, func(i, j int) bool { return faces[i].depth < faces[j].depth })
	for _, fc := range faces {
		fillQuad(screen, fc.pts, fc.col)
		if fc.stroke {
			strokeQuad(screen, fc.pts)
		}
	}
}

// drawMoveList shows the moves played so far, top-right, growing move-by-move as the
// solve animates. The current move is marked; long lists scroll so it stays visible.
func (g *gameState) drawMoveList(screen *ebiten.Image, v *cubeView) {
	if len(v.moves) == 0 {
		return
	}
	const lineH, colW = 16, 92
	x0, y0 := g.w-colW, 30
	capRows := (g.h - y0 - 24) / lineH
	if capRows < 1 {
		capRows = 1
	}
	last := min(v.idx, len(v.moves)-1) // reveal up to the current move
	start := 0
	if last-start+1 > capRows {
		start = last - capRows + 1
	}

	played := min(v.idx, len(v.moves))
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("moves %d/%d", played, len(v.moves)), x0, 12)
	for row, i := 0, start; i <= last; row, i = row+1, i+1 {
		y := y0 + row*lineH
		marker := "  "
		if i == v.idx && v.idx < len(v.moves) {
			vector.FillRect(screen, float32(x0-2), float32(y-1), colW, lineH, highlight, false)
			marker = "> "
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s%2d %s", marker, i+1, v.moves[i].String()), x0, y)
	}
}

func (g *gameState) Layout(int, int) (int, int) { return g.w, g.h }

func fillQuad(screen *ebiten.Image, q [4][2]float32, col color.RGBA) {
	var path vector.Path
	path.MoveTo(q[0][0], q[0][1])
	path.LineTo(q[1][0], q[1][1])
	path.LineTo(q[2][0], q[2][1])
	path.LineTo(q[3][0], q[3][1])
	path.Close()
	// The v2.9 replacement (vector.FillPath) batches and flushes via a callback, which
	// would reorder these fills relative to the strokes drawn between them and break the
	// back-to-front painter ordering. Keep the immediate-mode vertex path on purpose.
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil) //nolint:staticcheck // SA1019
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
func fsincos(a float32) (float32, float32) {
	s, c := math.Sincos(float64(a))
	return float32(s), float32(c)
}

// gridDims picks a near-square column/row count for n cells.
func gridDims(n int) (cols, rows int) {
	if n <= 1 {
		return 1, 1
	}
	cols = int(math.Ceil(math.Sqrt(float64(n))))
	rows = (n + cols - 1) / cols
	return cols, rows
}

// windowSize is the single-cube window for a 1×1 grid, else a grid-sized window capped
// at maxWin so cells stay legible.
func windowSize(cols, rows int) (w, h int) {
	if cols <= 1 && rows <= 1 {
		return winW, winH
	}
	clampI := func(v, lo, hi int) int { return max(lo, min(v, hi)) }
	return clampI(cols*cellSize, winW, maxWin), clampI(rows*cellSize, winH, maxWin)
}

func indexOf(names []string, name string) int {
	for i, n := range names {
		if n == name {
			return i
		}
	}
	return 0
}

func newSingleView(ctrl Controller) *cubeView {
	v := &cubeView{ctrl: ctrl, stratIdx: ctrl.Start, solveCh: make(chan solveOut, 8)}
	if ctrl.Initial != nil {
		v.start = *ctrl.Initial
	} else {
		v.start = ctrl.Scramble()
	}
	return v
}

func newGridView(ctrl Controller, cell Cell) *cubeView {
	return &cubeView{
		ctrl:     ctrl,
		start:    cell.Initial,
		stratIdx: indexOf(ctrl.Strategies, cell.Strategy),
		label:    cell.Label,
		solveCh:  make(chan solveOut, 8),
	}
}

// Play opens the visualizer driven by ctrl. It starts as a single self-driving cube;
// the 2 and 3 keys switch to the compare and replica grids at runtime.
func Play(ctrl Controller) error {
	g := &gameState{
		ctrl:  ctrl,
		mode:  modeSingle,
		polys: render.BuildPolys(),
		yaw:   0.6, pitch: 0.5, zoom: 1,
	}
	g.views = []*cubeView{newSingleView(ctrl)}
	g.cols, g.rows = gridDims(len(g.views))
	g.w, g.h = windowSize(g.cols, g.rows)
	if ctrl.Record != "" {
		g.recPath = ctrl.Record
		g.recFrames = ctrl.RecordFrames
		if g.recFrames < 1 {
			g.recFrames = 120
		}
		g.rec = newRecorder(ctrl.RecordScale, ctrl.RecordFPS)
		g.script = buildScript(ctrl.RecordKeys, g.recFrames)
	}
	for _, v := range g.views {
		v.kickSolve()
	}
	ebiten.SetWindowSize(g.w, g.h)
	ebiten.SetWindowTitle("rubix")
	return ebiten.RunGame(g)
}
