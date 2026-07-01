package robot

import "github.com/danielriddell21/rubix/pkg/cube"

type Robot interface {
	Scan() (cube.Facelets, error)

	Execute(moves []cube.Move) error

	Home() error

	Calibrate() error

	Close() error
}

const (
	slotU = iota
	slotR
	slotF
	slotD
	slotL
	slotB
)

type PrimKind int

const (
	Flip PrimKind = iota

	Rotate

	TurnBottom
)

type Primitive struct {
	Kind   PrimKind
	Amount int
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

func (p *planner) turnBottom(quarters int) {
	p.prims = append(p.prims, Primitive{Kind: TurnBottom, Amount: quarters})
}

func PlanMoves(moves []cube.Move) []Primitive {
	p := newPlanner()
	for _, m := range moves {
		p.bringDown(m.Face())
		p.turnBottom(amountOf(m))
	}
	return p.prims
}

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
