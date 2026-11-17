package tui

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"
)

// Toasts are short notices in the top-right corner (context compacted,
// provider failover, hook failed…). They slide in on a spring, show a
// shrinking life bar and slide out again, drawn as a layer over the screen
// so the conversation underneath never reflows.

const (
	toastLife = 4 * time.Second
	toastMaxW = 46
	toastFPS  = 9 // matches the 110 ms tick
)

type toastT struct {
	text      string
	born      time.Time
	off, vel  float64 // horizontal offset (cells pushed off-screen) and its velocity
	leaving   bool
	spring    harmonica.Spring
	targetOff float64
}

func (m *Model) toast(text string) {
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return
	}
	if m.plain { // no animation, but the message must still be seen
		m.info(text)
		return
	}
	if len(m.toasts) >= 3 {
		m.toasts = m.toasts[1:]
	}
	w := toastWidth(text)
	m.toasts = append(m.toasts, toastT{text: text, born: time.Now(), off: float64(w + 2),
		spring: harmonica.NewSpring(harmonica.FPS(toastFPS), 9.0, 0.8)})
}

func toastWidth(text string) int { return min(lipgloss.Width(text)+4, toastMaxW) }

// stepToasts advances the springs; called on every tick.
func (m *Model) stepToasts() {
	keep := m.toasts[:0]
	for _, t := range m.toasts {
		w := float64(toastWidth(t.text) + 2)
		if !t.leaving && time.Since(t.born) > toastLife {
			t.leaving = true
		}
		t.targetOff = 0
		if t.leaving {
			t.targetOff = w
		}
		t.off, t.vel = t.spring.Update(t.off, t.vel, t.targetOff)
		if t.leaving && t.off >= w-0.5 {
			continue // gone
		}
		keep = append(keep, t)
	}
	m.toasts = keep
}

func (m *Model) renderToast(t toastT) string {
	w := toastWidth(t.text)
	life := 1 - float64(time.Since(t.born))/float64(toastLife)
	life = max(0, min(1, life))
	barW := int(life*float64(w-2) + 0.5)
	bar := shimmer(strings.Repeat("━", barW), m.frame, aurora) + strings.Repeat(" ", w-2-barW)
	body := sText.Render(truncRight(t.text, w-2)) + "\n" + bar
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cTeal).
		Background(cPanel).Padding(0, 0).Width(w).Render(body)
}

// overlayToasts draws the toasts over the finished frame.
func (m *Model) overlayToasts(base string) string {
	if len(m.toasts) == 0 || m.w < 40 {
		return base
	}
	c := lipgloss.NewCanvas(m.w, max(m.h, lipgloss.Height(base)))
	c.Compose(lipgloss.NewLayer(base))
	y := 1
	for i, t := range m.toasts {
		box := m.renderToast(t)
		x := m.w - lipgloss.Width(box) - 1 + int(t.off+0.5)
		if x >= m.w {
			continue
		}
		c.Compose(lipgloss.NewLayer(clipLeftCols(box, m.w-x)).X(x).Y(y).Z(10 + i))
		y += lipgloss.Height(box)
	}
	return c.Render()
}

// clipLeftCols keeps the first n columns of every line (the part of a
// sliding toast that is already on screen).
func clipLeftCols(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return lipgloss.NewStyle().MaxWidth(n).Render(s)
}
