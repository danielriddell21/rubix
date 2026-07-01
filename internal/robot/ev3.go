//go:build ev3

package robot

import (
	"fmt"
	"time"

	"github.com/ev3go/ev3dev"

	"github.com/danielriddell21/rubix/pkg/cube"
)

const (
	basePort  = "ev3-ports:outA"
	armPort   = "ev3-ports:outB"
	colorPort = "ev3-ports:in1"

	lMotor = "lego-ev3-l-motor"
	mMotor = "lego-ev3-m-motor"
	colorS = "lego-ev3-color"

	baseQuarter  = 270
	baseBacklash = 18

	armHold = 110
	armFlip = 200

	baseSpeed = 600
	armSpeed  = 700
)

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

func run(m *ev3dev.TachoMotor, degrees, speed int) error {
	m.SetSpeedSetpoint(speed).SetPositionSetpoint(degrees).SetStopAction("hold").Command("run-to-rel-pos")
	if err := m.Err(); err != nil {
		return err
	}
	_, _, err := ev3dev.Wait(m, ev3dev.Running, 0, 0, false, 15*time.Second)
	return err
}

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
