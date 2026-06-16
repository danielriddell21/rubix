package cli

import (
	"os/exec"
	"testing"
	"time"

	"github.com/danielriddell21/rubix/internal/gui"
)

func recvNB(ch chan gui.Msg) (gui.Msg, bool) {
	select {
	case m := <-ch:
		return m, true
	default:
		return gui.Msg{}, false
	}
}

func TestHubBroadcastsStateToOthers(t *testing.T) {
	h := newHub("self")
	a := make(chan gui.Msg, 8)
	b := make(chan gui.Msg, 8)
	c := make(chan gui.Msg, 8)
	ia := h.addParticipant(a, nil)
	h.addParticipant(b, nil)
	h.addParticipant(c, nil)

	st := gui.Msg{Type: "state", Yaw: 1, Pitch: 2, Zoom: 3, Xray: true}
	h.handle(ia, st)

	if m, ok := recvNB(a); ok {
		t.Errorf("sender should not receive its own state, got %+v", m)
	}
	if m, ok := recvNB(b); !ok || m != st {
		t.Errorf("b got %+v ok=%v, want %+v", m, ok, st)
	}
	if m, ok := recvNB(c); !ok || m != st {
		t.Errorf("c got %+v ok=%v, want %+v", m, ok, st)
	}
}

func TestHubReplaysLastStateToNewWindow(t *testing.T) {
	h := newHub("self")
	a := make(chan gui.Msg, 8)
	ia := h.addParticipant(a, nil)
	st := gui.Msg{Type: "state", Yaw: 0.5}
	h.handle(ia, st) // stores last; no other windows yet

	b := make(chan gui.Msg, 8)
	h.addParticipant(b, nil) // should immediately receive the last shared state
	if m, ok := recvNB(b); !ok || m != st {
		t.Errorf("new window got %+v ok=%v, want %+v", m, ok, st)
	}
}

func TestHubRescrambleBroadcasts(t *testing.T) {
	h := newHub("self")
	a := make(chan gui.Msg, 8)
	b := make(chan gui.Msg, 8)
	ia := h.addParticipant(a, nil)
	h.addParticipant(b, nil)

	h.handle(ia, gui.Msg{Type: "rescramble"})
	if _, ok := recvNB(a); ok {
		t.Errorf("sender should not receive its own rescramble")
	}
	if m, ok := recvNB(b); !ok || m.Type != "rescramble" {
		t.Errorf("b want rescramble, got %+v ok=%v", m, ok)
	}
}

func TestHubDropRemovesParticipant(t *testing.T) {
	h := newHub("self")
	a := make(chan gui.Msg, 8)
	ia := h.addParticipant(a, nil)

	h.handle(ia, gui.Msg{Type: eofType})

	h.mu.Lock()
	n := len(h.parts)
	h.mu.Unlock()
	if n != 0 {
		t.Errorf("after eof, parts=%d want 0", n)
	}
	if _, open := <-a; open {
		t.Errorf("dropped participant's channel should be closed")
	}
}

func TestHubRunProcessesInbox(t *testing.T) {
	h := newHub("self")
	a := make(chan gui.Msg, 8)
	b := make(chan gui.Msg, 8)
	ia := h.addParticipant(a, nil)
	h.addParticipant(b, nil)

	go h.run()
	h.inbox <- srcMsg{ia, gui.Msg{Type: "state", Yaw: 9}}

	select {
	case m := <-b:
		if m.Yaw != 9 {
			t.Errorf("run delivered %+v, want Yaw 9", m)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not relay the message")
	}
	close(h.inbox)
}

func TestHubSpawnChildCapReturns(t *testing.T) {
	h := newHub("self")
	for i := 0; i < maxWindows; i++ {
		h.addParticipant(make(chan gui.Msg, 1), nil)
	}
	h.spawnChild() // at the cap: must not start a process or add a participant
	h.mu.Lock()
	n := len(h.parts)
	h.mu.Unlock()
	if n != maxWindows {
		t.Errorf("spawnChild past the cap changed parts to %d, want %d", n, maxWindows)
	}
}

func TestHubShutdownNoPanic(t *testing.T) {
	h := newHub("self")
	h.addParticipant(make(chan gui.Msg, 1), nil)                  // leader, no process
	h.addParticipant(make(chan gui.Msg, 1), exec.Command("noop")) // child, never started
	h.shutdown()                                                  // must not panic on an unstarted process
}

func TestHubRemoveNewestNoChildren(t *testing.T) {
	h := newHub("self")
	leader := make(chan gui.Msg, 1)
	h.addParticipant(leader, nil)
	h.removeNewest() // nothing to remove
	if _, ok := recvNB(leader); ok {
		t.Errorf("leader should not be told to quit when there are no children")
	}
}

func TestTrySendDropsWhenFull(t *testing.T) {
	ch := make(chan gui.Msg, 1)
	trySend(ch, gui.Msg{Type: "state", Yaw: 1})
	trySend(ch, gui.Msg{Type: "state", Yaw: 2}) // full: dropped, must not block
	got := <-ch
	if got.Yaw != 1 {
		t.Errorf("got %+v, want the first message (Yaw 1)", got)
	}
	if _, ok := recvNB(ch); ok {
		t.Errorf("second message should have been dropped")
	}
}

func TestHubRemoveNewestQuitsChild(t *testing.T) {
	h := newHub("self")
	leader := make(chan gui.Msg, 8)
	h.addParticipant(leader, nil) // id 0, leader (no process)
	child := make(chan gui.Msg, 8)
	h.addParticipant(child, exec.Command("rubix-noop")) // id 1, a child (never started)

	h.handle(0, gui.Msg{Type: "remove"})

	if m, ok := recvNB(child); !ok || m.Type != "quit" {
		t.Errorf("newest child should be told to quit, got %+v ok=%v", m, ok)
	}
	if _, ok := recvNB(leader); ok {
		t.Errorf("leader (no process) should not be told to quit")
	}
}
