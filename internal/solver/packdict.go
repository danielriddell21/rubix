package solver

import (
	"sync"

	"github.com/danielriddell21/rubix/pkg/cube"
)

func packedDict(moves []cube.Move, depth int) map[state]int {
	dist := map[state]int{solved: 0}
	frontier := []state{solved}
	for d := range depth {
		var next []state
		for _, s := range frontier {
			for _, m := range moves {
				ns := s.apply(m)
				if _, ok := dist[ns]; !ok {
					dist[ns] = d + 1
					next = append(next, ns)
				}
			}
		}
		frontier = next
	}
	return dist
}

const dictBase = 1000

func dictSolve(s state, dict map[state]int) []cube.Move {
	var path []cube.Move
	for {
		d, ok := dict[s]
		if !ok || d == 0 {
			return path
		}
		for _, m := range allMoves {
			ns := s.apply(m)
			if nd, ok := dict[ns]; ok && nd == d-1 {
				path = append(path, m)
				s = ns
				break
			}
		}
	}
}

func withLookup(dict map[state]int, fallback stateEval) stateEval {
	return func(s state) int {
		if d, ok := dict[s]; ok {
			return dictBase - d
		}
		return fallback(s)
	}
}

var (
	fullDictOnce sync.Once
	fullDict     map[state]int

	dominoDictOnce sync.Once
	dominoDict     map[state]int
)

func fullLookup() map[state]int {
	fullDictOnce.Do(func() { fullDict = packedDict(allMoves, 6) })
	return fullDict
}

func dominoLookup() map[state]int {
	dominoDictOnce.Do(func() { dominoDict = packedDict(dominoMoves, 7) })
	return dominoDict
}
