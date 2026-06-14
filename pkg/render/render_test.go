package render

import (
	"reflect"
	"testing"

	"github.com/danielriddell21/rubix/pkg/cube"
)

var testCam = Camera{Yaw: 0.6, Pitch: 0.5, Scale: 40}

func TestProjectSolvedYields54Quads(t *testing.T) {
	quads := Project(cube.Solved(), testCam, cube.NoMove, 0)
	if len(quads) != 54 {
		t.Fatalf("got %d quads, want 54 (6 faces × 9 stickers)", len(quads))
	}
}

func TestProjectDepthSorted(t *testing.T) {
	quads := Project(cube.ScrambledCube(25, 1), testCam, cube.NoMove, 0)
	for i := 1; i < len(quads); i++ {
		if quads[i].Depth < quads[i-1].Depth {
			t.Fatalf("quads not back-to-front: Depth[%d]=%v < Depth[%d]=%v", i, quads[i].Depth, i-1, quads[i-1].Depth)
		}
	}
}

func TestProjectTurnInterpolates(t *testing.T) {
	c := cube.Solved()

	// frac 0 is static, regardless of the move passed.
	static := Project(c, testCam, cube.R, 0)
	if !reflect.DeepEqual(static, Project(c, testCam, cube.NoMove, 0)) {
		t.Fatalf("frac 0 should render a static cube")
	}

	mid := Project(c, testCam, cube.R, 0.5)
	full := Project(c, testCam, cube.R, 1)
	if len(mid) != 54 || len(full) != 54 {
		t.Fatalf("turn animation changed the quad count: mid=%d full=%d", len(mid), len(full))
	}

	// The animation must actually move the turning layer, and frac must interpolate: the
	// three fractions all differ.
	if reflect.DeepEqual(static, full) {
		t.Errorf("frac 1 should differ from the static cube")
	}
	if reflect.DeepEqual(mid, static) || reflect.DeepEqual(mid, full) {
		t.Errorf("frac 0.5 should differ from both frac 0 and frac 1")
	}
}
