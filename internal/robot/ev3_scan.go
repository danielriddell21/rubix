//go:build ev3

package robot

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/danielriddell21/rubix/pkg/cube"
)

type refColor struct {
	r, g, b float64
	set     bool
}

var references [6]refColor

func (e *EV3) readRGB() (r, g, b float64, err error) {
	vals := make([]float64, 3)
	for i := range vals {
		s, e2 := e.color.Value(i)
		if e2 != nil {
			return 0, 0, 0, e2
		}
		n, e2 := strconv.Atoi(strings.TrimSpace(s))
		if e2 != nil {
			return 0, 0, 0, e2
		}
		vals[i] = float64(n)
	}
	return vals[0], vals[1], vals[2], nil
}

func classify(r, g, b float64) cube.Color {
	best := cube.Color(0)
	bestD := math.Inf(1)
	for f := range references {
		ref := references[f]
		if !ref.set {
			continue
		}
		d := (r-ref.r)*(r-ref.r) + (g-ref.g)*(g-ref.g) + (b-ref.b)*(b-ref.b)
		if d < bestD {
			bestD, best = d, cube.Color(f)
		}
	}
	return best
}

func (e *EV3) Calibrate() error {
	for f := range 6 {
		if err := e.presentFace(f); err != nil {
			return err
		}
		r, g, b, err := e.readRGB()
		if err != nil {
			return err
		}
		references[f] = refColor{r, g, b, true}
	}
	return e.Home()
}

func (e *EV3) Scan() (cube.Facelets, error) {
	for f := range references {
		if !references[f].set {
			return cube.Facelets{}, fmt.Errorf("sensor not calibrated; run Calibrate first")
		}
	}
	var f cube.Facelets
	for face := range 6 {
		if err := e.presentFace(face); err != nil {
			return f, err
		}
		for sticker := range 9 {
			if err := e.indexSticker(sticker); err != nil {
				return f, err
			}
			r, g, b, err := e.readRGB()
			if err != nil {
				return f, err
			}
			f[face*9+sticker] = classify(r, g, b)
		}
	}
	return f, e.Home()
}

func (e *EV3) presentFace(face int) error {
	switch face {
	case 0: // U — already up at rest
	case 3: // D
		if err := e.flip(); err != nil {
			return err
		}
		if err := e.flip(); err != nil {
			return err
		}
	case 2: // F
		if err := e.flip(); err != nil {
			return err
		}
	default: // R, L, B reached by a turntable index then a flip
		if err := e.rotate(faceTurn[face]); err != nil {
			return err
		}
		if err := e.flip(); err != nil {
			return err
		}
	}
	return nil
}

var faceTurn = map[int]int{1: 1, 4: -1, 5: 2}

func (e *EV3) indexSticker(n int) error {
	if n == 0 {
		return nil // centre
	}
	return e.rotate(0) // ring indexing handled by the sensor arm on real hardware
}
