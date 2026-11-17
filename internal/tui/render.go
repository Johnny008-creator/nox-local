package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Johnny008-creator/nox-local/internal/agent"
	"github.com/Johnny008-creator/nox-local/internal/agent/native"
)

const (
	sideW        = 32  // sidebar width incl. card borders
	sideMinWidth = 100 // below this the sidebar hides
)

var (
	spinFrames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	verbs      = []string{"Forging", "Tempering", "Wiring", "Hammering", "Untangling", "Annealing", "Riveting", "Smelting"}
	cardStyle  lipgloss.Style // set by applyTheme
)

func (m *Model) View() tea.View {
	if m.w == 0 {
		return tea.NewView("")
	}
	if m.diff != nil && m.approval == nil {
		v := tea.NewView(m.overlayToasts(m.renderDiffView()))
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		v.WindowTitle = "NØX · diff"
		return v
	}
	inner := m.w - 2
	mainW := inner
	showSide := m.w >= sideMinWidth
	if showSide {
		mainW = inner - sideW - 2
	}

	input := m.renderInput(inner)
	status := m.renderStatus(inner)
	var sugg string
	switch {
	case m.approval != nil:
		sugg = m.renderApproval(inner)
	case m.picker != nil:
		sugg = m.renderPicker(inner)
	default:
		if sugg = m.renderAtSuggestions(inner); sugg == "" {
			sugg = m.renderSuggestions(inner)
		}
	}
	if room := m.h - lipgloss.Height(input) - lipgloss.Height(status) - 2; sugg != "" && lipgloss.Height(sugg) > room {
		ls := strings.Split(sugg, "\n") // tiny terminal: keep the end (keys / current row)
		sugg = strings.Join(ls[max(0, len(ls)-max(room, 1)):], "\n")
	}
	bodyH := m.h - lipgloss.Height(input) - lipgloss.Height(status)
	if sugg != "" {
		bodyH -= lipgloss.Height(sugg)
	}

	header := m.renderHeader(mainW)
	spin := ""
	if m.busy {
		spin = m.renderSpinner(mainW)
		if pr := m.renderProgress(mainW); pr != "" {
			spin = pr + "\n" + spin
		}
	}
	m.vpTop = lipgloss.Height(header)
	vpH := bodyH - lipgloss.Height(header)
	if spin != "" {
		vpH -= lipgloss.Height(spin)
	}
	twin := m.twin && m.hasOtherEntries() && mainW >= 80
	paneW := mainW
	if twin {
		paneW = (mainW - 3) / 2
		vpH-- // pane header
	}
	m.vp.SetWidth(paneW)
	m.vp.SetHeight(max(vpH, 1))
	content := ""
	if twin {
		content = m.renderEntriesWhere(paneW, m.isMine)
	} else {
		content = m.renderEntries(mainW)
	}
	if content != m.vpContent { // resetting the viewport re-splits every line
		m.vpContent = content
		m.vp.SetContent(content)
	}
	if m.follow {
		m.vp.GotoBottom()
	}

	chat := m.vp.View()
	if twin {
		leftPane := lipgloss.JoinVertical(lipgloss.Left, m.paneHeader(m.prov.ID(), paneW), chat)
		leftPane = lipgloss.NewStyle().Width(paneW).Height(vpH + 1).MaxHeight(vpH + 1).Render(leftPane)
		rightPane := m.renderOtherPane(paneW, vpH)
		sep := fg(cCard).Render(strings.TrimSuffix(strings.Repeat("│\n", vpH+1), "\n"))
		chat = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", sep, " ", rightPane)
	}
	left := lipgloss.JoinVertical(lipgloss.Left, header, chat)
	if spin != "" {
		left = lipgloss.JoinVertical(lipgloss.Left, left, spin)
	}
	left = lipgloss.NewStyle().Width(mainW).Height(bodyH).MaxHeight(bodyH).Render(left)

	body := left
	if showSide {
		side := lipgloss.NewStyle().Height(bodyH).MaxHeight(bodyH).Render(m.renderSidebar(bodyH))
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", side)
	}

	parts := []string{body}
	if sugg != "" {
		parts = append(parts, sugg)
	}
	parts = append(parts, input, status)
	out := lipgloss.NewStyle().PaddingLeft(1).Render(lipgloss.JoinVertical(lipgloss.Left, parts...))
	out = m.overlayToasts(out)

	out = m.paintSelection(out)
	v := tea.NewView(out)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.WindowTitle = "NØX · " + filepath.Base(m.dir)
	switch {
	case m.approval != nil:
		v.WindowTitle = "NØX · needs approval"
		v.ProgressBar = tea.NewProgressBar(tea.ProgressBarWarning, 100)
	case m.busy:
		v.WindowTitle = "NØX · working · " + filepath.Base(m.dir)
		v.ProgressBar = tea.NewProgressBar(tea.ProgressBarIndeterminate, 0)
	}
	return v
}

