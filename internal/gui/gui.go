//go:build ebiten

package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/danielriddell21/crucible/canvas"
	"github.com/danielriddell21/crucible/record"
	"github.com/danielriddell21/crucible/window"

	"github.com/danielriddell21/rubix/pkg/cube"
	"github.com/danielriddell21/rubix/pkg/render"
)

var whiteSub *ebiten.Image

func init() {
	w := ebiten.NewImage(3, 3)
	w.Fill(color.White)
	whiteSub = w.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}

const (
	winH        = 640
	netFrames   = 8
	orbitSpeed  = 0.035
	zoomSpeed   = 0.04
	unfoldSpeed = 0.04
	cellSize    = 320
	maxWin      = 1280
	maxCells    = 16
)

func Available() bool { return true }

type solveOut struct {
	gen    int
	moves  []cube.Move
	solved bool
}

type cubeView struct {
	ctrl     Controller
	start    cube.Cube
	stratIdx int
	label    string
	solving  bool
	gen      int
	solveCh  chan solveOut

	c          cube.Cube
	moves      []cube.Move
	idx        int
	frame      int
	lastSolved bool

	turning    bool
	turnFrame  int
	turnMove   cube.Move
	manualTurn bool
}

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

func (v *cubeView) drainSolve() {
	select {
	case out := <-v.solveCh:
		if out.gen == v.gen {
			v.moves, v.solving, v.lastSolved = out.moves, false, out.solved
		}
	default:
	}
}

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
			if v.manualTurn {
				v.manualTurn = false // a by-hand turn: the solution index stays put
			} else {
				v.idx++
			}
		}
		return
	}
	if v.idx < len(v.moves) {
		v.turning, v.turnFrame, v.turnMove = true, 0, v.moves[v.idx]
	}
}

func (v *cubeView) applyManual(m cube.Move) {
	if v.turning {
		return
	}
	v.turning, v.turnFrame, v.turnMove, v.manualTurn = true, 0, m, true
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

type gameState struct {
	ctrl       Controller
	views      []*cubeView
	cols, rows int
	w, h       int

	polys []render.Poly

	yaw, pitch float32
	zoom       float32
	unfold     float32
	unfoldTo   float32
	xray       bool
	dragging   bool
	lastX      int
	lastY      int

	focus     int
	showMoves bool
	manual    bool

	rec       *record.Recorder
	canvas    *canvas.Canvas
	recPath   string
	recFrames int
	recSaved  bool
	frame     int
	script    *keyScript

	link     *Link
	lastSent Msg
}

func (g *gameState) shared() Msg {
	return Msg{
		Type: "state", Yaw: g.yaw, Pitch: g.pitch, Zoom: g.zoom,
		UnfoldTo: g.unfoldTo, Xray: g.xray, ShowMoves: g.showMoves,
	}
}

func (g *gameState) applyShared(m Msg) {
	g.yaw, g.pitch, g.zoom = m.Yaw, m.Pitch, m.Zoom
	g.unfoldTo, g.xray, g.showMoves = m.UnfoldTo, m.Xray, m.ShowMoves
	g.lastSent = g.shared()
}

func (g *gameState) grid() bool { return len(g.views) > 1 }

func (g *gameState) defaultStrategy() string {
	if g.ctrl.Start >= 0 && g.ctrl.Start < len(g.ctrl.Strategies) {
		return g.ctrl.Strategies[g.ctrl.Start]
	}
	if len(g.ctrl.Strategies) > 0 {
		return g.ctrl.Strategies[0]
	}
	return ""
}

func (g *gameState) newCube(i int) *cubeView {
	name := g.defaultStrategy()
	return newGridView(g.ctrl, Cell{Strategy: name, Initial: g.ctrl.Scramble(), Label: fmt.Sprintf("#%d %s", i, name)})
}

func (g *gameState) relayout() {
	g.cols, g.rows = gridDims(len(g.views))
	if g.focus >= len(g.views) {
		g.focus = len(g.views) - 1
	}
	if g.rec == nil {
		g.w, g.h = fitMonitor(windowSize(g.cols, g.rows))
		ebiten.SetWindowSize(g.w, g.h)
	}
}

func (g *gameState) setCount(n int) {
	n = clampInt(n, 1, maxCells)
	if n == len(g.views) {
		return
	}
	switch {
	case n == 1:
		v := newSingleView(g.ctrl)
		g.views = []*cubeView{v}
		v.kickSolve()
	case len(g.views) >= 2 && n > len(g.views): // grow the grid, keep existing cubes
		for i := len(g.views); i < n; i++ {
			v := g.newCube(i)
			g.views = append(g.views, v)
			v.kickSolve()
		}
	case len(g.views) >= 2: // shrink the grid
		g.views = g.views[:n]
	default: // single → grid: build a fresh set
		g.views = make([]*cubeView, 0, n)
		for i := range n {
			v := g.newCube(i)
			g.views = append(g.views, v)
			v.kickSolve()
		}
	}
	g.relayout()
}

func clampInt(v, lo, hi int) int { return max(lo, min(v, hi)) }

type frameRange struct{ lo, hi int }

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

func (g *gameState) keyDown(k ebiten.Key) bool {
	return ebiten.IsKeyPressed(k) || g.script.held(k, g.frame)
}

func (g *gameState) keyTapped(k ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(k) || g.script.justPressed(k, g.frame)
}

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
		switch {
		case scriptToggleKeys[tok] != nil:
			toggle(*scriptToggleKeys[tok], slot)
		case scriptTapKeys[tok] != nil:
			tap(*scriptTapKeys[tok], slot)
		default:
			for _, k := range scriptHoldKeys[tok] {
				hold(k)
			}
		}
		slot += gap
	}
	return s
}

