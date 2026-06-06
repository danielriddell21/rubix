package solver

import "github.com/danielriddell21/rubix/pkg/cube"

// stepLimit bounds how many moves a greedy descent will play before declaring failure.
const stepLimit = 300

// greedySolver is the first solver from Sebastian Lague's video: it repeatedly searches
// a few moves ahead and plays the move leading to the highest-scoring position, where a
// position's score is the number of cubies in their solved place and orientation. It is
// fast but frequently strands the last few pieces — the video solves only ~10%.
type greedySolver struct{}

func (greedySolver) Name() string     { return "greedy" }
func (greedySolver) Describe() string { return "greedy best-first by solved-cubie count" }

func (greedySolver) Solve(c cube.Cube) (Result, error) {
	return timed("greedy", c, func(c cube.Cube) ([]cube.Move, bool, uint64, error) {
		moves, nodes := descend(pack(c), state.solvedCount, state.isSolved, noTerminal, allMoves, 6, 7, stepLimit)
		return moves, c.Applied(moves...).IsSolved(), nodes, nil
	})
}
