package cli

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/danielriddell21/rubix/pkg/cube"
)

// addFormatFlags registers the shared -format/-output flags. The CLI uses stdlib flag
// (no persistent flags), so each machine-readable command calls this on its flag set.
func addFormatFlags(fs *flag.FlagSet) (format, output *string) {
	format = fs.String("format", "text", "output format: text, json, or csv")
	output = fs.String("output", "", "write output to this file instead of stdout")
	return format, output
}

// validFormat reports an error for an unrecognised -format value.
func validFormat(f string) error {
	switch f {
	case "text", "json", "csv":
		return nil
	default:
		return fmt.Errorf("invalid -format %q (want text, json, or csv)", f)
	}
}

// withOutput runs fn against the -output file, or stdout when no path is given.
func withOutput(path string, fn func(io.Writer) error) error {
	if path == "" {
		return fn(os.Stdout)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := fn(f); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeCSV(w io.Writer, header []string, rows [][]string) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(header); err != nil {
		return err
	}
	if err := cw.WriteAll(rows); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

// moveStrings renders each move in Singmaster notation, for the JSON move_list field.
func moveStrings(ms []cube.Move) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.String()
	}
	return out
}

func msHuman(ms int64) string { return (time.Duration(ms) * time.Millisecond).String() }

// --- solve ---

type solveOutput struct {
	Strategy  string   `json:"strategy"`
	Solved    bool     `json:"solved"`
	Moves     string   `json:"moves"`
	MoveList  []string `json:"move_list"`
	MoveCount int      `json:"move_count"`
	Nodes     uint64   `json:"nodes"`
	ElapsedMS int64    `json:"elapsed_ms"`
}

func renderSolve(format, output string, o solveOutput) error {
	return withOutput(output, func(w io.Writer) error {
		switch format {
		case "json":
			return writeJSON(w, o)
		case "csv":
			return writeCSV(w,
				[]string{"strategy", "solved", "move_count", "moves", "nodes", "elapsed_ms"},
				[][]string{{o.Strategy, strconv.FormatBool(o.Solved), strconv.Itoa(o.MoveCount),
					o.Moves, strconv.FormatUint(o.Nodes, 10), strconv.FormatInt(o.ElapsedMS, 10)}})
		default:
			if !o.Solved {
				fmt.Fprintf(w, "%s gave up after %d moves (this solver does not always succeed)\n", o.Strategy, o.MoveCount)
				return nil
			}
			fmt.Fprintf(w, "strategy: %s\n", o.Strategy)
			fmt.Fprintf(w, "solution (%d moves): %s\n", o.MoveCount, o.Moves)
			fmt.Fprintf(w, "nodes: %d   time: %s\n", o.Nodes, msHuman(o.ElapsedMS))
			return nil
		}
	})
}

// --- scramble ---

type scrambleOutput struct {
	N        int    `json:"n"`
	Seed     int64  `json:"seed"`
	Scramble string `json:"scramble"`
	Facelets string `json:"facelets"`
}

func renderScramble(format, output string, o scrambleOutput) error {
	return withOutput(output, func(w io.Writer) error {
		switch format {
		case "json":
			return writeJSON(w, o)
		case "csv":
			return writeCSV(w,
				[]string{"n", "seed", "scramble", "facelets"},
				[][]string{{strconv.Itoa(o.N), strconv.FormatInt(o.Seed, 10), o.Scramble, o.Facelets}})
		default:
			fmt.Fprintf(w, "scramble: %s\n", o.Scramble)
			fmt.Fprintf(w, "facelets: %s\n", o.Facelets)
			return nil
		}
	})
}

// --- solvers ---

type solverInfo struct {
	Name     string `json:"name"`
	Describe string `json:"describe"`
}

func renderSolvers(format, output string, list []solverInfo) error {
	return withOutput(output, func(w io.Writer) error {
		switch format {
		case "json":
			return writeJSON(w, list)
		case "csv":
			rows := make([][]string, len(list))
			for i, s := range list {
				rows[i] = []string{s.Name, s.Describe}
			}
			return writeCSV(w, []string{"name", "describe"}, rows)
		default:
			for i, s := range list {
				fmt.Fprintf(w, "%d. %-9s %s\n", i+1, s.Name, s.Describe)
			}
			return nil
		}
	})
}

// --- replica ---

type replicaRow struct {
	Index     int    `json:"index"`
	Strategy  string `json:"strategy"`
	Seed      int64  `json:"seed"`
	MoveCount int    `json:"move_count"`
	Solved    bool   `json:"solved"`
	TimeMS    int64  `json:"time_ms"`
	Nodes     uint64 `json:"nodes"`
	Error     string `json:"error,omitempty"`
}

type replicaSummary struct {
	Solved   int     `json:"solved"`
	Total    int     `json:"total"`
	AvgMoves float64 `json:"avg_moves"`
}

type replicaOutput struct {
	Results []replicaRow   `json:"results"`
	Summary replicaSummary `json:"summary"`
}

func renderReplica(format, output string, o replicaOutput) error {
	return withOutput(output, func(w io.Writer) error {
		switch format {
		case "json":
			return writeJSON(w, o)
		case "csv":
			rows := make([][]string, len(o.Results))
			for i, r := range o.Results {
				rows[i] = []string{strconv.Itoa(r.Index), r.Strategy, strconv.FormatInt(r.Seed, 10),
					strconv.Itoa(r.MoveCount), strconv.FormatBool(r.Solved),
					strconv.FormatInt(r.TimeMS, 10), strconv.FormatUint(r.Nodes, 10), r.Error}
			}
			return writeCSV(w,
				[]string{"index", "strategy", "seed", "move_count", "solved", "time_ms", "nodes", "error"}, rows)
		default:
			// The text path is the existing aligned table; callers pass it the raw results.
			return fmt.Errorf("renderReplica: text format handled by printReplicaTable")
		}
	})
}