var (
	scriptToggleKeys = map[string]*ebiten.Key{
		"space": keyPtr(ebiten.KeySpace),
		"x":     keyPtr(ebiten.KeyX),
		"m":     keyPtr(ebiten.KeyM),
	}
	scriptTapKeys = map[string]*ebiten.Key{
		"r":     keyPtr(ebiten.KeyR),
		"s":     keyPtr(ebiten.KeyS),
		"tab":   keyPtr(ebiten.KeyTab),
		"+":     keyPtr(ebiten.KeyEqual),
		"plus":  keyPtr(ebiten.KeyEqual),
		"-":     keyPtr(ebiten.KeyMinus),
		"minus": keyPtr(ebiten.KeyMinus),
	}
	scriptHoldKeys = map[string][]ebiten.Key{
		"left":       {ebiten.KeyArrowLeft},
		"right":      {ebiten.KeyArrowRight},
		"up":         {ebiten.KeyArrowUp},
		"down":       {ebiten.KeyArrowDown},
		"shift+up":   {ebiten.KeyShiftLeft, ebiten.KeyArrowUp},
		"shift+down": {ebiten.KeyShiftLeft, ebiten.KeyArrowDown},
	}
)

func keyPtr(k ebiten.Key) *ebiten.Key { return &k }

func (g *gameState) Update() error {
	if stop, err := g.tickRecording(); stop {
		if err != nil {
			return err
		}
		return ebiten.Termination
	}

	for _, v := range g.views {
		v.drainSolve()
	}

	shift := g.keyDown(ebiten.KeyShiftLeft) || g.keyDown(ebiten.KeyShiftRight)
	g.handleOrbit(shift)
	g.handleZoom(shift)
	g.handleKeys()

	for _, v := range g.views {
		v.advanceSolve(g.unfold)
	}

	return g.syncLink()
}

func (g *gameState) tickRecording() (stop bool, err error) {
	if g.rec == nil {
		return false, nil
	}
	g.frame++
	if g.rec.Len() < g.recFrames {
		return false, nil
	}
	if !g.recSaved {
		if err := g.rec.Save(g.recPath); err != nil {
			return true, fmt.Errorf("save recording: %w", err)
		}
		g.recSaved = true
	}
	return true, nil
}

func (g *gameState) handleOrbit(shift bool) {
	if g.unfold >= 0.5 {
		g.dragging = false
		g.yaw += clamp(-g.yaw, -0.08, 0.08)
		g.pitch += clamp(-g.pitch, -0.08, 0.08)
		return
	}
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
}

func (g *gameState) handleZoom(shift bool) {
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
}

