package gui

import "github.com/danielriddell21/crucible/hub"

type Msg struct {
	Type      string  `json:"t"`
	Yaw       float32 `json:"yaw,omitempty"`
	Pitch     float32 `json:"pitch,omitempty"`
	Zoom      float32 `json:"zoom,omitempty"`
	UnfoldTo  float32 `json:"unfoldTo,omitempty"`
	Xray      bool    `json:"xray,omitempty"`
	ShowMoves bool    `json:"showMoves,omitempty"`
}

// Link is the window's channel pair to the multi-window hub. It aliases the
// engine's generic hub link specialised to Msg, so the leader and child
// wiring in internal/cli can hand the window a hub.Link[Msg] directly.
type Link = hub.Link[Msg]
