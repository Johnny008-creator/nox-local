package tui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
)

// /diff opens a full diff viewer for everything changed this session:
// syntax highlighting, word-level emphasis, unified or side-by-side (auto
// on wide terminals, s toggles), n/p jumps between files.

type dline struct {
	sign         byte // ' ', '+', '-'
	text         string
	oldNo, newNo int
}

type dhunk struct {
	header string
	lines  []dline
}

type dfile struct {
	path     string
	hunks    []dhunk
	add, del int
}

func parseUnified(s string) []dfile {
	var files []dfile
	var f *dfile
	var h *dhunk
	oldNo, newNo := 0, 0
	for _, l := range strings.Split(s, "\n") {
		switch {
		case strings.HasPrefix(l, "diff --git "):
			files = append(files, dfile{})
			f, h = &files[len(files)-1], nil
			if i := strings.LastIndex(l, " b/"); i >= 0 {
				f.path = l[i+3:]
			}
		case f == nil:
		case strings.HasPrefix(l, "+++ "), strings.HasPrefix(l, "--- "):
			if p, ok := strings.CutPrefix(l, "+++ b/"); ok {
				f.path = p
			}
		case strings.HasPrefix(l, "@@"):
			f.hunks = append(f.hunks, dhunk{header: l})
			h = &f.hunks[len(f.hunks)-1]
			// @@ -a,b +c,d @@
			var a, c int
			parts := strings.Fields(l)
			if len(parts) >= 3 {
				a, _ = strconv.Atoi(strings.Split(strings.TrimPrefix(parts[1], "-"), ",")[0])
				c, _ = strconv.Atoi(strings.Split(strings.TrimPrefix(parts[2], "+"), ",")[0])
			}
			oldNo, newNo = a, c
		case h != nil && len(l) > 0 && (l[0] == '+' || l[0] == '-' || l[0] == ' '):
			dl := dline{sign: l[0], text: strings.ReplaceAll(l[1:], "\t", "    ")}
			switch l[0] {
			case '+':
				dl.newNo = newNo
				newNo++
				f.add++
			case '-':
				dl.oldNo = oldNo
				oldNo++
				f.del++
			default:
				dl.oldNo, dl.newNo = oldNo, newNo
				oldNo++
				newNo++
			}
			h.lines = append(h.lines, dl)
		}
	}
	return files
}

type diffView struct {
	files  []dfile
	vp     viewport.Model
	split  bool
	auto   bool  // split chosen automatically from width
	starts []int // line offset of each file in the rendered content
	w      int
}

// wordMasks pairs each run of deletions with the following additions and
// marks the changed runes on both sides.
func wordMasks(lines []dline) map[int][]bool {
	masks := map[int][]bool{}
	for i := 0; i < len(lines); {
		if lines[i].sign != '-' {
			i++
			continue
		}
		ds := i
		for i < len(lines) && lines[i].sign == '-' {
			i++
		}
		as := i
		for i < len(lines) && lines[i].sign == '+' {
			i++
		}
		nd, na := as-ds, i-as
		for k := 0; k < min(nd, na); k++ {
			masks[ds+k] = changedMask(lines[ds+k].text, lines[as+k].text)
			masks[as+k] = changedMask(lines[as+k].text, lines[ds+k].text)
		}
	}
	return masks
}

func numCol(n int) string {
	if n == 0 {
		return "    "
	}
	return fmt.Sprintf("%4d", n)
}

func (dv *diffView) renderFile(f dfile, w int) []string {
	lex := lexerFor(f.path)
	head := sBold.Render(f.path) + "  " + fg(cAdd).Render(fmt.Sprintf("+%d", f.add)) + " " + fg(cDel).Render(fmt.Sprintf("−%d", f.del))
	rows := []string{gradient(strings.Repeat("━", min(w, 60)), aurora, 0, false, false), head}
	for _, h := range f.hunks {
		rows = append(rows, sDim.Render(truncRight(h.header, w)))
		masks := wordMasks(h.lines)
		if dv.split {
			rows = append(rows, dv.splitRows(h.lines, masks, lex, w)...)
			continue
		}
		for i, l := range h.lines {
			rows = append(rows, unifiedRow(l, masks[i], lex, w))
		}
	}
	return append(rows, "")
}

func unifiedRow(l dline, mask []bool, lex chroma.Lexer, w int) string {
	nums := sDim.Render(numCol(l.oldNo) + " " + numCol(l.newNo) + " ")
	bg, em := cPanel, cPanel
	sign := sDim.Render("  ")
	switch l.sign {
	case '+':
		bg, em = cAddBg, lerp(cAddBg, cAdd, 0.3)
		sign = fg(cAdd).Background(bg).Render("+ ")
	case '-':
		bg, em = cDelBg, lerp(cDelBg, cDel, 0.3)
		sign = fg(cDel).Background(bg).Render("- ")
	default:
		mask = nil
	}
	return nums + sign + renderCode(lex, l.text, bg, em, mask, max(w-12, 10))
}

