//go:build !ev3

package robot

// New returns the default robot driver. Without the ev3 build tag this is the mock
// driver, so the tool runs anywhere.
func New() (Robot, error) { return NewMock(), nil }
