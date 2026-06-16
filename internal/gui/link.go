package gui

// This file is build-tag free so both the Ebiten visualizer (gui.go) and the CLI
// coordinator (which spawns and relays between windows) share the same wire types.

// Msg is one line of the window-coordination protocol, exchanged as JSON between the
// leader process and each child window. A "state" message carries the full shared view so
// the receiver can mirror it; the others are events/commands.
type Msg struct {
	Type      string  `json:"t"` // "state" | "rescramble" | "add" | "remove" | "quit"
	Yaw       float32 `json:"yaw,omitempty"`
	Pitch     float32 `json:"pitch,omitempty"`
	Zoom      float32 `json:"zoom,omitempty"`
	UnfoldTo  float32 `json:"unfoldTo,omitempty"`
	Xray      bool    `json:"xray,omitempty"`
	ShowMoves bool    `json:"showMoves,omitempty"`
}

// Link is a window's connection to the coordinator hub: it receives shared updates on In
// and publishes its own local changes on Out. It is nil for an uncoordinated window (a
// lone recording, or solve --view).
type Link struct {
	In  <-chan Msg
	Out chan<- Msg
}
