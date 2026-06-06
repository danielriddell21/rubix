package solver

import (
	"sync"

	"github.com/danielriddell21/rubix/pkg/cube"
)

// packdict.go is the backward "meet in the middle" lookup from video 1, on the packed
// state: a breadth-first expansion outward from solved recording each position's exact
// distance home. A descent that lands on one of these knows it is on a winning path.
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

// dictBase outranks any solved-cubie count, so a dictionary hit (a known path home) is
// always preferred and pulls the descent straight in.
const dictBase = 1000

// dictSolve walks a dictionary position home, always stepping to a neighbour one move
// closer to solved (over the full move set, so it can use front/back turns even when
// the forward search could not).
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

// withLookup wraps an evaluation so a position in the dictionary scores by its (small)
// distance home; otherwise it falls back to the base evaluation.
func withLookup(dict map[state]int, fallback stateEval) stateEval {
	return func(s state) int {
		if d, ok := dict[s]; ok {
			return dictBase - d
		}
		return fallback(s)
	}
}

// Cached dictionaries (built once on first use): the full 18-move dictionary used by
// sandwich and oriented, and the domino-move dictionary used by domino's phase 2.
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
