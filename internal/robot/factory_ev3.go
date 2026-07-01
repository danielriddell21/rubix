//go:build ev3

package robot

func New() (Robot, error) { return newEV3() }
