package search

import (
	"os"
	"path/filepath"

	"github.com/danielriddell21/rubix/internal/cube"
)

// CoordFunc projects a cube to an integer coordinate. For a pruning table to be
// valid, the coordinate's transition under every move in the table's move set must
// depend only on the coordinate itself (a well-defined quotient of the group).
type CoordFunc func(cube.Cube) int

// Unreached marks coordinates a BFS never reached.
const Unreached = uint8(0xff)

// BuildBFS computes a pruning table: for each value of coordOf it stores the
// minimal number of moves (from the given set) needed to bring that coordinate to
// its solved value (coordOf(Solved())). It performs a breadth-first search from the
// solved cube, deduplicating by coordinate.
func BuildBFS(size int, coordOf CoordFunc, moves []cube.Move) []uint8 {
	dist := make([]uint8, size)
	for i := range dist {
		dist[i] = Unreached
	}
	start := cube.Solved()
	dist[coordOf(start)] = 0
	frontier := []cube.Cube{start}
	for depth := uint8(0); len(frontier) > 0; depth++ {
		var next []cube.Cube
		for _, c := range frontier {
			for _, m := range moves {
				nc := c
				nc.Apply(m)
				idx := coordOf(nc)
				if dist[idx] == Unreached {
					dist[idx] = depth + 1
					next = append(next, nc)
				}
			}
		}
		frontier = next
	}
	return dist
}

// tablesDir returns the directory used to cache pruning tables.
func tablesDir() string {
	if d := os.Getenv("RUBIX_TABLES"); d != "" {
		return d
	}
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "rubix")
	}
	return "tables"
}

// LoadOrBuild returns a cached pruning table named name, building and persisting it
// with build when absent. Caching is best effort; failures fall back to building.
func LoadOrBuild(name string, build func() []uint8) []uint8 {
	path := filepath.Join(tablesDir(), name+".prt")
	if data, err := os.ReadFile(path); err == nil {
		return data
	}
	data := build()
	if err := os.MkdirAll(tablesDir(), 0o755); err == nil {
		_ = os.WriteFile(path, data, 0o644)
	}
	return data
}
