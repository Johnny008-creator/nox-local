package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// The TUI captures the mouse (wheel scrolling, clicks), which switches the
// terminal's own text selection off. selection brings it back: drag with
// the left button to select what is on screen, release to copy it.
type selection struct {
	down           bool // the left button is held
	moved          bool // the drag covered more than the first cell
	ax, ay, bx, by int  // anchor and current cell
}

func (s *selection) press(x, y int) {
	*s = selection{down: true, ax: x, ay: y, bx: x, by: y}
}

func (s *selection) drag(x, y int) {
	if !s.down {
		return
	}
	if x != s.ax || y != s.ay {
		s.moved = true
	}
	s.bx, s.by = x, y
}

// span returns the selected cells of row y as [from, to), reading order
// like a terminal: the first row from the start cell, the last row up to
// the end cell, whole rows in between.
func (s *selection) span(y, width int) (from, to int, ok bool) {
	x0, y0, x1, y1 := s.ax, s.ay, s.bx, s.by
	if y1 < y0 || (y1 == y0 && x1 < x0) {
		x0, y0, x1, y1 = x1, y1, x0, y0
	}
	if !s.moved || y < y0 || y > y1 {
		return 0, 0, false
	}
	from, to = 0, width
	if y == y0 {
		from = x0
	}
	if y == y1 {
		to = x1 + 1
	}
	from, to = max(from, 0), min(to, width)
	return from, to, from < to
}

// paintSelection remembers the rendered screen (so a release can copy from
// it) and shows the selected cells in reverse video.
func (m *Model) paintSelection(out string) string {
	lines := strings.Split(out, "\n")
	m.screen = append(m.screen[:0], lines...)
	if !m.sel.moved {
		return out
	}
	for y, line := range lines {
		w := ansi.StringWidth(line)
		from, to, ok := m.sel.span(y, w)
		if !ok {
			continue
		}
		lines[y] = ansi.Cut(line, 0, from) + "\x1b[0;7m" + ansi.Strip(ansi.Cut(line, from, to)) + "\x1b[0m" + ansi.Cut(line, to, w)
	}
	return strings.Join(lines, "\n")
}

// selectedText is the selection as plain text, without trailing spaces.
func (m *Model) selectedText() string {
	var rows []string
	for y, line := range m.screen {
		from, to, ok := m.sel.span(y, ansi.StringWidth(line))
		if !ok {
			if len(rows) > 0 && y <= max(m.sel.ay, m.sel.by) {
				rows = append(rows, "") // an empty row inside the selection
			}
			continue
		}
		rows = append(rows, strings.TrimRight(ansi.Strip(ansi.Cut(line, from, to)), " "))
	}
	return strings.TrimSpace(strings.Join(rows, "\n"))
}
