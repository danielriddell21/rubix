package solver

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/danielriddell21/rubix/pkg/cube"
)

func TestDiagTiming(t *testing.T) {
	if os.Getenv("DIAG") == "" {
		t.Skip("set DIAG=1")
	}
	n := 20
	if v := os.Getenv("RATE_N"); v != "" {
		n, _ = strconv.Atoi(v)
	}
	// By default iterate fast→slow (registry order reversed) so results stream
	// quickly; RATE_SOLVERS=a,b,c restricts/orders the set measured.
	var list []Solver
	if names := os.Getenv("RATE_SOLVERS"); names != "" {
		for _, name := range strings.Split(names, ",") {
			s, err := Get(strings.TrimSpace(name))
			if err != nil {
				t.Fatal(err)
			}
			list = append(list, s)
		}
	} else {
		all := All()
		for i := len(all) - 1; i >= 0; i-- {
			list = append(list, all[i])
		}
	}
	for _, s := range list {
		var solved, moves int
		start := time.Now()
		for seed := int64(0); seed < int64(n); seed++ {
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
