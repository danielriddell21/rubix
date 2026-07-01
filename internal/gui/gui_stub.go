//go:build !ebiten

package gui

import "errors"

func Available() bool { return false }

func Run(cfg Config) error {
	return errors.New("built without the GUI; rebuild with -tags ebiten")
}
