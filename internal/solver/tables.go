package solver

import (
	"sync"

	"github.com/danielriddell21/rubix/internal/solver/search"
	"github.com/danielriddell21/rubix/pkg/cube"
)

const (
	nUDSlice   = 495
	nSlicePerm = 24
)

var (
	twistSliceOnce sync.Once
	twistSlice     []uint8
	flipSliceOnce  sync.Once
	flipSlice      []uint8

	cornSliceOnce sync.Once
	cornSlice     []uint8
	edge8Once     sync.Once
	edge8Slice    []uint8

	cornOriOnce  sync.Once
	cornOriTab   []uint8
	edgeOriOnce  sync.Once
	edgeOriTab   []uint8
	cornPermOnce sync.Once
	cornPermTab  []uint8

	crossOnce sync.Once
	crossTab  []uint8
	pairOnce  [4]sync.Once
	pairTab   [4][]uint8
)

func phase1Tables() ([]uint8, []uint8) {
	twistSliceOnce.Do(func() {
		twistSlice = search.LoadOrBuild("twist_slice", func() []uint8 {
			return search.BuildBFS(2187*nUDSlice, func(c cube.Cube) int {
				return twistCoord(c)*nUDSlice + udSliceCoord(c)
			}, search.AllMoves)
		})
	})
	flipSliceOnce.Do(func() {
		flipSlice = search.LoadOrBuild("flip_slice", func() []uint8 {
			return search.BuildBFS(2048*nUDSlice, func(c cube.Cube) int {
				return flipCoord(c)*nUDSlice + udSliceCoord(c)
			}, search.AllMoves)
		})
	})
	return twistSlice, flipSlice
}

func phase2Tables() ([]uint8, []uint8) {
	cornSliceOnce.Do(func() {
		cornSlice = search.LoadOrBuild("corn_slice", func() []uint8 {
			return search.BuildBFS(40320*nSlicePerm, func(c cube.Cube) int {
				return cornPermCoord(c)*nSlicePerm + slicePermCoord(c)
			}, search.DominoMoves)
		})
	})
	edge8Once.Do(func() {
		edge8Slice = search.LoadOrBuild("edge8_slice", func() []uint8 {
			return search.BuildBFS(40320*nSlicePerm, func(c cube.Cube) int {
				return edge8PermCoord(c)*nSlicePerm + slicePermCoord(c)
			}, search.DominoMoves)
		})
	})
	return cornSlice, edge8Slice
}

func phase1Heuristic() search.Heuristic {
	ts, fs := phase1Tables()
	return func(c cube.Cube) int {
		a := ts[twistCoord(c)*nUDSlice+udSliceCoord(c)]
		b := fs[flipCoord(c)*nUDSlice+udSliceCoord(c)]
		return int(max(a, b))
	}
}

func phase2Heuristic() search.Heuristic {
	cs, es := phase2Tables()
	return func(c cube.Cube) int {
		a := cs[cornPermCoord(c)*nSlicePerm+slicePermCoord(c)]
		b := es[edge8PermCoord(c)*nSlicePerm+slicePermCoord(c)]
		return int(max(a, b))
	}
}

func cornerOriTable() []uint8 {
	cornOriOnce.Do(func() {
		cornOriTab = search.LoadOrBuild("corner_ori", func() []uint8 {
			return search.BuildBFS(2187, twistCoord, search.AllMoves)
		})
	})
	return cornOriTab
}

func edgeOriTable() []uint8 {
	edgeOriOnce.Do(func() {
		edgeOriTab = search.LoadOrBuild("edge_ori", func() []uint8 {
			return search.BuildBFS(2048, flipCoord, search.AllMoves)
		})
	})
	return edgeOriTab
}

func cornerPermTable() []uint8 {
	cornPermOnce.Do(func() {
		cornPermTab = search.LoadOrBuild("corner_perm", func() []uint8 {
			return search.BuildBFS(40320, cornPermCoord, search.AllMoves)
		})
	})
	return cornPermTab
}

func crossTable() []uint8 {
	crossOnce.Do(func() {
		crossTab = search.LoadOrBuild("cross", func() []uint8 {
			return search.BuildBFS(crossCoordSize, crossCoord, search.AllMoves)
		})
	})
	return crossTab
}

func pairTable(k int) []uint8 {
	pairOnce[k].Do(func() {
		coord := pairCoord(k)
		pairTab[k] = search.BuildBFS(pairCoordSize, coord, search.AllMoves)
	})
	return pairTab[k]
}
