package cli

import (
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/danielriddell21/rubix/internal/gui"
	"github.com/danielriddell21/rubix/internal/robot"
	"github.com/danielriddell21/rubix/internal/solver"
	"github.com/danielriddell21/rubix/pkg/cube"
)

func Run(version string, args []string) int {
	root := newRoot(version)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func newRoot(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "rubix",
		Short: "A Rubik's cube solver",
		Long: `rubix — a Rubik's cube solver.

A headless core models the cube as data and solves it with the methods from two
"Coding Adventure" videos; it can also drive an Ebiten visualizer or a LEGO EV3
robot.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		solveCmd(),
		replicaCmd("replica", false),
		replicaCmd("compare", true),
		solversCmd(),
		scrambleCmd(),
		verifyCmd(),
		scanCmd(),
		genTablesCmd(),
		viewCmd(),
		completionCmd(),
	)
	return root
}

func cubeFromInput(input string, scan bool, r robot.Robot) (cube.Cube, error) {
	switch {
	case scan:
		f, err := r.Scan()
		if err != nil {
			return cube.Cube{}, fmt.Errorf("scan cube: %w", err)
		}
		c, err := cube.FromFacelets(f)
		if err != nil {
			return cube.Cube{}, fmt.Errorf("build cube: %w", err)
		}
		return c, nil
	case input == "@-":
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return cube.Cube{}, fmt.Errorf("read stdin: %w", err)
		}
		return parseCube(string(data))
	case input != "":
		return parseCube(input)
	default:
		return cube.Cube{}, fmt.Errorf("provide --input <54 facelets> or --scan")
	}
}

func parseCube(s string) (cube.Cube, error) {
	f, err := cube.ParseFacelets(strings.TrimSpace(s))
	if err != nil {
		return cube.Cube{}, fmt.Errorf("parse facelets: %w", err)
	}
	c, err := cube.FromFacelets(f)
	if err != nil {
		return cube.Cube{}, fmt.Errorf("build cube: %w", err)
	}
	return c, nil
}

type solveOpts struct {
	input    string
	scan     bool
	strategy string
	execute  bool
	view     bool
	fmtOpts  formatOpts
}

func solveCmd() *cobra.Command {
	var o solveOpts
	cmd := &cobra.Command{
		Use:          "solve",
		Short:        "Solve a cube (from a facelet string or the robot)",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSolve(o)
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.input, "input", "", "54-character facelet string (URFDLB), or @- for stdin")
	f.BoolVar(&o.scan, "scan", false, "scan the cube with the robot instead of --input")
	f.StringVar(&o.strategy, "strategy", "prune", "solver: "+strings.Join(solver.Names(), ", "))
	f.BoolVar(&o.execute, "execute", false, "run the solution on the robot")
	f.BoolVar(&o.view, "view", false, "animate the solution in the visualizer (needs -tags ebiten)")
	o.fmtOpts.register(f)
	return cmd
}

func runSolve(o solveOpts) error {
	if err := validFormat(o.fmtOpts.format); err != nil {
		return err
	}
	s, err := solver.Get(o.strategy)
	if err != nil {
		return fmt.Errorf("get solver: %w", err)
	}
	r, err := robot.New()
	if err != nil {
		return fmt.Errorf("init robot: %w", err)
	}
	defer func() { _ = r.Close() }()

	c, err := cubeFromInput(o.input, o.scan, r)
	if err != nil {
		return err
	}
	res, err := s.Solve(c)
	if err != nil {
		return fmt.Errorf("solve: %w", err)
	}
	out := solveOutput{
		Strategy:  res.Strategy,
		Solved:    res.Solved,
		Moves:     cube.FormatMoves(res.Moves),
		MoveList:  moveStrings(res.Moves),
		MoveCount: len(res.Moves),
		Nodes:     res.Nodes,
		ElapsedMS: res.Elapsed.Milliseconds(),
	}
	if err := renderSolve(o.fmtOpts.format, o.fmtOpts.output, out); err != nil {
		return err
	}
	if !res.Solved {
		return nil
	}
	return finishSolve(o, r, c, res.Moves)
}

func finishSolve(o solveOpts, r robot.Robot, c cube.Cube, moves []cube.Move) error {
	if o.execute {
		if err := r.Execute(moves); err != nil {
			return fmt.Errorf("execute moves: %w", err)
		}
	}
	if o.view {
		// SA4023: without -tags ebiten the stub Run always errors, so staticcheck
		// reads this as constant. It is not, in the build that has a GUI.
		if err := gui.Run(gui.Config{Controller: guiController(&c, strategyIndex(o.strategy), 0)}); err != nil { //nolint:staticcheck
			return fmt.Errorf("run gui: %w", err)
		}
	}
	return nil
}

func guiController(initial *cube.Cube, start int, seed int64) gui.Controller {
	nextSeed := rand.Int64
	if seed != 0 {
		rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed)))
		nextSeed = func() int64 { return int64(rng.Uint64()) }
	}
	return gui.Controller{
		Strategies: solver.Names(),
		Start:      start,
		Initial:    initial,
		Scramble:   func() cube.Cube { return cube.ScrambledCube(25, nextSeed()) },
		Solve: func(name string, c cube.Cube) ([]cube.Move, bool) {
			s, err := solver.Get(name)
			if err != nil {
				return nil, false
			}
			res, err := s.Solve(c)
			if err != nil {
				return nil, false
			}
			// Return the attempt even when it doesn't solve, so the viewer plays it
			// and shows it getting stuck.
			return res.Moves, res.Solved
		},
	}
}

func strategyIndex(name string) int {
	for i, n := range solver.Names() {
		if n == name {
			return i
		}
	}
	return 0
}

func solversCmd() *cobra.Command {
	var fmtOpts formatOpts
	cmd := &cobra.Command{
		Use:          "solvers",
		Short:        "List the available solvers in video order",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validFormat(fmtOpts.format); err != nil {
				return err
			}
			list := make([]solverInfo, 0, len(solver.All()))
			for _, s := range solver.All() {
				list = append(list, solverInfo{Name: s.Name(), Describe: s.Describe()})
			}
			return renderSolvers(fmtOpts.format, fmtOpts.output, list)
		},
	}
	fmtOpts.register(cmd.Flags())
	return cmd
}

func scrambleCmd() *cobra.Command {
	var (
		n       int
		seed    int64
		fmtOpts formatOpts
	)
	cmd := &cobra.Command{
		Use:          "scramble",
		Short:        "Generate a scramble and its facelet string",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validFormat(fmtOpts.format); err != nil {
				return err
			}
			moves := cube.Scramble(n, seed)
			c := cube.Solved().Applied(moves...)
			return renderScramble(fmtOpts.format, fmtOpts.output, scrambleOutput{
				N:        n,
				Seed:     seed,
				Scramble: cube.FormatMoves(moves),
				Facelets: c.ToFacelets().String(),
			})
		},
	}
	f := cmd.Flags()
	f.IntVar(&n, "n", 25, "number of random moves")
	f.Int64Var(&seed, "seed", 0, "random seed")
	fmtOpts.register(f)
	return cmd
}

func verifyCmd() *cobra.Command {
	var input string
	cmd := &cobra.Command{
		Use:          "verify",
		Short:        "Validate a facelet string",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := cubeFromInput(input, false, nil)
			if err != nil {
				return err
			}
			solved := ""
			if c.IsSolved() {
				solved = " (already solved)"
			}
			fmt.Printf("valid cube%s\n", solved)
			return nil
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "54-character facelet string, or @- for stdin")
	return cmd
}

func scanCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "scan",
		Short:        "Scan a cube with the robot and print its facelets",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := robot.New()
			if err != nil {
				return fmt.Errorf("init robot: %w", err)
			}
			defer func() { _ = r.Close() }()
			f, err := r.Scan()
			if err != nil {
				return fmt.Errorf("scan cube: %w", err)
			}
			fmt.Println(f)
			return nil
		},
	}
}

func genTablesCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "gen-tables",
		Short:        "Precompute and cache the prune tables",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("building prune tables...")
			if err := warmTables(); err != nil {
				return err
			}
			fmt.Println("done")
			return nil
		},
	}
}

func warmTables() error {
	s, _ := solver.Get("prune")
	if _, err := s.Solve(cube.ScrambledCube(20, 1)); err != nil {
		return fmt.Errorf("warm tables: %w", err)
	}
	return nil
}

func viewCmd() *cobra.Command {
	var (
		seed  int64
		child int
	)
	cmd := &cobra.Command{
		Use:          "view",
		Short:        "Open the self-driving visualizer (needs -tags ebiten)",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// The visualizer is self-driving: it scrambles and solves on its own; press "r"
			// for a new scramble, "s" to switch solver, and "+"/"-" to open or close more
			// cube windows.
			ctrl := guiController(nil, strategyIndex("multi"), seed)
			switch {
			case child >= 0:
				ctrl.Title = fmt.Sprintf("rubix #%d", child)
				ctrl.OffsetIndex = child
				return runChild(ctrl)
			default:
				return runLeader(ctrl)
			}
		},
	}
	f := cmd.Flags()
	f.Int64Var(&seed, "seed", 0, "scramble seed (0 = random each run); set to replay the same scrambles")
	f.IntVar(&child, "child", -1, "internal: run as a coordinated child window with this index")
	return cmd
}

// DemoController builds the visualizer controller the documentation clips
// record: the self-driving multi-solver view, scrambling from a fixed seed so
// every run replays identically. tools/demogen is its only caller.
func DemoController(seed int64) gui.Controller {
	return guiController(nil, strategyIndex("multi"), seed)
}
