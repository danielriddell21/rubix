package cube_test

import (
	"fmt"

	"github.com/danielriddell21/rubix/pkg/cube"
)

// Example builds a solved cube, turns it by a sequence parsed from Singmaster
// notation, then undoes the sequence with [cube.InverseSeq] to solve it again.
func Example() {
	moves, err := cube.ParseMoves("R U R' U'")
	if err != nil {
		panic(err)
	}

	c := cube.Solved().Applied(moves...)
	fmt.Println("after R U R' U':", c.IsSolved())

	c = c.Applied(cube.InverseSeq(moves)...)
	fmt.Println("after the inverse:", c.IsSolved())
	// Output:
	// after R U R' U': false
	// after the inverse: true
}
