// Package robot abstracts a physical cube manipulator (a LEGO Mindstorms EV3
// "MindCub3r"-style robot). The same headless cube pipeline drives it: Scan reads a
// real cube into a cube.Facelets, the solver produces moves, and Execute plays them
// back. A mock driver (default build) prints what a real robot would do; the ev3
// driver (//go:build ev3) talks to real motors and the colour sensor.
package robot

import "github.com/danielriddell21/rubix/internal/cube"

// Robot is a physical cube manipulator.
type Robot interface {
	// Scan reads the physical cube into the shared facelet model.
	Scan() (cube.Facelets, error)
	// Execute performs the given outer-face turns on the physical cube.
	Execute(moves []cube.Move) error
	// Home returns motors to a known rest position.
	Home() error
	// Calibrate calibrates the colour sensor / motor positions.
	Calibrate() error
	// Close releases any hardware resources.
	Close() error
}

// Physical face slots, fixed in space. The robot turns the layer at the Down slot
// and re-orients the cube with flips (tilt arm) and rotations (turntable).
const (
	slotU = iota
	slotR
	slotF
	slotD
	slotL
	slotB
)

// PrimKind is a low-level robot action.
type PrimKind int

const (
	// Flip tilts the whole cube forward 90°: Front→Down→Back→Up→Front.
	Flip PrimKind = iota
	// Rotate spins the whole cube on the turntable; Amount is quarter turns.
	Rotate
	// TurnBottom turns just the bottom layer; Amount is quarter turns.
	TurnBottom
)

// Primitive is one robot action.
type Primitive struct {
	Kind   PrimKind
	Amount int // quarter turns for Rotate/TurnBottom; ignored for Flip
}

func (p Primitive) String() string {
	switch p.Kind {
	case Flip:
		return "FLIP"
	case Rotate:
		return "ROTATE " + quarterString(p.Amount)
	default:
		return "TURN " + quarterString(p.Amount)
	}
}

func quarterString(n int) string {
	switch ((n % 4) + 4) % 4 {
	case 1:
		return "+90"
	case 2:
		return "180"
	case 3:
		return "-90"
	default:
		return "0"
	}
}

// planner tracks the cube's orientation as it is manipulated: orient[slot] is the
// logical face currently occupying that physical slot.
type planner struct {
	orient [6]int
	prims  []Primitive
}

func newPlanner() *planner {
	return &planner{orient: [6]int{slotU, slotR, slotF, slotD, slotL, slotB}}
}

func (p *planner) flip() {
	o := p.orient
	p.orient[slotD] = o[slotF]
	p.orient[slotB] = o[slotD]
	p.orient[slotU] = o[slotB]
	p.orient[slotF] = o[slotU]
	p.prims = append(p.prims, Primitive{Kind: Flip})
}

func (p *planner) rotate() {
	o := p.orient
	p.orient[slotR] = o[slotF]
	p.orient[slotB] = o[slotR]
	p.orient[slotL] = o[slotB]
	p.orient[slotF] = o[slotL]
	p.prims = append(p.prims, Primitive{Kind: Rotate, Amount: 1})
}

// bringDown re-orients so logical face f occupies the Down slot.
func (p *planner) bringDown(f int) {
	for p.orient[slotD] != f {
		switch p.orient[slotF] {
		case f:
			p.flip()
		default:
			if p.orient[slotU] == f || p.orient[slotB] == f {
				p.flip()
			} else {
				p.rotate() // f is on a side slot; bring it into the flip cycle
			}
		}
	}
}

// turnBottom turns the current bottom layer by the given quarter turns.
func (p *planner) turnBottom(quarters int) {
	p.prims = append(p.prims, Primitive{Kind: TurnBottom, Amount: quarters})
}

// PlanMoves lowers a sequence of outer-face turns into robot primitives, tracking the
// cube's orientation so each turn is applied to the bottom layer.
func PlanMoves(moves []cube.Move) []Primitive {
	p := newPlanner()
	for _, m := range moves {
		p.bringDown(m.Face())
		p.turnBottom(amountOf(m))
	}
	return p.prims
}

// amountOf converts a move's power to quarter turns: CW=1, 180=2, CCW=3.
func amountOf(m cube.Move) int {
	switch m % 3 {
	case 0:
		return 1
	case 1:
		return 2
	default:
		return 3
	}
}
