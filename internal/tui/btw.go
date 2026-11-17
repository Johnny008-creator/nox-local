package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Johnny008-creator/nox-local/internal/agent"
)

// /btw asks a side question without touching the turn in flight: a separate
// one-shot request, read-only, with its own short history. The answer appears
// as a note and does not become part of the conversation the main turn is
// building — that is the whole point. Claude Code's /btw is the same idea.

// btwDoneMsg is the answer to a side question.
type btwDoneMsg struct {
	question string
	text     string
	secs     float64
	err      error
}

// btwContext is how much of the conversation a side question sees. Enough to
// understand "why is that failing?", short enough to stay cheap.
const btwTurns = 8

func cmdBtw(m *Model, arg string) tea.Cmd {
	q := strings.TrimSpace(arg)
	if q == "" {
		m.info("usage: /btw <question> · asks a side question without stopping what is running")
		return nil
	}
	p := m.prov
	if p == nil || !p.Available() {
		m.errorf("no provider is set up for a side question")
		return nil
	}
	dir := m.dir
	model := m.model
	if model == "" || model == "default" {
		model = p.DefaultModel()
	}
	transcript := m.btwTranscript()

	m.entries = append(m.entries, entry{kind: kUser, text: "btw: " + q, prov: p.ID()})
	m.toast("btw · asking on the side")
	m.info("btw · the answer arrives when the model is free; the turn in flight is not touched")

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()
		start := time.Now()
		prompt := q
		if transcript != "" {
			prompt = "Conversation so far (for context only):\n<transcript>\n" + transcript + "\n</transcript>\n\nQuestion: " + q
		}
		ch, err := p.Run(ctx, btwRequest(prompt, model, dir))
		if err != nil {
			return btwDoneMsg{question: q, err: err}
		}
		var text strings.Builder
		var runErr error
		for ev := range ch {
			switch ev.Kind {
			case agent.EvText:
				text.WriteString(ev.Text)
			case agent.EvDone:
				if ev.IsError {
					runErr = fmt.Errorf("%s", ev.Text)
				}
			case agent.EvError:
				runErr = ev.Err
			}
		}
		return btwDoneMsg{question: q, text: strings.TrimSpace(text.String()), secs: time.Since(start).Seconds(), err: runErr}
	}
}

// btwRequest is what a side question sends. Plan mode is the guarantee that
// it cannot edit files while the main turn is mid-change; low effort keeps it
// quick, since the point is a short answer.
func btwRequest(prompt, model, dir string) agent.Request {
	return agent.Request{
		Prompt: prompt,
		Model:  model,
		Dir:    dir,
		Mode:   agent.ModePlan,
		Effort: agent.EffortLow,
	}
}

// btwTranscript is the tail of the conversation, in the same shape the
// handoff uses, but smaller — a side question needs context, not history.
func (m *Model) btwTranscript() string {
	var lines []string
	userTurns := 0
	for i := len(m.entries) - 1; i >= 0 && userTurns < btwTurns; i-- {
		e := m.entries[i]
		switch e.kind {
		case kUser:
			if strings.HasPrefix(e.text, "btw:") {
				continue // side questions are not the conversation
			}
			userTurns++
			lines = append([]string{"USER: " + strings.Join(strings.Fields(e.text), " ")}, lines...)
		case kText:
			t := truncRight(strings.Join(strings.Fields(e.text), " "), 700)
			lines = append([]string{"NØX: " + t}, lines...)
		case kTool:
			// The point of /btw is asking about work in flight: without the
			// tool rows it sees a question and no answer to ask about.
			what := toolLabel(e.tool.Name) + " " + m.toolTarget(e.tool)
			what = truncRight(strings.Join(strings.Fields(what), " "), 160)
			status := "ok"
			switch {
			case e.isErr:
				status = "failed"
			case !e.done:
				status = "running"
			}
			line := "TOOL " + what + " · " + status
			if r := strings.TrimSpace(e.result); r != "" && status != "running" {
				line += " → " + truncRight(strings.Join(strings.Fields(r), " "), 220)
			}
			lines = append([]string{line}, lines...)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	s := strings.Join(lines, "\n")
	if m.busy {
		s += "\n\n(The agent is still working on the last USER message; the tool rows above are what it has done so far.)"
	}
	if len(s) > 8000 {
		s = "…" + s[len(s)-8000:]
	}
	return s
}

// onBtw shows the answer as a note, marked so it is clear it is not part of
// the conversation the main turn is building.
func (m *Model) onBtw(msg btwDoneMsg) {
	if msg.err != nil {
		m.errorf("btw: " + msg.err.Error())
		return
	}
	if msg.text == "" {
		m.errorf("btw: the model returned nothing")
		return
	}
	m.entries = append(m.entries, entry{
		kind: kBtw,
		text: msg.text,
		prov: "btw · " + fmtDur(time.Duration(msg.secs*float64(time.Second))),
	})
	m.follow = true
}
