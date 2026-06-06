package solver

import (
	"github.com/danielriddell21/rubix/internal/solver/search"
	"github.com/danielriddell21/rubix/pkg/cube"
)

// allMoves is the full 18-move set.
var allMoves = search.AllMoves

// orientedMoves is the 12-move set used by the oriented solver's second phase: once
// every edge is correctly oriented, front and back turns are no longer needed, so
// they are dropped entirely (video 1). This restriction is what lets the search go
// deeper — and, faithfully, why the oriented solver is not quite 100%.
var orientedMoves = []cube.Move{
	cube.U, cube.U2, cube.Up,
	cube.D, cube.D2, cube.Dp,
	cube.L, cube.L2, cube.Lp,
	cube.R, cube.R2, cube.Rp,
}

// dominoMoves is the domino move set ⟨U,D,F2,B2,L2,R2⟩: up/down quarter turns plus only
// half turns of the other four faces, which preserve the domino state.
var dominoMoves = search.DominoMoves
