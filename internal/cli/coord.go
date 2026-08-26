package cli

import (
	"fmt"
	"os"

	"github.com/danielriddell21/crucible/hub"

	"github.com/danielriddell21/rubix/internal/gui"
)

// route classifies a window message for the multi-window hub: shared cube
// state is cached and rebroadcast, a rescramble is a one-off broadcast, and
// add/remove open and close windows.
func route(m gui.Msg) hub.Route {
	switch m.Type {
	case "state":
		return hub.RouteState
	case "rescramble":
		return hub.RouteBroadcast
	case "add":
		return hub.RouteSpawn
	case "remove":
		return hub.RouteCloseNewest
	}
	return hub.RouteNone
}

func hubConfig() hub.Config[gui.Msg] {
	return hub.Config[gui.Msg]{
		Self: os.Args[0],
		ChildArgs: func(idx int) []string {
			return []string{"view", fmt.Sprintf("--child=%d", idx)}
		},
		Route: route,
		Quit:  gui.Msg{Type: "quit"},
	}
}

func runWindow(ctrl gui.Controller, l gui.Link) error {
	// SA4023: without -tags ebiten the stub Run always errors, so staticcheck
	// reads this as constant. It is not, in the build that has a GUI.
	if err := gui.Run(gui.Config{Controller: ctrl, Link: &l}); err != nil { //nolint:staticcheck
		return fmt.Errorf("run gui: %w", err)
	}
	return nil
}

func runLeader(ctrl gui.Controller) error {
	if err := hub.RunLeader(hubConfig(), func(l gui.Link) error { return runWindow(ctrl, l) }); err != nil {
		return fmt.Errorf("leader window: %w", err)
	}
	return nil
}

func runChild(ctrl gui.Controller) error {
	if err := hub.RunChild(func(l gui.Link) error { return runWindow(ctrl, l) }); err != nil {
		return fmt.Errorf("child window: %w", err)
	}
	return nil
}