// splitRows lays deletions on the left and additions on the right.
// sideNo is a line's number in the left (old) or right (new) column; an
// unchanged line has a different number on each side once lines were added
// or removed above it.
func sideNo(l *dline, left bool) int {
	switch {
	case l.sign == '-':
		return l.oldNo
	case l.sign == '+':
		return l.newNo
	case left:
		return l.oldNo
	}
	return l.newNo
}

func (dv *diffView) splitRows(lines []dline, masks map[int][]bool, lex chroma.Lexer, w int) []string {
	half := (w - 3) / 2
	cell := func(i int, l *dline, left bool) string {
		if l == nil {
			return lipgloss.NewStyle().Width(half).Render("")
		}
		n := sideNo(l, left)
		bg, em := cPanel, cPanel
		var mask []bool
		switch l.sign {
		case '-':
			bg, em, mask = cDelBg, lerp(cDelBg, cDel, 0.3), masks[i]
		case '+':
			bg, em, mask = cAddBg, lerp(cAddBg, cAdd, 0.3), masks[i]
		}
		return sDim.Render(numCol(n)+" ") + renderCode(lex, l.text, bg, em, mask, half-5)
	}
	var rows []string
	for i := 0; i < len(lines); {
		if lines[i].sign == ' ' {
			rows = append(rows, cell(i, &lines[i], true)+sDim.Render(" │ ")+cell(i, &lines[i], false))
			i++
			continue
		}
		ds := i
		for i < len(lines) && lines[i].sign == '-' {
			i++
		}
		as := i
		for i < len(lines) && lines[i].sign == '+' {
			i++
		}
		nd, na := as-ds, i-as
		for k := 0; k < max(nd, na); k++ {
			var left, right string
			if k < nd {
				left = cell(ds+k, &lines[ds+k], true)
			} else {
				left = cell(0, nil, true)
			}
			if k < na {
				right = cell(as+k, &lines[as+k], false)
			} else {
				right = cell(0, nil, false)
			}
			rows = append(rows, left+sDim.Render(" │ ")+right)
		}
	}
	return rows
}

func (dv *diffView) layout(w, h int) {
	if dv.auto {
		dv.split = w >= 140
	}
	dv.w = w
	var all []string
	dv.starts = dv.starts[:0]
	for _, f := range dv.files {
		dv.starts = append(dv.starts, len(all))
		all = append(all, dv.renderFile(f, w)...)
	}
	dv.vp.SetWidth(w)
	dv.vp.SetHeight(max(h, 3))
	dv.vp.SetContent(strings.Join(all, "\n"))
}

func (dv *diffView) current() int {
	cur := 0
	for i, s := range dv.starts {
		if dv.vp.YOffset() >= s {
			cur = i
		}
	}
	return cur
}

func cmdDiff(m *Model, arg string) tea.Cmd {
	if m.repo == nil || len(m.points) == 0 {
		m.errorf("/diff needs a git repository and at least one turn this session")
		return nil
	}
	txt, err := m.repo.DiffText(m.points[0], arg)
	if err != nil {
		m.errorf("diff: " + err.Error())
		return nil
	}
	files := parseUnified(txt)
	if len(files) == 0 {
		m.info("no changes since the session started")
		return nil
	}
	m.diff = &diffView{files: files, vp: viewport.New(), auto: true}
	return nil
}

func (m *Model) diffKey(k tea.KeyPressMsg) tea.Cmd {
	dv := m.diff
	switch k.String() {
	case "esc", "q":
		m.diff = nil
	case "s":
		dv.auto, dv.split = false, !dv.split
	case "n":
		if c := dv.current(); c+1 < len(dv.starts) {
			dv.vp.SetYOffset(dv.starts[c+1])
		}
	case "p":
		if c := dv.current(); c > 0 && dv.vp.YOffset() == dv.starts[c] {
			dv.vp.SetYOffset(dv.starts[c-1])
		} else {
			dv.vp.SetYOffset(dv.starts[c])
		}
	case "g", "home":
		dv.vp.GotoTop()
	case "G", "end":
		dv.vp.GotoBottom()
	case "j", "down":
		dv.vp.ScrollDown(1)
	case "k", "up":
		dv.vp.ScrollUp(1)
	case "pgdown", "space", "f":
		dv.vp.PageDown()
	case "pgup", "b":
		dv.vp.PageUp()
	}
	return nil
}

func (m *Model) renderDiffView() string {
	dv := m.diff
	w := m.w - 2
	off := dv.vp.YOffset()
	dv.layout(w, m.h-2)
	dv.vp.SetYOffset(off)
	c := dv.current()
	f := dv.files[c]
	mode := "unified"
	if dv.split {
		mode = "split"
	}
	bar := sTeal.Bold(true).Render(" DIFF ") + sDim.Render(fmt.Sprintf(" file %d/%d · ", c+1, len(dv.files))) + sText.Render(f.path) +
		sDim.Render(fmt.Sprintf(" · %s · n/p files · s %s · j/k scroll · esc close", mode, map[bool]string{true: "unified", false: "split"}[dv.split]))
	return lipgloss.NewStyle().PaddingLeft(1).Render(lipgloss.JoinVertical(lipgloss.Left, truncRight(bar, w), dv.vp.View()))
}
