// Package cli implements the rubix command-line interface: a small hand-rolled
// subcommand dispatcher over the shared cube/solver/robot/gui packages.
package cli

import (
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strings"

	"github.com/danielriddell21/rubix/internal/gui"
	"github.com/danielriddell21/rubix/internal/robot"
	"github.com/danielriddell21/rubix/internal/solver"
	"github.com/danielriddell21/rubix/pkg/cube"
)

// Run dispatches a subcommand. version is the build version reported by the version
// command. It returns a process exit code.
func Run(version string, args []string) int {
	if len(args) < 1 {
		usage(os.Stderr)
		return 2
	}
	cmd, rest := args[0], args[1:]
	var err error
	switch cmd {
	case "version", "--version", "-v":
		fmt.Println("rubix", version)
		return 0
	case "solve":
		err = cmdSolve(rest)
	case "replica":
		err = cmdReplica(rest, false)
	case "compare":
		err = cmdReplica(rest, true)
	case "solvers":
		err = cmdSolvers(rest)
	case "scramble":
		err = cmdScramble(rest)
	case "verify":
		err = cmdVerify(rest)
	case "scan":
		err = cmdScan(rest)
	case "gen-tables":
		err = cmdGenTables(rest)
	case "view":
		err = cmdView(rest)
	case "help", "-h", "--help":
		usage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage(os.Stderr)
		return 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func usage(w io.Writer) {
	fmt.Fprint(w, `rubix — a Rubik's cube solver

usage: rubix <command> [flags]

commands:
  solve       solve a cube (from a facelet string or the robot)
  replica     solve many cubes at once and compare (alias: compare)
  solvers     list the available solvers in video order
  scramble    generate a scramble and its facelet string
  verify      validate a facelet string
  scan        scan a cube with the robot and print its facelets
  gen-tables  precompute and cache the prune tables
  view        open the self-driving visualizer (needs -tags ebiten)
  version     print the rubix version

run "rubix <command> -h" for command flags.
`)
}

// cubeFromInput builds a cube from a facelet string, "@-" (stdin), or the robot scan.
func cubeFromInput(input string, scan bool, r robot.Robot) (cube.Cube, error) {
	switch {
	case scan:
		f, err := r.Scan()
		if err != nil {
			return cube.Cube{}, err
		}
		return cube.FromFacelets(f)
	case input == "@-":
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return cube.Cube{}, err
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
		return cube.Cube{}, err
	}
	return cube.FromFacelets(f)
}

func cmdSolve(args []string) error {
	fs := flag.NewFlagSet("solve", flag.ContinueOnError)
	input := fs.String("input", "", "54-character facelet string (URFDLB), or @- for stdin")
	scan := fs.Bool("scan", false, "scan the cube with the robot instead of --input")
	strategy := fs.String("strategy", "prune", "solver: "+strings.Join(solver.Names(), ", "))
	execute := fs.Bool("execute", false, "run the solution on the robot")
	view := fs.Bool("view", false, "animate the solution in the visualizer (needs -tags ebiten)")
	rec := addRecordFlags(fs)
	format, output := addFormatFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validFormat(*format); err != nil {
		return err
	}

	s, err := solver.Get(*strategy)
	if err != nil {
		return err
	}
	r, err := robot.New()
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	c, err := cubeFromInput(*input, *scan, r)
	if err != nil {
		return err
	}

	res, err := s.Solve(c)
	if err != nil {
		return err
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
	if err := renderSolve(*format, *output, out); err != nil {
		return err
	}
	if !res.Solved {
		return nil
	}

	if *execute {
		if err := r.Execute(res.Moves); err != nil {
			return err
		}
	}
	if *view {
		return gui.Play(rec.apply(guiController(&c, strategyIndex(*strategy), 0)))
	}
	return nil
}

// recordOpts holds the visualizer recording flags shared across the GUI-launching
// commands. The CLI uses stdlib flag, which has no persistent flags, so addRecordFlags
// registers them on each command's flag set.
type recordOpts struct {
	path   string
	frames int
	fps    int
	scale  int
	keys   string
}

// addRecordFlags registers the --record* flags on fs and returns the destination opts.
func addRecordFlags(fs *flag.FlagSet) *recordOpts {
	o := &recordOpts{}
	fs.StringVar(&o.path, "record", "", "record the visualizer to this GIF path, then exit (needs -tags ebiten)")
	fs.IntVar(&o.frames, "record-frames", 120, "number of frames to capture when recording")
	fs.IntVar(&o.fps, "record-fps", 25, "GIF playback frames per second")
	fs.IntVar(&o.scale, "record-scale", 2, "integer downscale factor for the recorded GIF")
	fs.StringVar(&o.keys, "record-keys", "", "comma-separated keybinds to script while recording (e.g. space, x, left, shift+up, tab)")
	return o
}

// apply copies the recording options onto a controller.
func (o *recordOpts) apply(ctrl gui.Controller) gui.Controller {
	ctrl.Record = o.path
	ctrl.RecordFrames = o.frames
	ctrl.RecordFPS = o.fps
	ctrl.RecordScale = o.scale
	ctrl.RecordKeys = o.keys
	return ctrl
}

// guiController wires the visualizer to the solver: a strategy list, a fresh-scramble
// source and a solve function. initial, if set, is the cube shown on load. A non-zero
// seed makes the scramble sequence reproducible (for deterministic recordings).
func guiController(initial *cube.Cube, start int, seed int64) gui.Controller {
	nextSeed := func() int64 { return rand.Int64() }
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

// strategyIndex returns the registry index of a solver name (0 if unknown).
func strategyIndex(name string) int {
	for i, n := range solver.Names() {
		if n == name {
			return i
		}
	}
	return 0
}

func cmdSolvers(args []string) error {
	fs := flag.NewFlagSet("solvers", flag.ContinueOnError)
	format, output := addFormatFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validFormat(*format); err != nil {
		return err
	}
	list := make([]solverInfo, 0, len(solver.All()))
	for _, s := range solver.All() {
		list = append(list, solverInfo{Name: s.Name(), Describe: s.Describe()})
	}
	return renderSolvers(*format, *output, list)
}

func cmdScramble(args []string) error {
	fs := flag.NewFlagSet("scramble", flag.ContinueOnError)
	n := fs.Int("n", 25, "number of random moves")
	seed := fs.Int64("seed", 0, "random seed")
	format, output := addFormatFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := validFormat(*format); err != nil {
		return err
	}
	moves := cube.Scramble(*n, *seed)
	c := cube.Solved().Applied(moves...)
	return renderScramble(*format, *output, scrambleOutput{
		N:        *n,
		Seed:     *seed,
		Scramble: cube.FormatMoves(moves),
		Facelets: c.ToFacelets().String(),
	})
}

func cmdVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	input := fs.String("input", "", "54-character facelet string, or @- for stdin")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := cubeFromInput(*input, false, nil)
	if err != nil {
		return err
	}
	solved := ""
	if c.IsSolved() {
		solved = " (already solved)"
	}
	fmt.Printf("valid cube%s\n", solved)
	return nil
}

func cmdScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	r, err := robot.New()
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	f, err := r.Scan()
	if err != nil {
		return err
	}
	fmt.Println(f)
	return nil
}

func cmdGenTables(args []string) error {
	fs := flag.NewFlagSet("gen-tables", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	fmt.Println("building prune tables...")
	if err := warmTables(); err != nil {
		return err
	}
	fmt.Println("done")
	return nil
}

// warmTables builds and caches every prune table by running the strongest solver
// once. Doing this before a parallel batch keeps goroutines from all blocking on the
// one-time (sync.Once) table build.
func warmTables() error {
	s, _ := solver.Get("prune")
	_, err := s.Solve(cube.ScrambledCube(20, 1))
	return err
}

func cmdView(args []string) error {
	fs := flag.NewFlagSet("view", flag.ContinueOnError)
	seed := fs.Int64("seed", 0, "scramble seed (0 = random each run); set for reproducible recordings")
	rec := addRecordFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	// The visualizer is self-driving: it scrambles and solves on its own; press "r"
	// for a new scramble and "s" to switch solver.
	return gui.Play(rec.apply(guiController(nil, strategyIndex("multi"), *seed)))
}
