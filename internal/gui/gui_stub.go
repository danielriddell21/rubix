//go:build !ebiten

// Package gui is an optional Ebiten visualizer of the live cube state. Without the
// ebiten build tag it is a stub, so default and EV3 builds carry no GUI dependency.
package gui

import (
	"errors"

	"github.com/danielriddell21/rubix/internal/cube"
)

// Available reports whether the visualizer was compiled in.
func Available() bool { return false }

// Play would animate the solution from the given start cube. In the stub build it
// returns an error directing the user to rebuild with the ebiten tag.
func Play(start cube.Cube, moves []cube.Move, resolve func() (cube.Cube, []cube.Move)) error {
	return errors.New("visualizer not built into this binary; rebuild with: go build -tags ebiten")
}
