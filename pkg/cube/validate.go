package cube

import "fmt"

// Validate reports whether the cube is a legal, solvable state. It checks that the
// positions are genuine permutations, that corner/edge orientation sums are zero
// (mod 3 / mod 2), and that corner and edge permutation parities agree.
func (c Cube) Validate() error {
	if err := checkPerm(c.CornerPos[:], 8); err != nil {
		return fmt.Errorf("corner permutation: %w", err)
	}
	if err := checkPerm(c.EdgePos[:], 12); err != nil {
		return fmt.Errorf("edge permutation: %w", err)
	}

	var coSum int
	for _, o := range c.CornerOri {
		if o > 2 {
			return fmt.Errorf("corner orientation %d out of range", o)
		}
		coSum += int(o)
	}
	if coSum%3 != 0 {
		return fmt.Errorf("corner orientation sum %d not divisible by 3 (twisted corner)", coSum)
	}

	var eoSum int
	for _, o := range c.EdgeOri {
		if o > 1 {
			return fmt.Errorf("edge orientation %d out of range", o)
		}
		eoSum += int(o)
	}
	if eoSum%2 != 0 {
		return fmt.Errorf("edge orientation sum %d is odd (flipped edge)", eoSum)
	}

	if parity(c.CornerPos[:]) != parity(c.EdgePos[:]) {
		return fmt.Errorf("corner and edge permutation parities differ (swapped pair)")
	}
	return nil
}

func checkPerm(p []uint8, n int) error {
	var seen uint16
	for _, v := range p {
		if int(v) >= n {
			return fmt.Errorf("value %d out of range", v)
		}
		bit := uint16(1) << v
		if seen&bit != 0 {
			return fmt.Errorf("value %d repeated", v)
		}
		seen |= bit
	}
	return nil
}

// parity returns the permutation parity: 0 for even, 1 for odd.
func parity(p []uint8) int {
	seen := make([]bool, len(p))
	par := 0
	for i := range p {
		if seen[i] {
			continue
		}
		// Walk the cycle containing i.
		length := 0
		for j := i; !seen[j]; j = int(p[j]) {
			seen[j] = true
			length++
		}
		if length%2 == 0 {
			par ^= 1
		}
	}
	return par
}
