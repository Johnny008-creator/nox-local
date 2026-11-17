package tui

import (
	"strings"
	"testing"
	"time"
)

func TestFmtDur(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                          "0s",
		400 * time.Millisecond:     "0.4s",
		59 * time.Second:           "59s",
		60 * time.Second:           "1m",
		62 * time.Second:           "1m 2s",
		1500 * time.Second:         "25m",
		3600 * time.Second:         "1h",
		3725 * time.Second:         "1h 2m",
		26*time.Hour + time.Minute: "26h 1m",
	} {
		if got := fmtDur(d); got != want {
			t.Errorf("fmtDur(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestMouseSelection(t *testing.T) {
	m := &Model{}
	screen := "\x1b[1mhello\x1b[0m world   \n\nsecond \x1b[32mline\x1b[0m here\nlast"
	m.paintSelection(screen)

	// a click without a drag selects nothing
	m.sel.press(2, 0)
	if m.sel.moved || m.selectedText() != "" {
		t.Fatal("a plain click must not select")
	}
	// drag from "world" on row 0 to "line" on row 2
	m.sel.drag(10, 2)
	if got, want := m.selectedText(), "llo world\n\nsecond line"; got != want {
		t.Errorf("selected %q, want %q", got, want)
	}
	painted := m.paintSelection(screen)
	if !strings.Contains(painted, "\x1b[0;7mllo world") || strings.Count(painted, "\n") != 3 {
		t.Errorf("painted = %q", painted)
	}
	// dragging backwards selects the same cells
	m.sel.press(10, 2)
	m.sel.drag(2, 0)
	if got := m.selectedText(); got != "llo world\n\nsecond line" {
		t.Errorf("backwards: %q", got)
	}
	// within one row
	m.sel.press(7, 2)
	m.sel.drag(10, 2)
	if got := m.selectedText(); got != "line" {
		t.Errorf("one row: %q", got)
	}
}
