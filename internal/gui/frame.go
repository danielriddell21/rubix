package gui

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/danielriddell21/crucible/canvas"
	"github.com/danielriddell21/crucible/keymap"

	"github.com/danielriddell21/rubix/pkg/cube"
	"github.com/danielriddell21/rubix/pkg/render"
)

// textAscent lifts a top-left text origin onto the canvas face's baseline.
const textAscent = 11

// helpLineH is the line spacing of the control hints along the bottom.
const helpLineH = 16

// colText is the HUD text colour.
var colText = color.RGBA{R: 230, G: 230, B: 230, A: 255}

// DrawScene composes a whole frame onto c — every cube, the grid chrome, the
// status line, the control hints and the move list. Drawing through the
// software canvas rather than the display means the same code paints a live
// window and a headless recording, pixel for pixel.
func DrawScene(c *canvas.Canvas, g SceneView) {
	c.Fill(background)
	cw := float32(g.W) / float32(g.Cols)
	ch := float32(g.H) / float32(g.Rows)
	px := scale * g.Zoom
	if g.Grid {
		px = scale * g.Zoom * min(cw, ch) / winW
	}

	for k, v := range g.Cubes {
		col, row := k%g.Cols, k/g.Cols
		ox, oy := float32(col)*cw, float32(row)*ch
		drawCube(c, g, v, ox+cw/2, oy+ch/2, px)
		if !g.Grid {
			continue
		}
		line := cellLine
		if k == g.Focus {
			line = focusLine
		}
		strokeRect(c, float64(ox), float64(oy), float64(cw), float64(ch), line)
		c.Text(int(ox)+6, int(oy)+6+textAscent, v.Label+"  "+v.Short, colText)
	}

	fv := g.Cubes[g.Focus]
	if !g.Grid {
		status := fv.Status
		if g.Manual {
			status = "MANUAL — turn the cube by hand"
		}
		c.Text(12, 12+textAscent, status, colText)
	}
	drawHelp(c, g)
	if g.ShowMoves {
		drawMoveList(c, g, fv)
	}
}

