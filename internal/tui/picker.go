package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Johnny008-creator/N0X-Local/internal/agent"
)

// pickItem is one row of a picker. label may be pre-styled; render overrides
// it per frame (used for animated effort names).
type pickItem struct {
	value   string
	label   string
	desc    string
	current bool
	dim     bool
	render  func(frame int) string
}

// picker is the overlay list opened by commands like /effort and /model:
// ↑↓ move, type to filter, enter picks, esc closes.
type picker struct {
	title   string
	items   []pickItem
	idx     int
	filter  string
	search  bool   // typing filters the list
	loading string // provider id we are waiting on for items
	pick    func(m *Model, it pickItem) tea.Cmd
	move    func(m *Model, it pickItem) // live preview of the highlighted item
	cancel  func(m *Model)              // undo the preview on esc
}

const pickRows = 10

func (p *picker) visible() []pickItem {
	if p.filter == "" {
		return p.items
	}
	f := strings.ToLower(p.filter)
	var out []pickItem
	for _, it := range p.items {
		if strings.Contains(strings.ToLower(it.value+" "+ansiStrip(it.label)+" "+it.desc), f) {
			out = append(out, it)
		}
	}
	return out
}

func newPicker(title string, items []pickItem, search bool, pick func(*Model, pickItem) tea.Cmd) *picker {
	p := &picker{title: title, items: items, search: search, pick: pick}
	for i, it := range items {
		if it.current {
			p.idx = i
		}
	}
	return p
}

func (m *Model) pickerKey(k tea.KeyPressMsg) tea.Cmd {
	p := m.picker
	defer func() {
		if m.picker == p && p.move != nil {
			if vis := p.visible(); len(vis) > 0 {
				p.move(m, vis[min(p.idx, len(vis)-1)])
			}
		}
	}()
	vis := p.visible()
	switch k.String() {
	case "esc", "ctrl+c":
		m.picker = nil
		if p.cancel != nil {
			p.cancel(m)
		}
		return nil
	case "up", "ctrl+k":
		if len(vis) > 0 {
			p.idx = (p.idx - 1 + len(vis)) % len(vis)
		}
	case "down", "ctrl+j", "tab":
		if len(vis) > 0 {
			p.idx = (p.idx + 1) % len(vis)
		}
	case "pgup":
		p.idx = max(p.idx-pickRows, 0)
	case "pgdown":
		p.idx = min(p.idx+pickRows, max(len(vis)-1, 0))
	case "home":
		p.idx = 0
	case "end":
		p.idx = max(len(vis)-1, 0)
	case "enter":
		if len(vis) == 0 {
			return nil
		}
		it := vis[min(p.idx, len(vis)-1)]
		m.picker = nil
		return p.pick(m, it)
	case "backspace":
		if p.filter != "" {
			_, n := utf8.DecodeLastRuneInString(p.filter)
			p.filter = p.filter[:len(p.filter)-n]
			p.idx = 0
		}
	default:
		if p.search {
			if t := k.Text; t != "" {
				p.filter += t
				p.idx = 0
			}
		}
	}
	return nil
}

func (m *Model) renderPicker(w int) string {
	p := m.picker
	cw := w - 4
	head := sTeal.Bold(true).Render(p.title)
	if p.search {
		f := p.filter
		if f == "" {
			f = sDim.Render("type to filter")
		} else {
			f = sText.Render(f) + sLime.Render("▏")
		}
		head += sDim.Render("  ⌕ ") + f
	}
	rows := []string{head}
	vis := p.visible()
	switch {
	case p.loading != "":
		rows = append(rows, sLime.Render(spinFrames[m.frame%len(spinFrames)])+sDim.Render(" loading models…"))
	case len(vis) == 0:
		rows = append(rows, sDim.Render("no matches"))
	}
	start := 0
	if p.idx >= pickRows {
		start = p.idx - pickRows + 1
	}
	for i := start; i < len(vis) && i < start+pickRows; i++ {
		it := vis[i]
		cursor, mark := "  ", "  "
		if i == p.idx {
			cursor = sTeal.Render("❯ ")
		}
		if it.current {
			mark = sLime.Render("● ")
		}
		label := it.label
		if it.render != nil {
			label = it.render(m.frame)
		}
		switch {
		case i == p.idx && it.render == nil:
			label = sBold.Render(ansiStrip(label))
		case it.dim:
			label = sDim.Render(ansiStrip(label))
		}
		left := cursor + mark + label
		desc := ""
		if it.desc != "" {
			room := cw - lipgloss.Width(left) - 2
			if room > 6 {
				desc = sDim.Render(truncRight(it.desc, room))
			}
		}
		gap := max(cw-lipgloss.Width(left)-lipgloss.Width(desc), 2)
		rows = append(rows, left+strings.Repeat(" ", gap)+desc)
	}
	foot := "↑↓ move · enter select · esc close"
	if len(vis) > pickRows {
		foot = fmt.Sprintf("%d/%d · ", p.idx+1, len(vis)) + foot
	}
	rows = append(rows, sDim.Render(foot))
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cTeal).Padding(0, 1).Width(w).Render(strings.Join(rows, "\n"))
}

