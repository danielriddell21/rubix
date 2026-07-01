package search

import (
	"os"
	"path/filepath"

	"github.com/danielriddell21/rubix/pkg/cube"
)

type CoordFunc func(cube.Cube) int

const Unreached = uint8(0xff)

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

func tablesDir() string {
	if d := os.Getenv("RUBIX_TABLES"); d != "" {
		return d
	}
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "rubix")
	}
	return "tables"
}

func LoadOrBuild(name string, build func() []uint8) []uint8 {
	path := filepath.Join(tablesDir(), name+".prt")
	if data, err := os.ReadFile(path); err == nil {
		return data
	}
	data := build()
	if err := os.MkdirAll(tablesDir(), 0o750); err == nil {
		_ = os.WriteFile(path, data, 0o600)
	}
	return data
}
