package solver

import (
	"github.com/danielriddell21/rubix/internal/solver/search"
	"github.com/danielriddell21/rubix/pkg/cube"
)

var allMoves = search.AllMoves

var orientedMoves = []cube.Move{
	cube.U, cube.U2, cube.Up,
	cube.D, cube.D2, cube.Dp,
	cube.L, cube.L2, cube.Lp,
	cube.R, cube.R2, cube.Rp,
}

var dominoMoves = search.DominoMoves
