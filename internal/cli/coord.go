package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/danielriddell21/rubix/internal/gui"
)

const maxWindows = 16

const eofType = "_eof"

type hub struct {
	self    string
	inbox   chan srcMsg
	done    chan struct{}
	mu      sync.Mutex
	parts   map[int]*participant
	nextID  int
	spawned int
	last    gui.Msg
}

type participant struct {
	out chan gui.Msg
	cmd *exec.Cmd
}

type srcMsg struct {
	id int
	m  gui.Msg
}

func newHub(self string) *hub {
	return &hub{self: self, inbox: make(chan srcMsg, 128), done: make(chan struct{}), parts: map[int]*participant{}}
}

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

func (h *hub) run() {
	// Stop on shutdown rather than ranging over inbox: child reader goroutines
	// may still send after teardown, so inbox is never closed.
	for {
		select {
		case sm := <-h.inbox:
			h.handle(sm.id, sm.m)
		case <-h.done:
			return
		}
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

func (h *hub) broadcastExcept(src int, m gui.Msg) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, p := range h.parts {
		if id != src {
			trySend(p.out, m)
		}
	}
}

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

func (h *hub) spawnChild() {
	h.mu.Lock()
	if len(h.parts) >= maxWindows {
		h.mu.Unlock()
		return
	}
	h.spawned++
	idx := h.spawned
	h.mu.Unlock()

	cmd := exec.CommandContext(context.Background(), h.self, "view", fmt.Sprintf("--child=%d", idx))
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "spawn child window: stdin pipe: %v\n", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "spawn child window: stdout pipe: %v\n", err)
		return
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "spawn child window: start: %v\n", err)
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
	close(h.done)
}

func trySend(ch chan gui.Msg, m gui.Msg) {
	select {
	case ch <- m:
	default:
	}
}

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
	err := gui.Run(gui.Config{Controller: ctrl, Link: &gui.Link{In: leaderIn, Out: leaderOut}})
	close(leaderOut)
	h.shutdown()
	if err != nil {
		return fmt.Errorf("run gui: %w", err)
	}
	return nil
}

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
	if err := gui.Run(gui.Config{Controller: ctrl, Link: &gui.Link{In: in, Out: out}}); err != nil {
		return fmt.Errorf("run gui: %w", err)
	}
	return nil
}
