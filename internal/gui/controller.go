package gui

import "github.com/danielriddell21/rubix/pkg/cube"

type Controller struct {
	Strategies []string
	Start      int
	Initial    *cube.Cube
	Scramble   func() cube.Cube

	Solve func(strategy string, c cube.Cube) (moves []cube.Move, solved bool)

	Record       string
	RecordFrames int
	RecordFPS    int
	RecordScale  int
	RecordKeys   string

	Title       string
	OffsetIndex int
}

type Cell struct {
	Strategy string
	Initial  cube.Cube
	Label    string
}
