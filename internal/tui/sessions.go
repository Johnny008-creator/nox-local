package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Johnny008-creator/nox-local/internal/session"
)

// Other NØX windows: /sessions lists every running NØX (like `nox agents`),
// /send passes a note to one of them, and incoming notes show up here.
// /loop re-sends a prompt on an interval while this window stays open.

// publishStatus writes this window's state for `nox agents`.
func (m *Model) publishStatus() {
	state := "idle"
	switch {
	case m.approval != nil:
		state = "needs approval"
	case m.busy:
		state = "working"
	}
	if state == m.lastState && time.Since(m.lastStatus) < 20*time.Second {
		return
	}
	m.lastState, m.lastStatus = state, time.Now()
	sid, title := "", ""
	if m.saved != nil {
		sid, title = m.saved.ID, m.saved.Title()
	}
	if title == "" {
		for _, e := range m.entries {
			if e.kind == kUser {
				title = truncRight(strings.Join(strings.Fields(e.text), " "), 60)
				break
			}
		}
	}
	session.Status(m.dir, sid, state, title, m.prov.ID()+" · "+m.modelName())
}

// FormatRunning renders the running NØX processes (nox agents, /sessions).
func FormatRunning(list []session.Running) string {
	if len(list) == 0 {
		return "no NØX running"
	}
	var b strings.Builder
	for _, r := range list {
		mark := "○"
		switch r.State {
		case "working":
			mark = "●"
		case "needs approval":
			mark = "!"
		}
		me := ""
		if r.PID == os.Getpid() {
			me = " (this window)"
		}
		fmt.Fprintf(&b, "%s %-7d %-15s %-22s %s%s\n   %s · %s ago\n", mark, r.PID, r.State, filepath.Base(r.Dir), r.Title, me, r.Model, time.Since(r.Updated).Truncate(time.Second))
	}
	return strings.TrimRight(b.String(), "\n")
}

func cmdSessions(m *Model, _ string) tea.Cmd {
	m.info("running NØX windows · /send <pid> <text> passes a note\n" + FormatRunning(session.List()))
	return nil
}

func cmdSend(m *Model, arg string) tea.Cmd {
	pidS, text, ok := strings.Cut(strings.TrimSpace(arg), " ")
	pid, err := strconv.Atoi(pidS)
	if !ok || err != nil || strings.TrimSpace(text) == "" {
		m.info("usage: /send <pid> <text> · /sessions lists the pids")
		return nil
	}
	if err := session.Send(pid, filepath.Base(m.dir), text); err != nil {
		m.errorf(err.Error())
		return nil
	}
	m.info(fmt.Sprintf("sent to %d", pid))
	return nil
}

// checkInbox handles notes from other windows and prompts from the Discord
// bridge (polled on the tick). A note is shown; a Discord message is a prompt
// and starts a turn, which is the whole point of the bridge.
func (m *Model) checkInbox() tea.Cmd {
	if (m.frame+m.plainTicks)%18 != 0 { // ~2 s
		return nil
	}
	var cmd tea.Cmd
	for _, msg := range session.Inbox() {
		if msg["from"] == "discord" {
			if m.busy {
				// The turn in flight owns the session; a queued prompt would
				// otherwise silently replace it.
				m.info("✉ discord (queued) · " + msg["text"])
				m.queue = append(m.queue, msg["text"])
				continue
			}
			m.info("✉ discord · " + msg["text"])
			cmd = m.startTurnOn(m.prov, msg["text"], m.mode, false)
			continue
		}
		m.info("✉ from " + msg["from"] + ": " + msg["text"])
		m.toast("message from " + msg["from"])
		m.inbox = append(m.inbox, msg["from"]+": "+msg["text"])
	}
	return cmd
}

type loopT struct {
	every  time.Duration
	prompt string
	next   time.Time
}

func cmdLoop(m *Model, arg string) tea.Cmd {
	arg = strings.TrimSpace(arg)
	if arg == "" || arg == "off" {
		if m.loop != nil && arg == "off" {
			m.info("loop stopped")
		} else if m.loop != nil {
			m.info(fmt.Sprintf("loop every %s: %s · /loop off stops it", m.loop.every, m.loop.prompt))
		} else {
			m.info("usage: /loop <interval> <prompt> · e.g. /loop 10m check the CI and fix anything red")
		}
		if arg == "off" {
			m.loop = nil
		}
		return nil
	}
	ivS, prompt, _ := strings.Cut(arg, " ")
	iv, err := time.ParseDuration(ivS)
	if err != nil || iv < 30*time.Second || strings.TrimSpace(prompt) == "" {
		m.errorf("usage: /loop <interval ≥ 30s, e.g. 5m> <prompt>")
		return nil
	}
	m.loop = &loopT{every: iv, prompt: prompt, next: time.Now()}
	m.info(fmt.Sprintf("loop: “%s” every %s while this window is open · /loop off", prompt, iv))
	return nil
}

// dueLoop starts the looped prompt when it is due and NØX is idle.
func (m *Model) dueLoop() tea.Cmd {
	if m.loop == nil || m.busy || m.approval != nil || time.Now().Before(m.loop.next) {
		return nil
	}
	m.loop.next = time.Now().Add(m.loop.every)
	return m.dispatch(m.loop.prompt)
}
