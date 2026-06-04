package robot

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/danielriddell21/rubix/internal/cube"
)

// Mock is the default, hardware-free driver. It prints the moves and the robot
// primitive plan a real machine would perform, and serves a preset facelet state for
// Scan. It lets the whole scan→solve→execute pipeline run on a PC.
type Mock struct {
	Out      io.Writer
	Scanned  cube.Facelets
	HasState bool
}

// NewMock returns a mock robot writing to stdout.
func NewMock() *Mock { return &Mock{Out: os.Stdout} }

// SetScan presets the facelets that Scan will report (e.g. from a --input string).
func (m *Mock) SetScan(f cube.Facelets) {
	m.Scanned = f
	m.HasState = true
}

func (m *Mock) Scan() (cube.Facelets, error) {
	if !m.HasState {
		return cube.Facelets{}, errors.New("mock robot has no cube to scan; pass --input with the facelet string")
	}
	return m.Scanned, nil
}

func (m *Mock) Execute(moves []cube.Move) error {
	prims := PlanMoves(moves)
	fmt.Fprintf(m.Out, "moves (%d): %s\n", len(moves), cube.FormatMoves(moves))
	fmt.Fprintf(m.Out, "robot plan (%d primitives):\n", len(prims))
	for _, p := range prims {
		fmt.Fprintf(m.Out, "  %s\n", p)
	}
	return nil
}

func (m *Mock) Home() error      { fmt.Fprintln(m.Out, "home"); return nil }
func (m *Mock) Calibrate() error { fmt.Fprintln(m.Out, "calibrate"); return nil }
func (m *Mock) Close() error     { return nil }