// ---------- header ----------

func (m *Model) renderHeader(w int) string {
	logo := gradient("N  Ø  X", aurora, 0, false, true)
	stops, cyclic, off := aurora, false, 0
	if fx := fxStops(m.effort); fx != nil {
		stops, cyclic, off = fx, true, -m.frame
	}
	rule := gradient(strings.Repeat("━", 19), stops, off, cyclic, false)
	meta := m.providerPill() + " " + sBold.Render(m.modelName()) + sDim.Render(" · llama.cpp · v"+m.version)
	cwd := sDim.Render(truncLeft(m.dir, w))
	return lipgloss.JoinVertical(lipgloss.Left, logo, rule, meta, cwd, "")
}

// ---------- conversation ----------

func (m *Model) renderEntries(w int) string {
	return m.renderEntriesWhere(w, func(entry) bool { return true })
}

// renderEntriesWhere renders the entries keep accepts (used by /twin).
func (m *Model) renderEntriesWhere(w int, keep func(entry) bool) string {
	if len(m.entries) == 0 {
		return m.renderSplash(w)
	}
	var out []string
	rows := make([]int, len(m.entries))
	line := 0
	for i, e := range m.entries {
		rows[i] = -1
		if !keep(e) {
			continue
		}
		// Finished rows come from the cache; text has its own (markdown.go)
		// and running rows animate, so those are rendered every frame.
		cacheable := e.kind != kText && (e.done || (e.kind != kTool && e.kind != kThinking))
		key := rowKey{w: w, gen: m.renderGen, done: e.done, isErr: e.isErr, expanded: e.expanded, children: e.children,
			lastChild: e.lastChild, prov: m.prov.ID(), textLen: len(e.text), resultLen: len(e.result), dur: e.dur, first: i == 0}
		if cacheable && e.row != "" && e.rowKey == key {
			rows[i] = line
			if strings.HasPrefix(e.row, "\n") {
				rows[i]++
			}
			line += e.rowH
			out = append(out, e.row)
			continue
		}
		var s string
		switch e.kind {
		case kUser:
			s = m.renderUser(e, w)
			if i > 0 {
				s = "\n" + s
			}
		case kText:
			s = m.renderText(i, w)
		case kThinking:
			if e.done {
				s = sDim.Italic(true).Render("∴ thought for " + fmtDur(e.dur))
			} else {
				s = sDim.Italic(true).Render("∴ thinking…")
			}
		case kTool:
			s = m.renderTool(e, w)
		case kInfo:
			s = sDim.Width(w).Render("· " + e.text)
		case kError:
			s = sErr.Width(w).Render("✗ " + e.text)
		case kCompact:
			s = m.renderCompact(e, w)
		case kBtw:
			// A side answer reads as a reply, but its heading says where it
			// came from: it is not part of the turn's conversation.
			s = sDim.Render("╭ "+e.prov) + "\n" + m.renderBtwBody(e, w)
		}
		rows[i] = line
		if strings.HasPrefix(s, "\n") {
			rows[i]++
		}
		h := lipgloss.Height(s)
		if cacheable && s != "" {
			m.entries[i].row, m.entries[i].rowH, m.entries[i].rowKey = s, h, key
		}
		line += h
		out = append(out, s)
	}
	m.entryRows = rows
	return strings.Join(out, "\n")
}

