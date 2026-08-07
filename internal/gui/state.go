package gui

import (
	"errors"
	"fmt"
	"math"

	"github.com/danielriddell21/rubix/pkg/cube"
	"github.com/danielriddell21/rubix/pkg/render"
)

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

// errQuit ends the run: the leader window closed the link, or asked this
// window to close. The display translates it into its own termination signal.
var errQuit = errors.New("quit")

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

// gameState is the whole viewer minus the display: it reads an [input], steps
// the cubes, and hands a [SceneView] to whoever draws. The window and the
// headless recorder both drive it, so a recording shows exactly what a player
// would see.
type gameState struct {
	ctrl       Controller
	in         input
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

	// resize refits the frame when the grid grows or shrinks. The window
	// resizes itself through it; a recording leaves it nil so the media keeps
	// stable dimensions.
	resize func(cols, rows int) (w, h int)

	link     *Link
	lastSent Msg
}

// newState builds the viewer's starting state: one cube, solving, sized for a
// single view.
func newState(cfg Config, in input) *gameState {
	g := &gameState{
		ctrl:  cfg.Controller,
		in:    in,
		link:  cfg.Link,
		polys: render.BuildPolys(),
		yaw:   0.6, pitch: 0.5, zoom: 1,
	}
	g.views = []*cubeView{newSingleView(cfg.Controller)}
	g.cols, g.rows = gridDims(len(g.views))
	g.w, g.h = windowSize(g.cols, g.rows)
	for _, v := range g.views {
		v.kickSolve()
	}
	g.lastSent = g.shared() // suppress an initial publish; all windows start identical
	return g
}

// step advances the viewer by one frame: drain finished solves, act on the
// input, animate, then exchange state with any linked windows.
func (g *gameState) step() error {
	for _, v := range g.views {
		v.drainSolve()
	}

	shift := g.in.Down(keyShift)
	g.handleOrbit(shift)
	g.handleZoom(shift)
	g.handleKeys()

	for _, v := range g.views {
		v.advanceSolve(g.unfold)
	}

	return g.syncLink()
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
	if g.resize != nil {
		g.w, g.h = g.resize(g.cols, g.rows)
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

func (g *gameState) handleOrbit(shift bool) {
	if g.unfold >= 0.5 {
		g.dragging = false
		g.yaw += clamp(-g.yaw, -0.08, 0.08)
		g.pitch += clamp(-g.pitch, -0.08, 0.08)
		return
	}
	if g.in.Down(keyArrowLeft) {
		g.yaw -= orbitSpeed
	}
	if g.in.Down(keyArrowRight) {
		g.yaw += orbitSpeed
	}
	if !shift {
		if g.in.Down(keyArrowUp) {
			g.pitch -= orbitSpeed
		}
		if g.in.Down(keyArrowDown) {
			g.pitch += orbitSpeed
		}
	}
	if x, y, ok := g.in.Drag(); ok {
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
	if dy := g.in.Wheel(); dy != 0 {
		g.zoom *= float32(math.Pow(1.12, dy))
	}
	if shift {
		if g.in.Down(keyArrowUp) {
			g.zoom *= 1 + zoomSpeed
		}
		if g.in.Down(keyArrowDown) {
			g.zoom *= 1 - zoomSpeed
		}
	}
	g.zoom = clamp(g.zoom, 0.5, 3)
}

func (g *gameState) handleKeys() {
	if g.in.Tapped(keySpace) {
		if g.unfoldTo == 0 {
			g.unfoldTo = 1
		} else {
			g.unfoldTo = 0
		}
	}
	g.unfold += clamp(g.unfoldTo-g.unfold, -unfoldSpeed, unfoldSpeed)

	if g.in.Tapped(keyX) {
		g.xray = !g.xray
	}
	if g.in.Tapped(keyEnter) {
		g.toggleManual()
	}
	if g.manual {
		g.handleManualMoves()
	} else {
		if g.in.Tapped(keyR) {
			for _, v := range g.views {
				v.rescramble(v.ctrl.Scramble)
			}
			g.send(Msg{Type: "rescramble"}) // make the other windows rescramble too
		}
		if g.in.Tapped(keyS) {
			v := g.views[g.focus]
			if len(v.ctrl.Strategies) > 0 {
				v.stratIdx = (v.stratIdx + 1) % len(v.ctrl.Strategies)
				v.label = fmt.Sprintf("#%d %s", g.focus, v.strategy())
				v.kickSolve()
			}
		}
	}
	if g.in.Tapped(keyTab) && g.grid() {
		g.focus = (g.focus + 1) % len(g.views)
	}
	if g.in.Tapped(keyM) {
		g.showMoves = !g.showMoves
	}
	g.handleCountKeys()
}

func (g *gameState) handleCountKeys() {
	if g.in.Tapped(keyPlus) {
		if g.link != nil {
			g.send(Msg{Type: "add"})
		} else {
			g.setCount(len(g.views) + 1)
		}
	}
	if g.in.Tapped(keyMinus) {
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
	key  key
	face int
}{
	{keyU, 0},
	{keyR, 1},
	{keyF, 2},
	{keyD, 3},
	{keyL, 4},
	{keyB, 5},
}

func (g *gameState) handleManualMoves() {
	prime := g.in.Down(keyShift)
	v := g.views[g.focus]
	for _, fk := range manualFaces {
		if !g.in.Tapped(fk.key) {
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
				return errQuit // leader/hub gone, or asked to close
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
