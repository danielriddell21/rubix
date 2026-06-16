package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/danielriddell21/rubix/internal/gui"
)

// maxWindows caps how many coordinated cube windows can be open at once (leader + children).
const maxWindows = 16

// eofType is an internal pseudo-message a child's reader injects when its pipe closes, so
// the hub drops that window.
const eofType = "_eof"

// hub relays the window-coordination protocol between the leader window (participant 0) and
// the child windows it spawns. A "state" or "rescramble" from any window is rebroadcast to
// the others; "add"/"remove" spawn and close child processes. All participant bookkeeping
// happens in the single run goroutine (plus shutdown from the main goroutine), guarded by mu.
type hub struct {
	self    string      // path to this binary, for spawning children
	inbox   chan srcMsg // merged inbound from every participant
	mu      sync.Mutex
	parts   map[int]*participant
	nextID  int     // next participant id (0 = leader)
	spawned int     // monotonic count of children ever spawned (for window titles)
	last    gui.Msg // last shared "state", replayed to a freshly added child
}

type participant struct {
	out chan gui.Msg // messages TO this window
	cmd *exec.Cmd    // nil for the leader
}

type srcMsg struct {
	id int
	m  gui.Msg
}

func newHub(self string) *hub {
	return &hub{self: self, inbox: make(chan srcMsg, 128), parts: map[int]*participant{}}
}

// addParticipant registers an outbound channel and returns its id. A newly added window is
// immediately sent the last known shared state so it mirrors the others on open.
func (h *hub) addParticipant(out chan gui.Msg, cmd *exec.Cmd) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	id := h.nextID
	h.nextID++
	h.parts[id] = &participant{out: out, cmd: cmd}
	if h.last.Type == "state" {
		trySend(out, h.last)
	}
	return id
}

// run processes inbound messages until the inbox is closed.
func (h *hub) run() {
	for sm := range h.inbox {
		h.handle(sm.id, sm.m)
	}
}

func (h *hub) handle(src int, m gui.Msg) {
	switch m.Type {
	case "state":
		h.mu.Lock()
		h.last = m
		h.mu.Unlock()
		h.broadcastExcept(src, m)
	case "rescramble":
		h.broadcastExcept(src, m)
	case "add":
		h.spawnChild()
	case "remove":
		h.removeNewest()
	case eofType:
		h.drop(src)
	}
}

// broadcastExcept delivers m to every participant but src. Sends are non-blocking so one
// slow window never stalls the others (shared state is idempotent — the next change resends).
func (h *hub) broadcastExcept(src int, m gui.Msg) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, p := range h.parts {
		if id != src {
			trySend(p.out, m)
		}
	}
}

// drop removes a window (its pipe closed) and stops its writer / kills its process.
func (h *hub) drop(id int) {
	h.mu.Lock()
	p := h.parts[id]
	delete(h.parts, id)
	if p != nil {
		close(p.out)
	}
	h.mu.Unlock()
	if p != nil && p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}

// removeNewest asks the most recently spawned child window to close.
func (h *hub) removeNewest() {
	h.mu.Lock()
	newest, out := -1, chan gui.Msg(nil)
	for id, p := range h.parts {
		if p.cmd != nil && id > newest {
			newest, out = id, p.out
		}
	}
	h.mu.Unlock()
	if out != nil {
		trySend(out, gui.Msg{Type: "quit"})
	}
}

// spawnChild starts a new child window process and wires its pipes to the hub.
func (h *hub) spawnChild() {
	h.mu.Lock()
	if len(h.parts) >= maxWindows {
		h.mu.Unlock()
		return
	}
	h.spawned++
	idx := h.spawned
	h.mu.Unlock()

	cmd := exec.Command(h.self, "view", fmt.Sprintf("--child=%d", idx))
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}

	out := make(chan gui.Msg, 64)
	id := h.addParticipant(out, cmd)

	go func() { // hub -> child stdin
		enc := json.NewEncoder(stdin)
		for m := range out {
			if enc.Encode(m) != nil {
				break
			}
		}
		_ = stdin.Close()
	}()
	go func() { // child stdout -> hub, then signal removal and reap
		dec := json.NewDecoder(stdout)
		for {
			var m gui.Msg
			if dec.Decode(&m) != nil {
				break
			}
			h.inbox <- srcMsg{id, m}
		}
		h.inbox <- srcMsg{id, gui.Msg{Type: eofType}}
		_ = cmd.Wait()
	}()
}

// shutdown kills every child window (used when the leader window closes). Each child also
// self-terminates when its stdin closes, so this is just prompt cleanup.
func (h *hub) shutdown() {
	h.mu.Lock()
	cmds := make([]*exec.Cmd, 0, len(h.parts))
	for _, p := range h.parts {
		if p.cmd != nil {
			cmds = append(cmds, p.cmd)
		}
	}
	h.mu.Unlock()
	for _, c := range cmds {
		if c.Process != nil {
			_ = c.Process.Kill()
		}
	}
}

func trySend(ch chan gui.Msg, m gui.Msg) {
	select {
	case ch <- m:
	default:
	}
}

// runLeader opens the leader window and runs the coordination hub that spawns and syncs
// child windows. It blocks until the leader window closes, then tears down the children.
func runLeader(ctrl gui.Controller) error {
	h := newHub(os.Args[0])
	leaderIn := make(chan gui.Msg, 64)  // hub -> leader (the leader's participant out)
	leaderOut := make(chan gui.Msg, 64) // leader -> hub
	h.addParticipant(leaderIn, nil)     // id 0
	go h.run()
	go func() {
		for m := range leaderOut {
			h.inbox <- srcMsg{0, m}
		}
	}()
	err := gui.Play(ctrl, &gui.Link{In: leaderIn, Out: leaderOut})
	close(leaderOut)
	h.shutdown()
	return err
}

// runChild opens a child window connected to the leader over stdin/stdout. When the leader
// exits (stdin closes) the window terminates itself.
func runChild(ctrl gui.Controller) error {
	in := make(chan gui.Msg, 64)
	out := make(chan gui.Msg, 64)
	go func() { // leader stdin -> in
		dec := json.NewDecoder(os.Stdin)
		for {
			var m gui.Msg
			if dec.Decode(&m) != nil {
				break
			}
			in <- m
		}
		close(in) // leader gone: closing In makes the window terminate
	}()
	go func() { // out -> leader via stdout
		enc := json.NewEncoder(os.Stdout)
		for m := range out {
			if enc.Encode(m) != nil {
				break
			}
		}
	}()
	return gui.Play(ctrl, &gui.Link{In: in, Out: out})
}
