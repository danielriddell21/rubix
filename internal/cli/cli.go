// Package cli implements the rubix command-line interface: a small hand-rolled
// subcommand dispatcher over the shared cube/solver/robot/gui packages.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/danielriddell21/rubix/internal/cube"
	"github.com/danielriddell21/rubix/internal/gui"
	"github.com/danielriddell21/rubix/internal/robot"
	"github.com/danielriddell21/rubix/internal/solver"
)

// Run dispatches a subcommand. It returns a process exit code.
func Run(args []string) int {
	if len(args) < 1 {
		usage(os.Stderr)
		return 2
	}
	cmd, rest := args[0], args[1:]
	var err error
	switch cmd {
	case "solve":
		err = cmdSolve(rest)
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
  solvers     list the available solvers in video order
  scramble    generate a scramble and its facelet string
  verify      validate a facelet string
  scan        scan a cube with the robot and print its facelets
  gen-tables  precompute and cache the prune tables
  view        open the visualizer on a cube (needs -tags ebiten)

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
	if err := fs.Parse(args); err != nil {
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
	defer r.Close()

	c, err := cubeFromInput(*input, *scan, r)
	if err != nil {
		return err
	}

	res, err := s.Solve(c)
	if err != nil {
		return err
	}
	if !res.Solved {
		fmt.Printf("%s gave up after %d moves (this solver does not always succeed)\n", res.Strategy, len(res.Moves))
		return nil
	}
	fmt.Printf("strategy: %s\n", res.Strategy)
	fmt.Printf("solution (%d moves): %s\n", len(res.Moves), cube.FormatMoves(res.Moves))
	fmt.Printf("nodes: %d   time: %s\n", res.Nodes, res.Elapsed.Round(1e6))

	if *execute {
		if err := r.Execute(res.Moves); err != nil {
			return err
		}
	}
	if *view {
		return gui.Play(c, res.Moves)
	}
	return nil
}

func cmdSolvers([]string) error {
	for i, s := range solver.All() {
		fmt.Printf("%d. %-9s %s\n", i+1, s.Name(), s.Describe())
	}
	return nil
}

func cmdScramble(args []string) error {
	fs := flag.NewFlagSet("scramble", flag.ContinueOnError)
	n := fs.Int("n", 25, "number of random moves")
	seed := fs.Int64("seed", 0, "random seed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	moves := cube.Scramble(*n, *seed)
	c := cube.Solved().Applied(moves...)
	fmt.Printf("scramble: %s\n", cube.FormatMoves(moves))
	fmt.Printf("facelets: %s\n", c.ToFacelets())
	return nil
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
	defer r.Close()
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
	// Warming the strongest solver builds and caches every prune table.
	s, _ := solver.Get("prune")
	fmt.Println("building prune tables...")
	if _, err := s.Solve(cube.ScrambledCube(20, 1)); err != nil {
		return err
	}
	fmt.Println("done")
	return nil
}

func cmdView(args []string) error {
	fs := flag.NewFlagSet("view", flag.ContinueOnError)
	input := fs.String("input", "", "54-character facelet string, or @- for stdin")
	strategy := fs.String("strategy", "prune", "solver to animate")
	solve := fs.Bool("solve", true, "solve and animate (otherwise just show the cube)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c, err := cubeFromInput(*input, false, nil)
	if err != nil {
		return err
	}
	var moves []cube.Move
	if *solve {
		s, err := solver.Get(*strategy)
		if err != nil {
			return err
		}
		res, err := s.Solve(c)
		if err != nil {
			return err
		}
		moves = res.Moves
	}
	return gui.Play(c, moves)
}
