//go:build ebiten

// Package gui is an Ebiten visualizer of the live cube state. It renders the shared
// cube.Cube as a colour net and animates a solution move by move, so a solve (or a
// real EV3 scan/execute) can be watched. Built only with the `ebiten` tag.
package gui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/danielriddell21/rubix/internal/cube"
)

const (
	cell          = 34 // sticker size in px
	framesPerMove = 18 // animation pacing
	marginX       = 30
	marginY       = 30
)

// faceOrigin places each face in a standard unfolded cross (units of one cell).
var faceOrigin = [6][2]int{
	cube.ColU: {3, 0},
	cube.ColR: {6, 3},
	cube.ColF: {3, 3},
	cube.ColD: {3, 6},
	cube.ColL: {0, 3},
	cube.ColB: {9, 3},
}

var palette = [6]color.RGBA{
	cube.ColU: {245, 245, 245, 255}, // white
	cube.ColR: {200, 30, 30, 255},   // red
	cube.ColF: {30, 170, 60, 255},   // green
	cube.ColD: {235, 210, 40, 255},  // yellow
	cube.ColL: {235, 140, 30, 255},  // orange
	cube.ColB: {40, 90, 210, 255},   // blue
}

// Available reports whether the visualizer is compiled in.
func Available() bool { return true }

type game struct {
	c     cube.Cube
	moves []cube.Move
	idx   int
	frame int
}

func (g *game) Update() error {
	if g.idx >= len(g.moves) {
		return nil
	}
	g.frame++
	if g.frame >= framesPerMove {
		g.frame = 0
		g.c.Apply(g.moves[g.idx])
		g.idx++
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{24, 24, 28, 255})
	f := g.c.ToFacelets()
	for face := range 6 {
		ox := marginX + faceOrigin[face][0]*cell
		oy := marginY + faceOrigin[face][1]*cell
		for r := range 3 {
			for c := range 3 {
				col := palette[f[face*9+r*3+c]]
				x := float32(ox + c*cell)
				y := float32(oy + r*cell)
				vector.DrawFilledRect(screen, x+1, y+1, cell-2, cell-2, col, false)
				vector.StrokeRect(screen, x, y, cell, cell, 1, color.RGBA{20, 20, 20, 255}, false)
			}
		}
	}

	status := "solved"
	if g.idx < len(g.moves) {
		status = fmt.Sprintf("move %d/%d: %s", g.idx+1, len(g.moves), g.moves[g.idx])
	} else if len(g.moves) > 0 {
		status = fmt.Sprintf("done (%d moves)", len(g.moves))
	}
	ebitenutil.DebugPrintAt(screen, status, marginX, marginY+9*cell+10)
}

func (g *game) Layout(int, int) (int, int) {
	return marginX*2 + 12*cell, marginY*2 + 9*cell + 30
}

// Play opens the visualizer on the start cube and animates the moves.
func Play(start cube.Cube, moves []cube.Move) error {
	g := &game{c: start, moves: moves}
	ebiten.SetWindowSize(marginX*2+12*cell, marginY*2+9*cell+30)
	ebiten.SetWindowTitle("rubix")
	return ebiten.RunGame(g)
}
