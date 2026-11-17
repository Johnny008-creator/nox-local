package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Johnny008-creator/nox-local/internal/agent"
)

// todo is one item of the agent's task list (TodoWrite, same shape for the
// native loop and Claude Code).
type todo struct {
	Content, Active, Status string
}

// parseTodos reads a TodoWrite input.
func parseTodos(in map[string]any) []todo {
	items, _ := in["todos"].([]any)
	var out []todo
	for _, x := range items {
		t, ok := x.(map[string]any)
		if !ok {
			continue
		}
		c, _ := t["content"].(string)
		a, _ := t["activeForm"].(string)
		s, _ := t["status"].(string)
		if c == "" {
			continue
		}
		out = append(out, todo{c, a, s})
	}
	return out
}

// agentTodos converts the backend's task list into the TUI's rows.
func agentTodos(ts []agent.Todo) []todo {
	out := make([]todo, 0, len(ts))
	for _, t := range ts {
		if strings.TrimSpace(t.Content) == "" {
			continue
		}
		out = append(out, todo{Content: t.Content, Status: t.Status})
	}
	return out
}

func (m *Model) noteTodos(t agent.Tool) {
	if t.Name != "TodoWrite" {
		return
	}
	if ts := parseTodos(t.Input); ts != nil {
		m.todos = ts
	}
}

func todoCounts(ts []todo) (done, total int) {
	for _, t := range ts {
		if t.Status == "completed" {
			done++
		}
	}
	return done, len(ts)
}

// todoGlyph is ✓ done, ● working, ○ waiting.
func todoGlyph(status string, frame int) string {
	switch status {
	case "completed":
		return sTeal.Render("✓")
	case "in_progress":
		return pulse("●", frame)
	}
	return sDim.Render("○")
}

// renderTodoCard is the PLAN sidebar card.
func (m *Model) renderTodoCard(cw int) []string {
	done, total := todoCounts(m.todos)
	lines := []string{sDim.Render(spaced("PLAN")) + sDim.Render(fmt.Sprintf("  %d/%d", done, total))}
	lines = append(lines, progressBar(float64(done)/float64(max(total, 1)), cw, m.frame))
	shown := 0
	for _, t := range m.todos {
		if shown == 7 {
			lines = append(lines, sDim.Render(fmt.Sprintf("+%d more", total-shown)))
			break
		}
		st := sMuted
		switch t.Status {
		case "completed":
			st = sDim.Strikethrough(true)
		case "in_progress":
			st = sText
		}
		lines = append(lines, todoGlyph(t.Status, m.frame)+" "+st.Render(truncRight(t.Content, cw-2)))
		shown++
	}
	return lines
}

// renderProgress is the "what is NØX doing" panel shown above the spinner
// while a turn with a plan runs: the current step and what is next.
func (m *Model) renderProgress(w int) string {
	if len(m.todos) == 0 {
		return ""
	}
	done, total := todoCounts(m.todos)
	if done == total {
		return ""
	}
	var cur, next string
	for _, t := range m.todos {
		switch {
		case t.Status == "in_progress" && cur == "":
			cur = t.Active
			if cur == "" {
				cur = t.Content
			}
		case t.Status == "pending" && next == "":
			next = t.Content
		}
	}
	bar := progressBar(float64(done)/float64(total), 12, m.frame)
	s := bar + sDim.Render(fmt.Sprintf(" %d/%d ", done, total))
	if cur != "" {
		s += pulse("●", m.frame) + " " + sText.Render(cur)
	}
	if next != "" {
		s += sDim.Render("  ·  next: " + next)
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(truncRight(s, w))
}

// progressBar draws ━ segments: an aurora gradient for the done part and a
// soft shimmer travelling along it while work continues.
func progressBar(ratio float64, w, frame int) string {
	if w <= 0 {
		return ""
	}
	filled := min(int(ratio*float64(w)+0.5), w)
	done := ""
	if filled > 0 {
		done = shimmer(strings.Repeat("━", filled), frame, aurora)
	}
	return done + fg(cCard).Render(strings.Repeat("━", w-filled))
}
