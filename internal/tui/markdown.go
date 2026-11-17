package tui

import (
	"strings"
	"sync"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
)

var (
	mdMu        sync.Mutex
	mdRenderers = map[int]*glamour.TermRenderer{}
)

func sp(s string) *string { return &s }
func bp(b bool) *bool     { return &b }
func up(u uint) *uint     { return &u }

// haloMarkdown is Glamour's dark style re-tinted to the Halo palette and
// without document margins (the chat column already has its own).
func haloMarkdown() glamour.TermRendererOption {
	st := styles.DarkStyleConfig
	st.Document.Margin = up(0)
	st.Document.BlockPrefix, st.Document.BlockSuffix = "", ""
	st.Document.Color = sp(current.fg)
	st.Paragraph.Color = sp(current.fg)
	st.Heading.Color, st.Heading.Bold = sp(current.a1), bp(true)
	st.H1.Prefix, st.H1.Suffix = "", ""
	st.H1.Color, st.H1.BackgroundColor, st.H1.Bold = sp(current.a2), nil, bp(true)
	st.H2.Color = sp(current.a1)
	st.H3.Color = sp(current.a1)
	st.Strong.Color, st.Strong.Bold = sp(current.white), bp(true)
	st.Emph.Color = sp(current.muted)
	st.Code.Color, st.Code.BackgroundColor = sp(current.a3), sp(current.panel)
	st.Link.Color = sp(current.a1)
	st.LinkText.Color = sp(current.a1)
	st.BlockQuote.Color = sp(current.muted)
	st.Item.BlockPrefix = "• "
	st.HorizontalRule.Color = sp(current.card)
	st.CodeBlock.Margin = up(0)
	return glamour.WithStyles(st)
}

func renderer(w int) *glamour.TermRenderer {
	mdMu.Lock()
	defer mdMu.Unlock()
	if r, ok := mdRenderers[w]; ok {
		return r
	}
	r, err := glamour.NewTermRenderer(haloMarkdown(), glamour.WithWordWrap(w), glamour.WithChromaFormatter("terminal16m"))
	if err != nil {
		return nil
	}
	mdRenderers[w] = r
	return r
}

// renderMarkdown renders assistant text; falls back to plain text on error.
func renderMarkdown(s string, w int) (string, bool) {
	r := renderer(w)
	if r == nil {
		return "", false
	}
	out, err := r.Render(s)
	if err != nil {
		return "", false
	}
	return strings.Trim(out, "\n"), true
}

// renderText renders entry i as Markdown. While an answer streams, the part
// up to the last finished block (blank line outside a code fence) is
// rendered once and cached; only the growing tail is re-rendered each frame,
// so long answers stay cheap instead of re-rendering everything per token.
func (m *Model) renderText(i, w int) string {
	e := &m.entries[i]
	if e.cache != "" && e.cacheW == w && e.cacheLen == len(e.text) {
		return e.cache
	}
	src := strings.TrimRight(e.text, "\n")
	cut := stableCut(src)
	if e.cacheW != w || cut < e.prefixLen || e.prefixOut == "" && cut > 0 {
		e.prefixLen, e.prefixOut = 0, ""
	}
	if cut > e.prefixLen {
		if out, ok := renderMarkdown(src[:cut], w); ok {
			e.prefixLen, e.prefixOut = cut, out
		}
	}
	var out string
	if e.prefixLen > 0 {
		tail := strings.TrimLeft(src[e.prefixLen:], "\n")
		out = e.prefixOut
		if tail != "" {
			t, ok := renderMarkdown(tail, w)
			if !ok {
				t = sText.Width(w).Render(tail)
			}
			out += "\n\n" + t
		}
	} else {
		var ok bool
		if out, ok = renderMarkdown(src, w); !ok {
			out = sText.Width(w).Render(src)
		}
	}
	e.cache, e.cacheW, e.cacheLen = out, w, len(e.text)
	return out
}

// stableCut is the end of the last complete Markdown block in s: the last
// blank line that is not inside a ``` fence (0 if none).
func stableCut(s string) int {
	cut, inFence, pos := 0, false, 0
	lines := strings.SplitAfter(s, "\n")
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			inFence = !inFence
		}
		pos += len(l)
		if !inFence && t == "" && i > 0 && i < len(lines)-1 {
			cut = pos
		}
	}
	return cut
}

// resetMarkdown drops cached renderers after a theme change.
func resetMarkdown() {
	mdMu.Lock()
	mdRenderers = map[int]*glamour.TermRenderer{}
	mdMu.Unlock()
}