func (g *gameState) handleKeys() {
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
	if g.keyTapped(ebiten.KeyEnter) || g.keyTapped(ebiten.KeyKPEnter) {
		g.toggleManual()
	}
	if g.manual {
		g.handleManualMoves()
	} else {
		if g.keyTapped(ebiten.KeyR) {
			for _, v := range g.views {
				v.rescramble(v.ctrl.Scramble)
			}
			g.send(Msg{Type: "rescramble"}) // make the other windows rescramble too
		}
		if g.keyTapped(ebiten.KeyS) {
			v := g.views[g.focus]
			if len(v.ctrl.Strategies) > 0 {
				v.stratIdx = (v.stratIdx + 1) % len(v.ctrl.Strategies)
				v.label = fmt.Sprintf("#%d %s", g.focus, v.strategy())
				v.kickSolve()
			}
		}
	}
	if g.keyTapped(ebiten.KeyTab) && g.grid() {
		g.focus = (g.focus + 1) % len(g.views)
	}
	if g.keyTapped(ebiten.KeyM) {
		g.showMoves = !g.showMoves
	}
	g.handleCountKeys()
}

func (g *gameState) handleCountKeys() {
	if g.keyTapped(ebiten.KeyEqual) || g.keyTapped(ebiten.KeyKPAdd) {
		if g.link != nil {
			g.send(Msg{Type: "add"})
		} else {
			g.setCount(len(g.views) + 1)
		}
	}
	if g.keyTapped(ebiten.KeyMinus) || g.keyTapped(ebiten.KeyKPSubtract) {
		if g.link != nil {
			g.send(Msg{Type: "remove"})
		} else {
			g.setCount(len(g.views) - 1)
		}
	}
}

func (g *gameState) toggleManual() {
	g.manual = !g.manual
	v := g.views[g.focus]
	if g.manual {
		v.gen++ // discard any in-flight solve so it can't repopulate the move list
		v.moves, v.idx, v.turning, v.solving = nil, 0, false, false
		v.start = v.c
		return
	}
	v.start = v.c
	v.kickSolve()
}

var manualFaces = []struct {
	key  ebiten.Key
	face int
}{
	{ebiten.KeyU, 0},
	{ebiten.KeyR, 1},
	{ebiten.KeyF, 2},
	{ebiten.KeyD, 3},
	{ebiten.KeyL, 4},
	{ebiten.KeyB, 5},
}

func (g *gameState) handleManualMoves() {
	prime := g.keyDown(ebiten.KeyShiftLeft) || g.keyDown(ebiten.KeyShiftRight)
	v := g.views[g.focus]
	for _, fk := range manualFaces {
		if !g.keyTapped(fk.key) {
			continue
		}
		m := cube.Move(fk.face * 3) // clockwise quarter turn
		if prime {
			m = cube.Move(fk.face*3 + 2) // counter-clockwise
		}
		if g.unfold >= 0.5 {
			v.c.Apply(m) // no turn animation while unfolded to the flat net
		} else {
			v.applyManual(m)
		}
	}
}

func (g *gameState) syncLink() error {
	if g.link == nil {
		return nil
	}
	for done := false; !done; {
		select {
		case m, ok := <-g.link.In:
			if !ok || m.Type == "quit" {
				return ebiten.Termination // leader/hub gone, or asked to close
			}
			switch m.Type {
			case "state":
				g.applyShared(m)
			case "rescramble":
				for _, v := range g.views {
					v.rescramble(v.ctrl.Scramble)
				}
			}
		default:
			done = true
		}
	}
	if cur := g.shared(); cur != g.lastSent {
		if g.send(cur) {
			g.lastSent = cur
		}
	}
	return nil
}

func (g *gameState) send(m Msg) bool {
	if g.link == nil || g.link.Out == nil {
		return false
	}
	select {
	case g.link.Out <- m:
		return true
	default:
		return false
	}
}

func (g *gameState) Draw(screen *ebiten.Image) {
	if g.canvas == nil || func() bool { w, h := g.canvas.Size(); return w != g.w || h != g.h }() {
		g.canvas = canvas.New(g.w, g.h)
	}
	DrawScene(g.canvas, g.scene())
	screen.WritePixels(g.canvas.Pixels())

	if g.rec != nil && g.rec.Len() < g.recFrames {
		g.rec.Add(record.FromRGBA(g.canvas.Pixels(), g.w, g.h))
	}
}

