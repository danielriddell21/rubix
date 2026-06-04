package solver

import (
	"fmt"

	"github.com/danielriddell21/rubix/internal/cube"
)

// multiSolver runs several strong solvers concurrently and returns the shortest
// solution any of them finds (video 2: "Multi-Search").
type multiSolver struct{}

func (multiSolver) Name() string { return "multi" }
func (multiSolver) Describe() string {
	return "multi-search: run several solvers concurrently, keep the shortest"
}

func (multiSolver) Solve(c cube.Cube) (Result, error) {
	return timed("multi", c, func(c cube.Cube) ([]cube.Move, uint64, error) {
		type result struct {
			moves []cube.Move
			nodes uint64
		}
		searches := []func(cube.Cube) ([]cube.Move, uint64, error){
			func(c cube.Cube) ([]cube.Move, uint64, error) { return twoPhase(c, 20) },
			orientThenSolve,
			cfopSolve,
		}
		ch := make(chan result, len(searches))
		for _, s := range searches {
			go func(f func(cube.Cube) ([]cube.Move, uint64, error)) {
				moves, nodes, err := f(c)
				if err != nil {
					moves = nil
				}
				ch <- result{moves, nodes}
			}(s)
		}
		var best []cube.Move
		var nodes uint64
		bestLen := 1 << 30
		for range searches {
			r := <-ch
			nodes += r.nodes
			if r.moves != nil && len(r.moves) < bestLen {
				best, bestLen = r.moves, len(r.moves)
			}
		}
		if best == nil {
			return nil, nodes, fmt.Errorf("multi: no solver returned a solution")
		}
		return best, nodes, nil
	})
}
