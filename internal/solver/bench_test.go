package solver

import (
	"testing"

	"github.com/danielriddell21/rubix/pkg/cube"
)

func BenchmarkSolve(b *testing.B) {
	scramble := cube.ScrambledCube(25, 42)
	if warm, err := Get("prune"); err == nil {
		_, _ = warm.Solve(cube.ScrambledCube(20, 1)) // build & cache prune tables once
	}
	for _, name := range []string{"cfop", "prune", "multi"} {
		s, err := Get(name)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := s.Solve(scramble); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