// ---------- pickers ----------

func (m *Model) openEffortPicker() tea.Cmd {
	descs := map[agent.Effort]string{
		agent.EffortLow:    "answer directly, fastest",
		agent.EffortMedium: "think briefly first",
		agent.EffortHigh:   "longer reasoning, best on hard tasks",
	}
	var items []pickItem
	for _, e := range agent.Efforts {
		e := e
		items = append(items, pickItem{value: e.String(), label: e.String(), desc: descs[e], current: e == m.effort,
			render: func(f int) string { return effortLabel(e, f) }})
	}
	m.picker = newPicker("Effort", items, false, func(m *Model, it pickItem) tea.Cmd {
		m.effort, _ = agent.ParseEffort(it.value)
		return nil
	})
	return nil
}

func (m *Model) openModePicker() tea.Cmd {
	descs := map[agent.Mode]string{
		agent.ModeManual: "asks before edits and commands",
		agent.ModePlan:   "read-only, writes a plan",
		agent.ModeAuto:   "edits and runs, asks for risky things",
		agent.ModeYolo:   "never asks",
	}
	var items []pickItem
	for _, md := range agent.Modes {
		md := md
		items = append(items, pickItem{value: md.String(), label: md.String(), desc: descs[md], current: md == m.mode,
			render: func(int) string { return modeLabel(md) }})
	}
	m.picker = newPicker("Mode", items, false, func(m *Model, it pickItem) tea.Cmd {
		m.mode, _ = agent.ParseMode(it.value)
		return nil
	})
	return nil
}

func (m *Model) openModelPicker() tea.Cmd {
	ms, ok := m.modelCache[m.prov.ID()]
	if !ok {
		m.picker = &picker{title: "Model · " + m.prov.Name(), loading: m.prov.ID(), search: true,
			pick: func(*Model, pickItem) tea.Cmd { return nil }}
		return fetchModels(m.prov)
	}
	cur := m.currentModel()
	var items []pickItem
	for _, mi := range ms {
		var parts []string
		if mi.Context > 0 {
			parts = append(parts, fmtK(mi.Context)+" ctx")
		}
		if mi.Note != "" {
			parts = append(parts, mi.Note)
		}
		label := mi.ID
		if mi.Name != "" && mi.Name != mi.ID {
			label = mi.Name + sDim.Render(" "+mi.ID)
		}
		items = append(items, pickItem{value: mi.ID, label: label, desc: strings.Join(parts, " · "), current: mi.ID == cur})
	}
	m.picker = newPicker("Model · "+m.prov.Name(), items, len(items) > 6,
		func(m *Model, it pickItem) tea.Cmd {
			m.setModel(it.value)
			m.info("model → " + it.value)
			return nil
		})
	return nil
}

func (m *Model) openCommandPicker() tea.Cmd {
	var items []pickItem
	for _, c := range commands {
		desc := c.desc
		if c.soon != "" {
			desc += " · " + c.soon
		}
		items = append(items, pickItem{value: c.name, label: "/" + c.name, desc: desc, dim: c.soon != ""})
	}
	m.picker = newPicker("Commands", items, true, func(m *Model, it pickItem) tea.Cmd {
		return m.runCommand("/" + it.value)
	})
	return nil
}

func ansiStrip(s string) string { return ansi.Strip(s) }
