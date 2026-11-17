package tui

import (
	"image/color"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Syntax highlighting for diffs, from Chroma: token colours over the
// add/delete backgrounds, plus word-level emphasis for the parts of a line
// that actually changed.

var chromaStyle = styles.Get("github-dark")

func lexerFor(path string) chroma.Lexer {
	l := lexers.Match(filepath.Base(path))
	if l == nil {
		return nil
	}
	return chroma.Coalesce(l)
}

type span struct {
	text string
	fg   color.Color // nil = default text colour
}

// highlight splits one line into coloured spans.
func highlight(lex chroma.Lexer, line string) []span {
	if lex == nil {
		return []span{{text: line}}
	}
	it, err := lex.Tokenise(nil, line)
	if err != nil {
		return []span{{text: line}}
	}
	var out []span
	for _, tok := range it.Tokens() {
		v := strings.TrimRight(tok.Value, "\n")
		if v == "" {
			continue
		}
		e := chromaStyle.Get(tok.Type)
		var fgc color.Color
		if e.Colour.IsSet() {
			fgc = lipgloss.Color(e.Colour.String())
		}
		out = append(out, span{v, fgc})
	}
	return out
}

// tokens splits a line into words and the gaps between them, for word diffs.
func tokens(s string) []string {
	var out []string
	start := 0
	isWord := func(r rune) bool {
		return r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r > 127
	}
	rs := []rune(s)
	for i := 1; i <= len(rs); i++ {
		if i == len(rs) || isWord(rs[i]) != isWord(rs[i-1]) || !isWord(rs[i]) {
			out = append(out, string(rs[start:i]))
			start = i
		}
	}
	return out
}

// changedMask marks, per rune of a, whether it is outside the longest
// common token subsequence with b.
func changedMask(a, b string) []bool {
	ta, tb := tokens(a), tokens(b)
	n, m := len(ta), len(tb)
	if n*m > 40000 { // very long lines: mark everything
		mask := make([]bool, len([]rune(a)))
		for i := range mask {
			mask[i] = true
		}
		return mask
	}
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if ta[i] == tb[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}
	var mask []bool
	i, j := 0, 0
	for i < n {
		switch {
		case j < m && ta[i] == tb[j]:
			mask = append(mask, make([]bool, len([]rune(ta[i])))...)
			i, j = i+1, j+1
		case j < m && dp[i][j+1] >= dp[i+1][j]:
			j++
		default:
			for range []rune(ta[i]) {
				mask = append(mask, true)
			}
			i++
		}
	}
	return mask
}

// renderCode draws one diff line: syntax colours on bg, changed runes
// (mask) on the stronger emphasis background. width pads the line.
func renderCode(lex chroma.Lexer, line string, bg, emph color.Color, mask []bool, width int) string {
	var b strings.Builder
	pos, used := 0, 0
	for _, sp := range highlight(lex, line) {
		for _, r := range sp.text {
			if used >= width {
				break
			}
			st := lipgloss.NewStyle().Background(bg)
			if sp.fg != nil {
				st = st.Foreground(sp.fg)
			} else {
				st = st.Foreground(cFg)
			}
			if pos < len(mask) && mask[pos] && emph != nil {
				st = st.Background(emph).Bold(true)
			}
			w := lipgloss.Width(string(r))
			if used+w > width {
				used = width
				break
			}
			b.WriteString(st.Render(string(r)))
			used += w
			pos++
		}
	}
	if used < width {
		b.WriteString(lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width-used)))
	}
	return b.String()
}
