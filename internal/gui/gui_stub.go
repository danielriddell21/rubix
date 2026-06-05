//go:build !ebiten

// Package gui is an optional Ebiten visualizer of the live cube state. Without the
// ebiten build tag it is a stub, so default and EV3 builds carry no GUI dependency.
package gui

import "errors"

// Available reports whether the visualizer was compiled in.
func Available() bool { return false }

// Play would open the visualizer. In the stub build it returns an error directing the
// user to rebuild with the ebiten tag.
func Play(ctrl Controller) error {
	return errors.New("visualizer not built into this binary; rebuild with: go build -tags ebiten")
}
