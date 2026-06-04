package solver

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/danielriddell21/rubix/internal/cube"
	"github.com/danielriddell21/rubix/internal/solver/search"
)

func TestDiagTiming(t *testing.T) {
	if os.Getenv("DIAG") == "" {
		t.Skip("set DIAG=1")
	}
	_ = search.AllMoves
	for _, name := range []string{"cfop", "domino", "multi"} {
		s, _ := Get(name)
		for seed := int64(0); seed < 3; seed++ {
			c := cube.ScrambledCube(25, seed)
			start := time.Now()
			res, err := s.Solve(c)
			fmt.Fprintf(os.Stderr, "%-8s seed%d err=%v len=%d nodes=%d elapsed=%s\n",
				name, seed, err, len(res.Moves), res.Nodes, time.Since(start))
		}
	}
}