// scene lifts the drawable state out of the game so the composer needs nothing
// from the display.
func (g *gameState) scene() SceneView {
	cubes := make([]CubeView, len(g.views))
	for i, v := range g.views {
		cubes[i] = CubeView{
			Facelets: v.c.ToFacelets(), Label: v.label, Status: v.status(), Short: v.shortStatus(),
			Turning: v.turning, TurnMove: v.turnMove, TurnFrame: v.turnFrame,
			Idx: v.idx, Moves: v.moves,
		}
	}
	return SceneView{
		Cubes: cubes, Cols: g.cols, Rows: g.rows, W: g.w, H: g.h, Focus: g.focus,
		Yaw: g.yaw, Pitch: g.pitch, Zoom: g.zoom, Unfold: g.unfold,
		Xray: g.xray, Manual: g.manual, ShowMoves: g.showMoves, Grid: g.grid(),
		Polys: g.polys,
	}
}

func (g *gameState) Layout(outsideW, outsideH int) (int, int) {
	if g.rec == nil && outsideW > 0 && outsideH > 0 {
		g.w, g.h = outsideW, outsideH
	}
	return g.w, g.h
}

func fitMonitor(w, h int) (int, int) {
	mw, mh := ebiten.Monitor().Size()
	if mw <= 0 || mh <= 0 {
		return w, h
	}
	maxW, maxH := mw*9/10, mh*9/10
	s := 1.0
	if w > maxW {
		s = math.Min(s, float64(maxW)/float64(w))
	}
	if h > maxH {
		s = math.Min(s, float64(maxH)/float64(h))
	}
	if s < 1 {
		w, h = int(float64(w)*s), int(float64(h)*s)
	}
	return w, h
}

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

func gridDims(n int) (cols, rows int) {
	if n <= 1 {
		return 1, 1
	}
	cols = int(math.Ceil(math.Sqrt(float64(n))))
	rows = (n + cols - 1) / cols
	return cols, rows
}

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

func Run(cfg Config) error {
	ctrl, link := cfg.Controller, cfg.Link
	g := &gameState{
		ctrl:  ctrl,
		link:  link,
		polys: render.BuildPolys(),
		yaw:   0.6, pitch: 0.5, zoom: 1,
	}
	g.views = []*cubeView{newSingleView(ctrl)}
	g.cols, g.rows = gridDims(len(g.views))
	g.w, g.h = windowSize(g.cols, g.rows)
	if ctrl.Rec.Recording() {
		g.recPath = ctrl.Rec.Path
		g.recFrames = ctrl.Rec.Frames
		if g.recFrames < 1 {
			g.recFrames = 120
		}
		// maxFrames stays 0: rubix caps the recording itself so the scripted
		// keybinds and the frame budget stay in step.
		g.rec = record.NewRecorder(ctrl.Rec.FPS, ctrl.Rec.Scale, 0)
		g.script = buildScript(ctrl.RecordKeys, g.recFrames)
	}
	for _, v := range g.views {
		v.kickSolve()
	}
	g.lastSent = g.shared() // suppress an initial publish; all windows start identical
	title := ctrl.Title
	if title == "" {
		title = "rubix"
	}
	if g.rec == nil {
		// The user may resize the window; Layout reflows the scene to the new size.
		window.Configure(window.Options{Title: title, Width: g.w, Height: g.h, MinWidth: winW / 2, MinHeight: winH / 2})
	} else {
		// During recording the window stays a fixed size so the GIF dimensions are stable.
		ebiten.SetWindowSize(g.w, g.h)
		ebiten.SetWindowTitle(title)
	}
	if ctrl.OffsetIndex > 0 {
		// Cascade child windows so they don't open exactly on top of the leader.
		ebiten.SetWindowPosition(60+ctrl.OffsetIndex*36, 60+ctrl.OffsetIndex*36)
	}
	if link != nil {
		// Coordinated windows must keep updating while unfocused, so background windows
		// stay in sync, keep animating, and notice the leader closing (their Update must
		// run to drain the closed link and terminate).
		ebiten.SetRunnableOnUnfocused(true)
	}
	if err := ebiten.RunGame(g); err != nil {
		return fmt.Errorf("run window: %w", err)
	}
	return nil
}