// renderUser draws the prompt block with the aurora bar on its left edge.
func (m *Model) renderUser(e entry, w int) string {
	text := e.text
	if e.prov != "" && e.prov != m.prov.ID() {
		text = "→ " + e.prov + "  " + text
	}
	body := lipgloss.NewStyle().Width(w-2).Background(cPanel).Foreground(cWhite).Padding(0, 1).Render(text)
	lines := strings.Split(body, "\n")
	pal := lipgloss.Blend1D(max(len(lines), 2), aurora...)
	for i, l := range lines {
		lines[i] = fg(pal[i]).Render("▎") + l
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderTool(e entry, w int) string {
	var status string
	switch {
	case !e.done:
		status = sLime.Render(spinFrames[m.frame%len(spinFrames)])
	case e.isErr:
		status = sErr.Render("✗")
	default:
		status = sTeal.Render("✓") + sDim.Render(" "+fmtDur(e.dur))
	}
	name := toolLabel(e.tool.Name)
	tgt := m.toolTarget(e.tool)
	head := sTeal.Render("◆") + " " + sText.Bold(true).Render(name)
	room := w - lipgloss.Width(head) - lipgloss.Width(status) - 3
	if tgt != "" && room > 4 {
		head += " " + sDim.Render(truncRight(tgt, room))
	}
	head += " " + status

	if e.children > 0 {
		head += "\n" + sDim.Render(fmt.Sprintf("  ↳ %s · %s", plural(e.children, "step"), truncRight(e.lastChild, w-20)))
	}
	if !e.done {
		return head
	}
	lines := 4
	if e.expanded {
		lines = 80
	}
	if e.isErr {
		return head + "\n" + m.preview(e.result, w, max(lines-1, 3), sErr)
	}
	switch e.tool.Name {
	case "Edit", "MultiEdit", "Write":
		if d := diffFor(e.tool); len(d) > 0 {
			out := head + "\n" + m.renderDiffCard(e.tool, d, w)
			if n := lspErrors(e.result); n > 0 {
				out += "\n" + sErr.Render(fmt.Sprintf("  ⚠ %s after this edit (LSP)", plural(n, "error")))
			} else if strings.Contains(e.result, "LSP: no errors") {
				out += "\n" + sDim.Render("  ✓ LSP: no errors")
			}
			return out
		}
	case "Read":
		if e.expanded {
			return head + "\n" + m.preview(e.result, w, lines, sDim)
		}
		return head + "\n" + sDim.Render("  ⎿ "+plural(lineCount(e.result), "line"))
	case "Grep", "Glob":
		return head + "\n" + sDim.Render("  ⎿ "+plural(nonEmptyLines(e.result), "result"))
	case "TodoWrite":
		if ts := parseTodos(e.tool.Input); len(ts) > 0 {
			done, total := todoCounts(ts)
			return head + "\n" + sDim.Render(fmt.Sprintf("  ⎿ plan %d/%d", done, total))
		}
		return head
	}
	if strings.TrimSpace(e.result) == "" {
		return head
	}
	out := head + "\n" + m.preview(e.result, w, lines, sDim)
	if !e.expanded && lineCount(e.result) > lines {
		out += sDim.Render("  (click to expand)")
	}
	return out
}

func (m *Model) preview(s string, w, maxLines int, st lipgloss.Style) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	extra := 0
	if len(lines) > maxLines {
		extra = len(lines) - maxLines
		lines = lines[:maxLines]
	}
	for i, l := range lines {
		prefix := "    "
		if i == 0 {
			prefix = "  ⎿ "
		}
		lines[i] = st.Render(prefix + truncRight(strings.ReplaceAll(l, "\t", "  "), w-4))
	}
	if extra > 0 {
		lines = append(lines, sDim.Render("    … +"+plural(extra, "line")))
	}
	return strings.Join(lines, "\n")
}

type diffLine struct {
	sign rune
	text string
}

// diffFor turns Edit / MultiEdit / Write inputs into display lines.
func diffFor(t agent.Tool) []diffLine {
	var out []diffLine
	pair := func(old, new string) {
		var a, b []string
		if old != "" {
			a = strings.Split(old, "\n")
		}
		if new != "" {
			b = strings.Split(new, "\n")
		}
		out = append(out, trimContext(lineDiff(a, b), 2)...)
	}
	switch t.Name {
	case "Edit":
		o, _ := t.Input["old_string"].(string)
		n, _ := t.Input["new_string"].(string)
		pair(o, n)
	case "MultiEdit":
		edits, _ := t.Input["edits"].([]any)
		for _, x := range edits {
			if ed, ok := x.(map[string]any); ok {
				o, _ := ed["old_string"].(string)
				n, _ := ed["new_string"].(string)
				pair(o, n)
			}
		}
	case "Write":
		c, _ := t.Input["content"].(string)
		pair("", c)
	}
	return out
}

// lineDiff compares two line lists: unchanged lines are ' ', the rest '-'
// (old) then '+' (new) per change, so an edit whose old_string carries
// context shows what really changed. Very large inputs fall back to all
// '-' then all '+'.
func lineDiff(a, b []string) []diffLine {
	var out []diffLine
	p := 0
	for p < len(a) && p < len(b) && a[p] == b[p] {
		out = append(out, diffLine{' ', a[p]})
		p++
	}
	ea, eb := len(a), len(b)
	for ea > p && eb > p && a[ea-1] == b[eb-1] {
		ea--
		eb--
	}
	ma, mb := a[p:ea], b[p:eb]
	if len(ma)*len(mb) > 250_000 {
		for _, l := range ma {
			out = append(out, diffLine{'-', l})
		}
		for _, l := range mb {
			out = append(out, diffLine{'+', l})
		}
	} else {
		// lcs[i][j]: longest common subsequence of ma[i:] and mb[j:]
		lcs := make([][]int, len(ma)+1)
		for i := range lcs {
			lcs[i] = make([]int, len(mb)+1)
		}
		for i := len(ma) - 1; i >= 0; i-- {
			for j := len(mb) - 1; j >= 0; j-- {
				if ma[i] == mb[j] {
					lcs[i][j] = lcs[i+1][j+1] + 1
				} else {
					lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
				}
			}
		}
		var dels, adds []diffLine
		flush := func() {
			out = append(append(out, dels...), adds...)
			dels, adds = dels[:0], adds[:0]
		}
		i, j := 0, 0
		for i < len(ma) || j < len(mb) {
			switch {
			case i < len(ma) && j < len(mb) && ma[i] == mb[j]:
				flush()
				out = append(out, diffLine{' ', ma[i]})
				i++
				j++
			case j >= len(mb) || (i < len(ma) && lcs[i+1][j] >= lcs[i][j+1]):
				dels = append(dels, diffLine{'-', ma[i]})
				i++
			default:
				adds = append(adds, diffLine{'+', mb[j]})
				j++
			}
		}
		flush()
	}
	for k := ea; k < len(a); k++ {
		out = append(out, diffLine{' ', a[k]})
	}
	return out
}

// trimContext keeps at most n unchanged lines before the first change and
// after the last one.
func trimContext(d []diffLine, n int) []diffLine {
	first, last := -1, -1
	for i, l := range d {
		if l.sign != ' ' {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return nil // nothing changed
	}
	return d[max(0, first-n):min(len(d), last+n+1)]
}

func (m *Model) renderDiffCard(t agent.Tool, d []diffLine, w int) string {
	var add, del int
	for _, l := range d {
		switch l.sign {
		case '+':
			add++
		case '-':
			del++
		}
	}
	path, _ := t.Input["file_path"].(string)
	inner := w - 2
	head := sDim.Render(" "+filepath.Base(path)+"  ") + fg(cAdd).Render(fmt.Sprintf("+%d", add)) + " " + fg(cDel).Render(fmt.Sprintf("−%d", del))
	lex := lexerFor(path)
	lines := make([]dline, len(d))
	for i, l := range d {
		lines[i] = dline{sign: byte(l.sign), text: strings.ReplaceAll(l.text, "\t", "    ")}
	}
	masks := wordMasks(lines)
	maxRows := 14
	if m.cardRows > 0 {
		maxRows = m.cardRows
	}
	rows := []string{head}
	for i, l := range lines {
		if i == maxRows {
			rows = append(rows, sDim.Render(fmt.Sprintf("  … %d more lines · /diff shows everything", len(d)-maxRows)))
			break
		}
		bg, em, sc := cAddBg, lerp(cAddBg, cAdd, 0.3), cAdd
		switch l.sign {
		case '-':
			bg, em, sc = cDelBg, lerp(cDelBg, cDel, 0.3), cDel
		case ' ':
			bg, em, sc = cPanel, cPanel, cDim
		}
		rows = append(rows, lipgloss.NewStyle().Foreground(sc).Background(bg).Render(" "+string(l.sign)+" ")+renderCode(lex, l.text, bg, em, masks[i], max(inner-3, 4)))
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cCard).Render(strings.Join(rows, "\n"))
}

// ---------- spinner ----------

func (m *Model) renderSpinner(w int) string {
	el := time.Since(m.turnStart)
	verb := verbs[int(el.Seconds()/4)%len(verbs)]
	parts := []string{fmtDur(el.Truncate(time.Second)), "↓ " + fmtInt(m.turnTokens()) + " tokens"}
	if m.thinking {
		parts = append(parts, "thinking")
	} else if m.thinkDur > 0 {
		parts = append(parts, "thought for "+fmtDur(m.thinkDur))
	}
	stops := aurora
	if fx := fxStops(m.effort); fx != nil {
		stops = fx
	}
	s := scramble(4, m.frame-m.turnFrame, stops) + " " +
		shimmer(verb+"…", m.frame, stops) + " " +
		sDim.Render("("+strings.Join(parts, " · ")+")")
	if len(m.queue) > 0 {
		s += sDim.Render(" · " + plural(len(m.queue), "queued prompt"))
	}
	if sp := m.sparkline(); sp != "" && lipgloss.Width(s)+lipgloss.Width(sp)+2 <= w {
		s += "  " + sp
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}

// ---------- sidebar ----------

func (m *Model) renderSidebar(maxH int) string {
	cw := sideW - 4 // content width inside border + padding
	title := func(s string) string { return sDim.Render(spaced(s)) }
	card := func(lines ...string) string {
		return cardStyle.Width(sideW).Render(strings.Join(lines, "\n"))
	}

	// MODELS
	ml := []string{title("MODELS")}
	ms := m.modelCache[m.prov.ID()]
	cur := m.currentModel()
	if len(ms) == 0 {
		ml = append(ml, sDim.Render("no model downloaded"))
	}
	for _, mi := range ms {
		dot, name, state := sDim.Render("○"), sMuted.Render(mi.ID), ""
		if mi.ID == cur {
			dot, name, state = sTeal.Render("●"), sBold.Render(mi.ID), "active"
		}
		left := dot + " " + name
		ml = append(ml, left+strings.Repeat(" ", max(cw-lipgloss.Width(left)-len(state), 1))+sDim.Render(state))
	}

	// CONTEXT
	ctxMax := m.contextMax()
	ratio := float64(m.ctxTokens) / float64(ctxMax)
	filled := min(int(ratio*float64(cw)+0.5), cw)
	stops, cyclic, off := meterStops(ratio), false, 0
	if fx := fxStops(m.effort); fx != nil && ratio < 0.65 {
		stops, cyclic, off = fx, true, -m.frame
	}
	bar := gradient(strings.Repeat("━", max(filled, 0)), stops, off, cyclic, false) + fg(cCard).Render(strings.Repeat("━", cw-max(filled, 0)))
	cl := []string{title("CONTEXT"), bar, sDim.Render(fmt.Sprintf("%s / %s · %d%%", fmtK(m.ctxTokens), fmtK(ctxMax), int(ratio*100)))}

	// CHANGED
	ch := []string{title("CHANGED")}
	if len(m.changeOrder) == 0 {
		ch = append(ch, sDim.Render("nothing yet"))
	}
	ch = append(ch, m.changedTree(cw)...)

	// EFFORT + SESSION
	ef := []string{title("EFFORT"), effortLabel(m.effort, m.frame) + sDim.Render("  /effort")}
	se := []string{title("SESSION"), sText.Render(plural(m.turns, "turn")) + sDim.Render(" · "+fmtDur(time.Since(m.sessStart).Truncate(time.Second)))}

	cards := []string{card(ml...), card(cl...)}
	if len(m.todos) > 0 {
		cards = append(cards, card(m.renderTodoCard(cw)...))
	}
	if jobs := native.Jobs(); len(jobs) > 0 {
		jl := []string{title("JOBS")}
		for _, j := range jobs {
			what := j.URL
			if what == "" {
				what = strings.Join(strings.Fields(j.Command), " ")
			}
			up := fmtDur(time.Since(j.Started).Truncate(time.Second))
			jl = append(jl, pulse("●", m.frame)+" "+sText.Render(truncRight(what, cw-lipgloss.Width(up)-3))+" "+sDim.Render(up))
		}
		cards = append(cards, card(jl...))
	}
	cards = append(cards, card(ch...))
	// Optional cards go last and are dropped first when the column is short
	// (effort is also shown in the input box).
	for _, c := range []string{card(se...), card(ef...)} {
		if maxH <= 0 || lipgloss.Height(lipgloss.JoinVertical(lipgloss.Left, append(cards, c)...)) <= maxH {
			cards = append(cards, c)
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, cards...)
}

// ---------- input + status ----------

func (m *Model) renderInput(w int) string {
	label := sDim.Render(m.prov.ID()+" · "+m.modelName()+" · ") + effortLabel(m.effort, m.frame)
	prompt := sTeal.Bold(true).Render("❯ ")
	if m.vim.on && m.vim.normal {
		prompt = fg(cAmber).Bold(true).Render("N ")
	} else if m.vim.on {
		prompt = sTeal.Bold(true).Render("I ")
	}
	innerW := w - 4
	m.ta.SetWidth(max(innerW-lipgloss.Width(prompt)-lipgloss.Width(label)-2, 10))
	row := lipgloss.JoinHorizontal(lipgloss.Top, prompt, m.ta.View(), "  ", label)

	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(w)
	if fx := fxStops(m.effort); fx != nil {
		box = box.BorderForegroundBlend(append(fx, fx[0])...).BorderForegroundBlendOffset(-m.frame)
	} else if m.busy && !m.plain {
		box = box.BorderForegroundBlend(aurora[0], aurora[1], aurora[2], aurora[1], aurora[0]).BorderForegroundBlendOffset(-m.frame / 2)
	} else if m.busy {
		box = box.BorderForeground(cTeal)
	} else {
		box = box.BorderForeground(cInputLine)
	}
	return box.Render(row)
}

func (m *Model) renderStatus(w int) string {
	left := " " + modeLabel(m.mode) + sDim.Render(" (shift+tab to cycle)")
	switch {
	case m.planPending:
		left += sDim.Render(" · ") + sTeal.Bold(true).Render("plan ready") + sDim.Render(" · /go runs it")
	case m.planFirst:
		left += sDim.Render(" · ") + sTeal.Render("plan first")
	}
	if m.webOff {
		left += sDim.Render(" · ") + sDim.Render("web off")
	}
	if !m.leader.IsZero() && time.Since(m.leader) < 3*time.Second {
		left = " " + shimmer("ctrl+x …", m.frame, aurora) + sDim.Render(" waiting for a leader key (/keys lists them)")
	}
	if m.busy {
		left += sDim.Render(" · esc to interrupt")
	}
	right := sDim.Render(fmtInt(m.ctxTokens) + " tokens ")
	if n := m.lspErrTotal(); n > 0 {
		right = sErr.Render(fmt.Sprintf("✗ %s ", plural(n, "LSP error"))) + right
	}
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m *Model) renderSuggestions(w int) string {
	s := m.suggestions()
	if len(s) == 0 {
		return ""
	}
	sel := m.sugIdx % len(s)
	start := 0
	if sel >= 8 {
		start = sel - 7
	}
	var rows []string
	for i := start; i < len(s) && i < start+8; i++ {
		c := s[i]
		name := "/" + c.name
		st, cur := sMuted, "  "
		if c.soon != "" {
			st = sDim
		}
		if i == sel {
			st, cur = sTeal.Bold(true), sTeal.Render("❯ ")
		}
		desc := c.desc
		if c.soon != "" {
			desc += " · " + c.soon
		}
		rows = append(rows, cur+st.Render(fmt.Sprintf("%-14s", name))+sDim.Render(truncRight(desc, w-18)))
	}
	if len(s) > 8 {
		rows = append(rows, sDim.Render(fmt.Sprintf("  %d/%d · ↑↓ · tab complete · enter run", sel+1, len(s))))
	}
	return strings.Join(rows, "\n")
}

// renderApproval is the permission prompt shown above the input.
func (m *Model) renderApproval(w int) string {
	a := m.approval
	tgt := m.toolTarget(a.Tool)
	name := a.Tool.Name
	if strings.HasPrefix(name, "mcp__") {
		name = toolLabel(name)
	}
	head := fg(cAmber).Bold(true).Render("Allow ") + sBold.Render(name) + " " + sText.Render(truncRight(tgt, w-20)) + fg(cAmber).Bold(true).Render("?")
	rows := []string{head}
	if a.Reason != "" {
		rows = append(rows, sDim.Render(a.Reason))
	}
	if d := diffFor(a.Tool); len(d) > 0 {
		m.cardRows = max(3, min(14, m.h-14)) // keep the y/a/n keys and the input on screen
		rows = append(rows, m.renderDiffCard(a.Tool, d, w-4))
		m.cardRows = 0
	}
	always := ""
	if s := approvalScopeOf(a); s != "" {
		always = sTeal.Bold(true).Render("a") + sDim.Render(" always allow "+scopeLabel(s)+" this session  ")
	}
	rows = append(rows, sTeal.Bold(true).Render("y")+sDim.Render(" allow once  ")+always+fg(cErr).Bold(true).Render("n")+sDim.Render(" deny"))
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cAmber).Padding(0, 1).Width(w).Render(strings.Join(rows, "\n"))
}

// ---------- helpers ----------

func toolLabel(name string) string {
	if rest, ok := strings.CutPrefix(name, "mcp__browser__browser_"); ok {
		return "Browser " + strings.ReplaceAll(rest, "_", " ")
	}
	if rest, ok := strings.CutPrefix(name, "mcp__"); ok {
		return strings.Replace(rest, "__", ".", 1)
	}
	switch name {
	case "Bash", "PowerShell":
		return "Run"
	case "MultiEdit":
		return "Edit"
	case "Grep":
		return "Search"
	case "Glob":
		return "Find"
	case "WebFetch":
		return "Fetch"
	case "WebSearch":
		return "Web"
	case "Task", "Agent":
		return "Agent"
	case "TodoWrite":
		return "Todo"
	case "Verify":
		return "Verify"
	case "LSP":
		return "LSP"
	case "Jobs":
		return "Jobs"
	case "Terminal":
		return "Terminal"
	}
	return name
}

func (m *Model) toolTarget(t agent.Tool) string {
	for _, k := range []string{"file_path", "path", "command", "pattern", "url", "query", "description", "prompt"} {
		if v, ok := t.Input[k].(string); ok && v != "" {
			v = strings.Join(strings.Fields(v), " ")
			if k == "file_path" || k == "path" {
				v = m.rel(v)
			}
			return v
		}
	}
	return ""
}

func (m *Model) rel(p string) string {
	if r, err := filepath.Rel(m.dir, p); err == nil && !strings.HasPrefix(r, "..") {
		return filepath.ToSlash(r)
	}
	return p
}

var modelRe = regexp.MustCompile(`^claude-([a-z]+)-(\d+)-(\d+)`)

// prettyModel turns "claude-opus-5-5" into "Opus 5.5".
func prettyModel(id string) string {
	if mm := modelRe.FindStringSubmatch(id); mm != nil {
		return strings.ToUpper(mm[1][:1]) + mm[1][1:] + " " + mm[2] + "." + mm[3]
	}
	return id
}

func windowName(w string) string {
	switch w {
	case "five_hour":
		return "5h"
	case "seven_day":
		return "weekly"
	}
	return strings.ReplaceAll(w, "_", " ")
}

func spaced(s string) string { return strings.Join(strings.Split(s, ""), " ") }

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// fmtInt shows a token count compactly: 820, 1.0k, 1.2k, 37.8k, 120k, 1.4M.
func fmtInt(n int) string {
	switch {
	case n < 1000:
		return fmt.Sprint(n)
	case n < 99_950: // below this, one decimal still fits: 99.9k
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	case n < 999_500:
		return fmt.Sprintf("%dk", (n+500)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1e6)
}

func fmtK(n int) string {
	if n >= 1_000_000 {
		s := fmt.Sprintf("%.1fM", float64(n)/1e6)
		return strings.Replace(s, ".0M", "M", 1)
	}
	if n >= 1000 && n%1000 == 0 {
		return fmt.Sprintf("%dk", n/1000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprint(n)
}

func fmtDur(d time.Duration) string {
	switch {
	case d > 0 && d < time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	// From a minute on: "1m", "1m 2s", "1h", "1h 5m" — never "62s".
	s := int(d.Seconds())
	h, mins, sec := s/3600, s%3600/60, s%60
	switch {
	case h > 0 && mins == 0:
		return fmt.Sprintf("%dh", h)
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, mins)
	case sec == 0:
		return fmt.Sprintf("%dm", mins)
	}
	return fmt.Sprintf("%dm %ds", mins, sec)
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(strings.TrimRight(s, "\n"), "\n") + 1
}

func nonEmptyLines(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}

func truncRight(s string, w int) string {
	if w <= 1 || lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)) > w-1 {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

func truncLeft(s string, w int) string {
	if w <= 1 || lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r)) > w-1 {
		r = r[1:]
	}
	return "…" + string(r)
}

var lspCountRe = regexp.MustCompile(`LSP diagnostics after this edit \((\d+) errors`)

// lspErrors reads the error count NØX appended to an Edit/Write result.
func lspErrors(result string) int {
	if mm := lspCountRe.FindStringSubmatch(result); mm != nil {
		n := 0
		fmt.Sscan(mm[1], &n)
		return n
	}
	return 0
}

func (m *Model) lspErrTotal() int {
	n := 0
	for _, v := range m.lspErr {
		n += v
	}
	return n
}

// renderCompact shows a compaction: what the context shrank to and the
// summary the model now works from (collapsed; click to expand).
func (m *Model) renderCompact(e entry, w int) string {
	head := sTeal.Render("◇") + " " + sText.Bold(true).Render("context compacted") + sDim.Render(" · "+strings.Replace(e.result, "→", " → ", 1)+" tokens · the model continues from this summary")
	lines := strings.Split(strings.TrimSpace(e.text), "\n")
	n := 3
	if e.expanded {
		n = len(lines)
	}
	var body []string
	for i, l := range lines {
		if i == n {
			body = append(body, sDim.Render(fmt.Sprintf("    … %d more lines (click to expand)", len(lines)-n)))
			break
		}
		prefix := "    "
		if i == 0 {
			prefix = "  ⎿ "
		}
		body = append(body, sDim.Render(prefix+truncRight(l, w-4)))
	}
	return head + "\n" + strings.Join(body, "\n")
}

// renderBtwBody renders a side answer as Markdown, the same way an answer is
// rendered, but without the streaming cache: a btw answer arrives whole.
func (m *Model) renderBtwBody(e entry, w int) string {
	out, ok := renderMarkdown(strings.TrimRight(e.text, "\n"), w)
	if !ok {
		out = lipgloss.NewStyle().Width(w).Render(e.text)
	}
	return out
}
