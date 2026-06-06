//go:build ev3

package robot

import (
	"fmt"
	"time"

	"github.com/ev3go/ev3dev"

	"github.com/danielriddell21/rubix/pkg/cube"
)

// ev3.go drives a real MindCub3r-style robot through the ev3dev sysfs interface: a
// turntable motor spins/holds the cube, a tilt-arm motor flips it and holds the top
// layers while the turntable twists the bottom, and a colour sensor on a swing arm
// scans the faces. Gear ratios and arm positions below are the standard MindCub3r
// build; adjust them with Calibrate for a different rig.
//
// This file only compiles with the `ev3` tag and is meant to run on an ev3dev brick;
// it is exercised here through cross-compilation rather than on hardware.

const (
	basePort  = "ev3-ports:outA" // turntable
	armPort   = "ev3-ports:outB" // tilt arm
	colorPort = "ev3-ports:in1"  // colour sensor

	lMotor = "lego-ev3-l-motor"
	mMotor = "lego-ev3-m-motor"
	colorS = "lego-ev3-color"

	// Turntable gearing: motor degrees for a 90° cube rotation, with a little extra
	// to take up backlash on each quarter turn.
	baseQuarter  = 270
	baseBacklash = 18

	// Tilt-arm motor positions (degrees from rest).
	armHold = 110 // lower onto the cube to hold the upper layers
	armFlip = 200 // push further to tip the cube forward

	baseSpeed = 600
	armSpeed  = 700
)

// EV3 is the ev3dev-backed robot driver.
type EV3 struct {
	base  *ev3dev.TachoMotor
	arm   *ev3dev.TachoMotor
	color *ev3dev.Sensor
}

func newEV3() (*EV3, error) {
	base, err := ev3dev.TachoMotorFor(basePort, lMotor)
	if err != nil {
		return nil, fmt.Errorf("turntable motor on %s: %w", basePort, err)
	}
	arm, err := ev3dev.TachoMotorFor(armPort, mMotor)
	if err != nil {
		return nil, fmt.Errorf("arm motor on %s: %w", armPort, err)
	}
	color, err := ev3dev.SensorFor(colorPort, colorS)
	if err != nil {
		return nil, fmt.Errorf("colour sensor on %s: %w", colorPort, err)
	}
	color.SetMode("RGB-RAW")
	e := &EV3{base: base, arm: arm, color: color}
	return e, e.Home()
}

// run drives a motor a relative number of degrees and waits for it to stop.
func run(m *ev3dev.TachoMotor, degrees, speed int) error {
	m.SetSpeedSetpoint(speed).SetPositionSetpoint(degrees).SetStopAction("hold").Command("run-to-rel-pos")
	if err := m.Err(); err != nil {
		return err
	}
	_, _, err := ev3dev.Wait(m, ev3dev.Running, 0, 0, false, 15*time.Second)
	return err
}

// armTo moves the tilt arm to an absolute position (0 = rest).
func (e *EV3) armTo(pos int) error {
	cur, err := e.arm.Position()
	if err != nil {
		return err
	}
	return run(e.arm, pos-cur, armSpeed)
}

func (e *EV3) flip() error {
	if err := e.armTo(armFlip); err != nil {
		return err
	}
	return e.armTo(0)
}

func (e *EV3) rotate(quarters int) error {
	return e.spin(quarters, false)
}

// spin turns the turntable. When holding, the arm pins the upper layers so only the
// bottom layer turns; backlash is taken up by over-rotating and easing back.
func (e *EV3) spin(quarters int, hold bool) error {
	if hold {
		if err := e.armTo(armHold); err != nil {
			return err
		}
	}
	deg := quarters * baseQuarter
	if deg > 0 {
		deg += baseBacklash
	} else if deg < 0 {
		deg -= baseBacklash
	}
	if err := run(e.base, deg, baseSpeed); err != nil {
		return err
	}
	if quarters > 0 {
		if err := run(e.base, -baseBacklash, baseSpeed); err != nil {
			return err
		}
	} else if quarters < 0 {
		if err := run(e.base, baseBacklash, baseSpeed); err != nil {
			return err
		}
	}
	if hold {
		return e.armTo(0)
	}
	return nil
}

func (e *EV3) turnBottom(quarters int) error { return e.spin(quarters, true) }

// Execute lowers the moves to primitives and drives the motors.
func (e *EV3) Execute(moves []cube.Move) error {
	for _, p := range PlanMoves(moves) {
		var err error
		switch p.Kind {
		case Flip:
			err = e.flip()
		case Rotate:
			err = e.rotate(p.Amount)
		case TurnBottom:
			err = e.turnBottom(p.Amount)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *EV3) Home() error {
	e.base.SetStopAction("hold").Command("stop")
	return e.armTo(0)
}

func (e *EV3) Close() error {
	e.base.Command("stop")
	e.arm.Command("stop")
	return nil
}
