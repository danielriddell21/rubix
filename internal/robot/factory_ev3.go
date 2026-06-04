//go:build ev3

package robot

// New returns the ev3dev-backed robot driver. Built only with the `ev3` tag, for an
// ev3dev brick.
func New() (Robot, error) { return newEV3() }
