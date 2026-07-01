package gui

type Msg struct {
	Type      string  `json:"t"`
	Yaw       float32 `json:"yaw,omitempty"`
	Pitch     float32 `json:"pitch,omitempty"`
	Zoom      float32 `json:"zoom,omitempty"`
	UnfoldTo  float32 `json:"unfoldTo,omitempty"`
	Xray      bool    `json:"xray,omitempty"`
	ShowMoves bool    `json:"showMoves,omitempty"`
}

type Link struct {
	In  <-chan Msg
	Out chan<- Msg
}
