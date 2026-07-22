package gui

import (
	"github.com/danielriddell21/crucible/record"

	"github.com/danielriddell21/rubix/pkg/cube"
)

type Controller struct {
	Strategies []string
	Start      int
	Initial    *cube.Cube
	Scramble   func() cube.Cube

	Solve func(strategy string, c cube.Cube) (moves []cube.Move, solved bool)

	// Rec holds the shared --record flags; RecordKeys is rubix's own
	// scripted-keybind extension layered on top.
	Rec        record.Options
	RecordKeys string

	Title       string
	OffsetIndex int
}

type Cell struct {
	Strategy string
	Initial  cube.Cube
	Label    string
}
