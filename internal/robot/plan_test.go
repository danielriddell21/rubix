package robot

import (
	"testing"

	"github.com/danielriddell21/rubix/pkg/cube"
)

// recover replays robot primitives and reconstructs the outer-face turns they
// perform, mirroring the planner's orientation bookkeeping.
func recoverMoves(prims []Primitive) []cube.Move {
	orient := [6]int{slotU, slotR, slotF, slotD, slotL, slotB}
	flip := func() {
		o := orient
		orient[slotD] = o[slotF]
		orient[slotB] = o[slotD]
		orient[slotU] = o[slotB]
		orient[slotF] = o[slotU]
	}
	rotate := func() {
		o := orient
		orient[slotR] = o[slotF]
		orient[slotB] = o[slotR]
		orient[slotL] = o[slotB]
		orient[slotF] = o[slotL]
	}
	var moves []cube.Move
	for _, p := range prims {
		switch p.Kind {
		case Flip:
			flip()
		case Rotate:
			for range p.Amount {
				rotate()
			}
		case TurnBottom:
			power := []cube.Move{0, 0, 1, 2}[p.Amount] // amount 1->CW,2->180,3->CCW
			moves = append(moves, cube.Move(orient[slotD]*3)+power)
		}
	}
	return moves
}

func TestPlanMovesReproducesMoves(t *testing.T) {
	for seed := int64(0); seed < 50; seed++ {
		moves := cube.Scramble(25, seed)
		prims := PlanMoves(moves)
		got := recoverMoves(prims)
		if len(got) != len(moves) {
			t.Fatalf("seed %d: recovered %d moves, want %d", seed, len(got), len(moves))
		}
		for i := range moves {
			if got[i] != moves[i] {
				t.Fatalf("seed %d move %d: recovered %s want %s", seed, i, got[i], moves[i])
			}
		}
	}
}

func TestPlanMovesEffectMatches(t *testing.T) {
	// The planned primitives, interpreted as cube turns, must reach the same state.
	for seed := int64(0); seed < 50; seed++ {
		moves := cube.Scramble(20, seed)
		want := cube.Solved().Applied(moves...)
		got := cube.Solved().Applied(recoverMoves(PlanMoves(moves))...)
		if got != want {
			t.Fatalf("seed %d: planned execution diverged from move sequence", seed)
		}
	}
}
