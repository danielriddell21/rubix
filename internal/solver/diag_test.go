package solver

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/danielriddell21/rubix/internal/cube"
)

// TestDiagTiming reports each solver's solve rate, average move count and time over a
// batch of full random scrambles. Gated behind DIAG so it does not run normally.
func TestDiagTiming(t *testing.T) {
	if os.Getenv("DIAG") == "" {
		t.Skip("set DIAG=1")
	}
	const n = 20
	for _, s := range All() {
		var solved, moves int
		start := time.Now()
		for seed := int64(0); seed < n; seed++ {
			c := cube.ScrambledCube(25, seed)
			res, err := s.Solve(c)
			if err != nil {
				t.Fatalf("%s: %v", s.Name(), err)
			}
			if res.Solved {
				solved++
				moves += len(res.Moves)
			}
		}
		avg := 0.0
		if solved > 0 {
			avg = float64(moves) / float64(solved)
		}
		fmt.Fprintf(os.Stderr, "%-9s rate=%3d%% avgMoves=%.1f totalTime=%s\n",
			s.Name(), solved*100/n, avg, time.Since(start).Round(time.Millisecond))
	}
}
