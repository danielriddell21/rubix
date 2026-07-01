package cli

import (
	"fmt"
	"io"
	"math/rand/v2"
	"runtime"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/danielriddell21/rubix/internal/solver"
	"github.com/danielriddell21/rubix/pkg/cube"
)

type replicaJob struct {
	index    int
	seed     int64
	strategy string
	scramble cube.Cube
}

type replicaResult struct {
	job replicaJob
	res solver.Result
	err error
}

func replicaCmd(use string, compareDefault bool) *cobra.Command {
	var (
		count      int
		strategies string
		compare    bool
		seed       int64
		n          int
		fmtOpts    formatOpts
	)
	short := "Solve many cubes at once and compare"
	if use == "compare" {
		short = "Solve each scramble with every solver (replica --compare)"
	}
	cmd := &cobra.Command{
		Use:          use,
		Short:        short,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validFormat(fmtOpts.format); err != nil {
				return err
			}
			if count < 1 {
				return fmt.Errorf("--count must be at least 1")
			}

			names, err := resolveStrategies(strategies, compare)
			if err != nil {
				return err
			}

			// One scramble per cube index, one job per strategy (so a comparison row
			// solves the same scramble with every solver).
			jobs := make([]replicaJob, 0, count*len(names))
			for i := range count {
				s := seed + int64(i)
				if seed == 0 {
					s = int64(rand.Uint64())
				}
				scr := cube.ScrambledCube(n, s)
				for _, name := range names {
					jobs = append(jobs, replicaJob{index: i, seed: s, strategy: name, scramble: scr})
				}
			}

			if err := warmTables(); err != nil {
				return err
			}
			results := solveBatch(jobs)
			if fmtOpts.format == "text" {
				return withOutput(fmtOpts.output, func(w io.Writer) error {
					printReplicaTable(w, results)
					return nil
				})
			}
			return renderReplica(fmtOpts.format, fmtOpts.output, buildReplicaOutput(results))
		},
	}
	f := cmd.Flags()
	f.IntVar(&count, "count", 1, "number of cubes (distinct scrambles)")
	f.StringVar(&strategies, "strategies", "", `comma-separated solvers, or "all" (default: prune, or all with --compare)`)
	f.BoolVar(&compare, "compare", compareDefault, "solve each scramble with every solver")
	f.Int64Var(&seed, "seed", 0, "base seed; cube i uses seed+i (0 = random each run)")
	f.IntVar(&n, "n", 25, "scramble length (moves)")
	fmtOpts.register(f)
	return cmd
}

func buildReplicaOutput(results []replicaResult) replicaOutput {
	out := replicaOutput{Results: make([]replicaRow, 0, len(results))}
	solved, totalMoves := 0, 0
	for _, r := range results {
		j := r.job
		row := replicaRow{Index: j.index, Strategy: j.strategy, Seed: j.seed}
		if r.err != nil {
			row.Error = r.err.Error()
			out.Results = append(out.Results, row)
			continue
		}
		row.MoveCount = len(r.res.Moves)
		row.Solved = r.res.Solved
		row.TimeMS = r.res.Elapsed.Milliseconds()
		row.Nodes = r.res.Nodes
		if r.res.Solved {
			solved++
			totalMoves += len(r.res.Moves)
		}
		out.Results = append(out.Results, row)
	}
	out.Summary = replicaSummary{Solved: solved, Total: len(results)}
	if solved > 0 {
		out.Summary.AvgMoves = float64(totalMoves) / float64(solved)
	}
	return out
}

func resolveStrategies(list string, compare bool) ([]string, error) {
	if list == "" {
		if compare {
			return solver.Names(), nil
		}
		return []string{"prune"}, nil
	}
	if list == "all" {
		return solver.Names(), nil
	}
	var names []string
	for _, raw := range strings.Split(list, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if _, err := solver.Get(name); err != nil {
			return nil, fmt.Errorf("get solver %q: %w", name, err)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no strategies given")
	}
	return names, nil
}

func solveBatch(jobs []replicaJob) []replicaResult {
	out := make([]replicaResult, len(jobs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	for i, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, j replicaJob) {
			defer wg.Done()
			defer func() { <-sem }()
			s, err := solver.Get(j.strategy)
			if err != nil {
				out[i] = replicaResult{job: j, err: err}
				return
			}
			res, err := s.Solve(j.scramble)
			out[i] = replicaResult{job: j, res: res, err: err}
		}(i, j)
	}
	wg.Wait()
	return out
}

func printReplicaTable(w io.Writer, results []replicaResult) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "#\tstrategy\tseed\tmoves\tsolved\ttime\tnodes")
	solved, totalMoves := 0, 0
	for _, r := range results {
		j := r.job
		if r.err != nil {
			fmt.Fprintf(tw, "%d\t%s\t%d\t-\tERR: %v\t-\t-\n", j.index, j.strategy, j.seed, r.err)
			continue
		}
		mark := "no"
		if r.res.Solved {
			mark = "yes"
			solved++
			totalMoves += len(r.res.Moves)
		}
		fmt.Fprintf(tw, "%d\t%s\t%d\t%d\t%s\t%s\t%d\n",
			j.index, j.strategy, j.seed, len(r.res.Moves), mark,
			r.res.Elapsed.Round(time.Millisecond), r.res.Nodes)
	}
	_ = tw.Flush()

	fmt.Fprintf(w, "\n%d/%d solved", solved, len(results))
	if solved > 0 {
		fmt.Fprintf(w, "   avg moves %.1f", float64(totalMoves)/float64(solved))
	}
	fmt.Fprintln(w)
}
