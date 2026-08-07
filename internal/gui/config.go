package gui

import "github.com/danielriddell21/crucible/record"

type Config struct {
	Controller Controller
	Link       *Link

	// Rec names the recording [Render] writes and Keys is the comma-separated
	// keybind script it replays (e.g. "space", "shift+up", "plus,plus,tab").
	// tools/demogen sets both; the window uses neither.
	Rec  record.Options
	Keys string
}