// drawCube projects one cube's faces and paints them back to front.
func drawCube(c *canvas.Canvas, g SceneView, v CubeView, cx, cy, px float32) {
	f := v.Facelets
	sinY, cosY := fsincos(g.Yaw)
	sinX, cosX := fsincos(g.Pitch)
	key := render.Normalize(render.Vec3{X: 0.5, Y: 0.8, Z: 0.9})
	fill := render.Normalize(render.Vec3{X: -0.5, Y: 0.3, Z: 0.4})

	// Current animated turn angle, applied to the moving layer.
	turnAxis, turnAng := render.TurnAngle(v.TurnMove, float32(v.TurnFrame)/turnFrames)
	tsin, tcos := fsincos(turnAng)

	type face2d struct {
		pts    [4][2]float64
		depth  float32
		col    color.RGBA
		stroke bool
	}
	faces := make([]face2d, 0, len(g.Polys))
	for _, p := range g.Polys {
		if g.Xray && p.Kind != render.Top {
			continue // x-ray: drop the plastic body so every side shows through
		}
		spin := v.Turning && render.InTurnLayer(v.TurnMove, p.Center)
		var pts [4][2]float64
		var depth float32
		for i := range 4 {
			pv := render.Lerp(p.Cube[i], p.Net[i], g.Unfold)
			if spin {
				pv = render.RotateAxis(pv, turnAxis, tsin, tcos)
			}
			pv = render.RotateCamera(pv, sinY, cosY, sinX, cosX)
			pts[i] = [2]float64{float64(cx + pv.X*px), float64(cy - pv.Y*px)}
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
		if g.Xray {
			col.A = 125
		}
		faces = append(faces, face2d{pts: pts, depth: depth / 4, col: col, stroke: p.Kind == render.Top && !g.Xray})
	}
	sort.Slice(faces, func(i, j int) bool { return faces[i].depth < faces[j].depth })
	for _, fc := range faces {
		c.Polygon(fc.pts[:], fc.col)
		if fc.stroke {
			strokeQuadTo(c, fc.pts)
		}
	}
}

// helpPad insets the control bar from the window edges.
const helpPad = 12

// helpFace tells keymap how wide the canvas's built-in face draws, so the bar
// wraps where it actually runs out of room.
var helpFace = keymap.Face{
	LineHeight: helpLineH,
	Measure:    func(s string) int { return len(s) * canvas.GlyphWidth },
}

func drawHelp(c *canvas.Canvas, g SceneView) {
	hints := []keymap.Binding{
		{Key: "drag/arrows", Action: "orbit"},
		{Key: "wheel", Action: "zoom"},
		{Key: "space", Action: "unfold"},
		{Key: "x", Action: "x-ray"},
	}
	if g.Manual {
		hints = append(hints,
			keymap.Binding{Key: "U R F D L B", Action: "turn (shift: prime)"},
			keymap.Binding{Key: "enter", Action: "resume"})
	} else {
		hints = append(hints,
			keymap.Binding{Key: "r", Action: "scramble"},
			keymap.Binding{Key: "s", Action: "solver"},
			keymap.Binding{Key: "enter", Action: "manual"})
	}
	hints = append(hints,
		keymap.Binding{Key: "m", Action: "moves"},
		keymap.Binding{Key: "+/-", Action: "cubes"})
	if g.Grid {
		hints = append(hints, keymap.Binding{Key: "tab", Action: "focus"})
	}
	// BottomBar wraps to the window width and stacks the rows up from the
	// bottom, so nothing clips however many hints the mode adds.
	for _, line := range keymap.BottomBar(hints, g.W, g.H, helpPad, helpFace) {
		c.Text(line.X, line.Y+textAscent, line.Text, colText)
	}
}

func drawMoveList(c *canvas.Canvas, g SceneView, v CubeView) {
	if len(v.Moves) == 0 {
		return
	}
	const lineH, colW = 16, 92
	x0, y0 := g.W-colW, 30
	capRows := max((g.H-y0-24)/lineH, 1)
	last := min(v.Idx, len(v.Moves)-1) // reveal up to the current move
	start := 0
	if last-start+1 > capRows {
		start = last - capRows + 1
	}

	played := min(v.Idx, len(v.Moves))
	c.Text(x0, 12+textAscent, fmt.Sprintf("moves %d/%d", played, len(v.Moves)), colText)
	for row, i := 0, start; i <= last; row, i = row+1, i+1 {
		y := y0 + row*lineH
		marker := "  "
		if i == v.Idx && v.Idx < len(v.Moves) {
			c.Rect(x0-2, y-1, colW, lineH, highlight)
			marker = "> "
		}
		c.Text(x0, y+textAscent, fmt.Sprintf("%s%2d %s", marker, i+1, v.Moves[i].String()), colText)
	}
}

// strokeRect outlines a rectangle a pixel wide.
func strokeRect(c *canvas.Canvas, x, y, w, h float64, col color.RGBA) {
	c.Line(x, y, x+w, y, 1, col)
	c.Line(x+w, y, x+w, y+h, 1, col)
	c.Line(x+w, y+h, x, y+h, 1, col)
	c.Line(x, y+h, x, y, 1, col)
}

// strokeQuadTo outlines a facelet so neighbouring stickers stay distinct.
func strokeQuadTo(c *canvas.Canvas, q [4][2]float64) {
	for i := range 4 {
		j := (i + 1) % 4
		c.Line(q[i][0], q[i][1], q[j][0], q[j][1], 1.2, outline)
	}
}

// CubeView is one cube's drawable state.
type CubeView struct {
	Facelets  cube.Facelets
	Label     string
	Status    string
	Short     string
	Turning   bool
	TurnMove  cube.Move
	TurnFrame int
	Idx       int
	Moves     []cube.Move
}

// SceneView is everything a frame draws, lifted out of the windowed game state
// so the composer needs nothing from the display.
type SceneView struct {
	Cubes                    []CubeView
	Cols, Rows, W, H, Focus  int
	Yaw, Pitch, Zoom, Unfold float32
	Xray, Manual, ShowMoves  bool
	Grid                     bool
	Polys                    []render.Poly
}
