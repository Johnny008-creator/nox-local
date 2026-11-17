package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Johnny008-creator/N0X-Local/internal/agent"
)

const reviewPrompt = `ULTRACODE REVIEW. Another NØX agent just finished the task above and changed these files:
%s
Review those changes critically and independently: correctness, edge cases, missed requirements, broken builds or tests, style that clashes with the codebase. Read the files and run read-only checks (build, tests, linters) where useful. Do not edit anything.
` + reviewContract

// mention splits "@codex do X" into the provider and the rest.
func (m *Model) mention(text string) (agent.Provider, string, bool) {
	if !strings.HasPrefix(text, "@") {
		return nil, text, false
	}
	word, rest, _ := strings.Cut(text[1:], " ")
	p := m.provider(strings.ToLower(word))
	if p == nil {
		return nil, text, false
	}
	return p, strings.TrimSpace(rest), true
}

// sendMention runs one turn on another provider without switching to it.
func (m *Model) sendMention(p agent.Provider, text string) tea.Cmd {
	if !p.Available() {
		m.errorf(p.Name() + " is not set up · " + setupHint(p.ID()))
		return nil
	}
	if text == "" {
		return m.switchProvider(p.ID())
	}
	if m.busy {
		m.queue = append(m.queue, "@"+p.ID()+" "+text)
		m.info("queued for " + p.Name())
		return nil
	}
	return m.startTurnOn(p, text, m.mode, false)
}

// afterTurn chains automatic follow-ups: the ultracode review pass and
// auto-compact when the context is nearly full.
func (m *Model) afterTurn() tea.Cmd {
	if m.lastFailed {
		return nil
	}
	return nil
}

// reviewAsk is the pending "run the ultracode review?" question.
type reviewAsk struct {
	prov  agent.Provider
	files string
}

// answerReview handles the first key after the question: y starts the
// review (handled = true), anything else drops the offer and is then
// processed as a normal key.
func (m *Model) answerReview(key string) (tea.Cmd, bool) {
	ask := m.reviewAsk
	m.reviewAsk = nil
	if (key != "y" && key != "Y") || m.ta.Value() != "" {
		m.info("ultracode: review skipped · /review runs it later")
		return nil, false
	}
	if m.busy {
		m.info("ultracode: a turn is running · use /review when it is done")
		return nil, true
	}
	if m.w >= 150 {
		m.twin = true // writer and reviewer side by side
	}
	return m.startTurnOn(ask.prov, strings.Replace(reviewPrompt, "%s", ask.files, 1), agent.ModePlan, true), true
}
