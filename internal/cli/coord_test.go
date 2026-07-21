package cli

import (
	"testing"

	"github.com/danielriddell21/crucible/hub"

	"github.com/danielriddell21/rubix/internal/gui"
)

func TestRoutePolicy(t *testing.T) {
	cases := []struct {
		typ  string
		want hub.Route
	}{
		{"state", hub.RouteState},
		{"rescramble", hub.RouteBroadcast},
		{"add", hub.RouteSpawn},
		{"remove", hub.RouteCloseNewest},
		{"quit", hub.RouteNone},
		{"", hub.RouteNone},
	}
	for _, c := range cases {
		if got := route(gui.Msg{Type: c.typ}); got != c.want {
			t.Errorf("route(%q) = %v, want %v", c.typ, got, c.want)
		}
	}
}

func TestHubConfigChildArgs(t *testing.T) {
	cfg := hubConfig()
	if cfg.Quit.Type != "quit" {
		t.Errorf("quit message = %q", cfg.Quit.Type)
	}
	if args := cfg.ChildArgs(3); len(args) != 2 || args[0] != "view" || args[1] != "--child=3" {
		t.Errorf("ChildArgs(3) = %v", args)
	}
}
