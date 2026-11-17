package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Johnny008-creator/nox-local/internal/session"
)

// /find searches this conversation and scrolls to a hit; /search looks
// through every saved session (prompts, answers, files touched) and resumes
// the one you pick.

func entryText(e entry) string {
	switch e.kind {
	case kTool:
		in := ""
		for _, k := range []string{"file_path", "command", "pattern", "url", "query"} {
			if v, ok := e.tool.Input[k].(string); ok {
				in += " " + v
			}
		}
		return e.tool.Name + in + " " + e.result
	}
	return e.text
}

// snippet returns the text around the first match of q.
func snippet(s, q string, width int) string {
	s = strings.Join(strings.Fields(s), " ")
	i := strings.Index(strings.ToLower(s), strings.ToLower(q))
	if i < 0 {
		return truncRight(s, width)
	}
	start := max(0, i-width/3)
	r := []rune(s[start:])
	if start > 0 {
		r = append([]rune("…"), r...)
	}
	return truncRight(string(r), width)
}

func cmdFind(m *Model, q string) tea.Cmd {
	if q == "" {
		m.borrowPrompt("/find ")
		return nil
	}
	var items []pickItem
	lq := strings.ToLower(q)
	for i, e := range m.entries {
		if e.kind == kInfo {
			continue
		}
		if strings.Contains(strings.ToLower(entryText(e)), lq) {
			kind := map[entryKind]string{kUser: "you", kText: "NØX", kThinking: "thinking", kTool: toolLabel(e.tool.Name), kError: "error"}[e.kind]
			items = append(items, pickItem{value: strconv.Itoa(i), label: kind, desc: snippet(entryText(e), q, 90)})
		}
	}
	if len(items) == 0 {
		m.info("no match for " + q)
		return nil
	}
	m.picker = newPicker(fmt.Sprintf("Find “%s” · %d hits", q, len(items)), items, true, func(m *Model, it pickItem) tea.Cmd {
		i, _ := strconv.Atoi(it.value)
		m.scrollToEntry(i)
		return nil
	})
	return nil
}

// scrollToEntry moves the viewport so entry i is at the top.
func (m *Model) scrollToEntry(i int) {
	if i <= 0 || i >= len(m.entries) {
		m.vp.GotoTop()
		m.follow = false
		return
	}
	w := m.vp.Width()
	if w <= 0 {
		w = 80
	}
	saved := m.entries
	m.entries = saved[:i]
	before := m.renderEntries(w)
	m.entries = saved
	m.follow = false
	m.vp.SetContent(m.renderEntries(w))
	m.vp.SetYOffset(lipgloss.Height(before))
}

func cmdSearch(m *Model, q string) tea.Cmd {
	if q == "" {
		m.borrowPrompt("/search ")
		return nil
	}
	if m.store == nil {
		m.info("sessions are not saved in this mode")
		return nil
	}
	list, err := m.store.List("")
	if err != nil {
		m.errorf(err.Error())
		return nil
	}
	lq := strings.ToLower(q)
	var items []pickItem
	for _, s := range list {
		hit, files := "", map[string]bool{}
		for _, e := range s.Entries {
			if p, ok := e.ToolInput["file_path"].(string); ok && (e.ToolName == "Edit" || e.ToolName == "Write") {
				files[p] = true
			}
			txt := e.Text
			if e.Kind == session.KindTool {
				p, _ := e.ToolInput["file_path"].(string)
				c, _ := e.ToolInput["command"].(string)
				txt = e.ToolName + " " + p + " " + c
			}
			if hit == "" && strings.Contains(strings.ToLower(txt), lq) {
				hit = snippet(txt, q, 70)
			}
		}
		if hit == "" {
			continue
		}
		items = append(items, pickItem{value: s.ID, label: truncRight(s.Title(), 50),
			desc: fmt.Sprintf("%s · %s · %s", s.Updated.Format("Jan 2"), shortDir(s.Dir), hit)})
	}
	if len(items) == 0 {
		m.info("no saved session mentions " + q)
		return nil
	}
	m.picker = newPicker(fmt.Sprintf("Sessions mentioning “%s”", q), items, true, func(m *Model, it pickItem) tea.Cmd {
		s, err := m.store.Load(it.value)
		if err != nil {
			m.errorf(err.Error())
			return nil
		}
		m.save()
		m.resetConversation()
		m.restore(s)
		return nil
	})
	return nil
}

func shortDir(d string) string {
	parts := strings.Split(strings.ReplaceAll(d, "\\", "/"), "/")
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return d
}

// borrowPrompt puts a command into the prompt and keeps the draft that was
// there; it comes back after the command runs.
func (m *Model) borrowPrompt(cmd string) {
	if v := m.ta.Value(); v != "" && !strings.HasPrefix(v, "/") {
		m.stash = v
	}
	m.ta.SetValue(cmd)
	m.ta.CursorEnd()
}
