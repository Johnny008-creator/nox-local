package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// /twin splits the chat: the active provider on the left, everything other
// providers said (@mentions, the ultracode reviewer) on the right.

// isMine: entries that belong in the left pane.
func (m *Model) isMine(e entry) bool {
	return e.prov == "" || e.prov == m.prov.ID()
}

func (m *Model) hasOtherEntries() bool {
	for _, e := range m.entries {
		if !m.isMine(e) {
			return true
		}
	}
	return false
}

// otherProvider is who spoke last in the right pane.
func (m *Model) otherProvider() string {
	for i := len(m.entries) - 1; i >= 0; i-- {
		if e := m.entries[i]; !m.isMine(e) {
			return e.prov
		}
	}
	return ""
}

func (m *Model) paneHeader(id string, w int) string {
	role := "writer"
	if id != m.prov.ID() {
		role = "second opinion"
	}
	busy := ""
	if m.busy && m.turnProv != nil && m.turnProv.ID() == id {
		busy = " " + sLime.Render(spinFrames[m.frame%len(spinFrames)])
	}
	left := sTeal.Render("● ") + sBold.Render(id) + busy
	right := sDim.Render(role)
	gap := max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", gap) + right
}

// renderOtherPane shows the tail of the other providers' entries.
func (m *Model) renderOtherPane(w, h int) string {
	body := m.renderEntriesWhere(w, func(e entry) bool { return !m.isMine(e) })
	lines := strings.Split(body, "\n")
	if len(lines) > h {
		lines = lines[len(lines)-h:]
	}
	pane := lipgloss.JoinVertical(lipgloss.Left, m.paneHeader(m.otherProvider(), w), strings.Join(lines, "\n"))
	return lipgloss.NewStyle().Width(w).Height(h + 1).MaxHeight(h + 1).Render(pane)
}

func cmdTwin(m *Model, _ string) tea.Cmd {
	m.twin = !m.twin
	switch {
	case !m.twin:
		m.info("twin view off")
	case !m.hasOtherEntries():
		m.info("twin view on · the right pane fills when a side question answers (/btw …)")
	default:
		m.info("twin view on")
	}
	return nil
}
