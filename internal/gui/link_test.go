//go:build !ebiten

package gui

import (
	"encoding/json"
	"testing"
)

func TestMsgJSONRoundTrip(t *testing.T) {
	cases := []Msg{
		{Type: "state", Yaw: 0.6, Pitch: 0.5, Zoom: 1.2, UnfoldTo: 1, Xray: true, ShowMoves: true},
		{Type: "rescramble"},
		{Type: "add"},
		{Type: "quit"},
	}
	for _, in := range cases {
		b, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("marshal %+v: %v", in, err)
		}
		var out Msg
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if out != in {
			t.Errorf("round trip: got %+v want %+v (json %s)", out, in, b)
		}
	}
}
