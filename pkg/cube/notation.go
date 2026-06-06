package cube

import (
	"fmt"
	"strings"
)

var moveNames = [NumMoves]string{
	"U", "U2", "U'",
	"R", "R2", "R'",
	"F", "F2", "F'",
	"D", "D2", "D'",
	"L", "L2", "L'",
	"B", "B2", "B'",
}

// String returns the Singmaster notation for the move (e.g. "R'", "U2").
func (m Move) String() string {
	if int(m) < len(moveNames) {
		return moveNames[m]
	}
	return fmt.Sprintf("Move(%d)", uint8(m))
}

var faceMoves = map[byte]Move{
	'U': U, 'R': R, 'F': F, 'D': D, 'L': L, 'B': B,
}

// ParseMove parses a single token such as "R", "U'", "F2".
func ParseMove(tok string) (Move, error) {
	if tok == "" {
		return 0, fmt.Errorf("empty move")
	}
	base, ok := faceMoves[tok[0]]
	if !ok {
		return 0, fmt.Errorf("unsupported move %q (only outer-face turns U R F D L B are supported)", tok)
	}
	switch tok[1:] {
	case "":
		return base, nil
	case "2":
		return base + 1, nil
	case "'", "’":
		return base + 2, nil
	default:
		return 0, fmt.Errorf("invalid move modifier in %q", tok)
	}
}

// ParseMoves parses a whitespace-separated move sequence such as "R U R' U2".
func ParseMoves(s string) ([]Move, error) {
	fields := strings.Fields(s)
	out := make([]Move, 0, len(fields))
	for _, f := range fields {
		m, err := ParseMove(f)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// FormatMoves renders a move sequence as space-separated Singmaster notation.
func FormatMoves(ms []Move) string {
	parts := make([]string, len(ms))
	for i, m := range ms {
		parts[i] = m.String()
	}
	return strings.Join(parts, " ")
}
