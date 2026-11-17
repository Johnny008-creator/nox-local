package tui

import (
	"fmt"
	"hash/fnv"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ---------- provider identity ----------

// provColors gives every provider a stable signature colour.
var provPalette = hexes("#2DD4BF", "#A78BFA", "#F472B6", "#FBBF24", "#60A5FA", "#34D399", "#FB923C", "#F87171")

func provColor(id string) color.Color {
	h := fnv.New32a()
	h.Write([]byte(id))
	return provPalette[h.Sum32()%uint32(len(provPalette))]
}

// batonFrames is how long the header pill takes to morph on a switch.
const batonFrames = 9

// providerPill renders "● id" in the provider colour, morphing from the
// previous provider's colour right after a switch.
func (m *Model) providerPill() string {
	c := provColor(m.prov.ID())
	if m.batonFrom != "" {
		t := float64(m.frame-m.batonStart) / batonFrames
		if t >= 1 {
			m.batonFrom = ""
		} else {
			e := 1 - math.Pow(1-t, 3) // ease-out
			c = lerp(provColor(m.batonFrom), c, e)
		}
	}
	return lipgloss.NewStyle().Foreground(cPanel).Background(c).Bold(true).Render(" " + m.prov.ID() + " ")
}

// ---------- throughput sparkline ----------

var sparkRunes = []rune("▁▂▃▄▅▆▇█")

// sampleRate records output characters per tick while a turn runs.
func (m *Model) sampleRate() {
	if !m.busy {
		return
	}
	d := m.outChars - m.lastOut
	m.lastOut = m.outChars
	m.rates = append(m.rates, d)
	if len(m.rates) > 24 {
		m.rates = m.rates[len(m.rates)-24:]
	}
}

// tokPerSec is the generation speed read off the same sample the sparkline
// uses: output characters per 110 ms tick, about four characters per token.
func (m *Model) tokPerSec() string {
	if len(m.rates) < 2 {
		return "—"
	}
	last := m.rates[len(m.rates)-1] + m.rates[len(m.rates)-2]
	return fmt.Sprintf("%.0f tok/s", float64(last)/4/0.22)
}

func (m *Model) sparkline() string {
	if len(m.rates) < 3 {
		return ""
	}
	mx := 1
	for _, r := range m.rates {
		mx = max(mx, r)
	}
	pal := lipgloss.Blend1D(len(sparkRunes), aurora...)
	var b strings.Builder
	for _, r := range m.rates {
		i := r * (len(sparkRunes) - 1) / mx
		b.WriteString(fg(pal[i]).Render(string(sparkRunes[i])))
	}
	// chars per tick → tokens per second (≈4 chars per token, 110 ms ticks)
	last := m.rates[len(m.rates)-1] + m.rates[len(m.rates)-2]
	tps := float64(last) / 4 / 0.22
	return b.String() + sDim.Render(fmt.Sprintf(" %.0f tok/s", tps))
}

// ---------- context meter colours ----------

// meterStops warms the context bar from aurora to amber to red as it fills.
func meterStops(ratio float64) []color.Color {
	switch {
	case ratio >= 0.85:
		return []color.Color{cAmber, cErr}
	case ratio >= 0.65:
		return []color.Color{cLime, cAmber}
	}
	return aurora
}

// ---------- changed files as a tree ----------

func (m *Model) changedTree(cw int) []string {
	type node struct {
		dir   string
		files []string
	}
	byDir := map[string][]string{}
	var dirs []string
	for _, p := range m.changeOrder {
		d := filepath.ToSlash(filepath.Dir(p))
		if _, ok := byDir[d]; !ok {
			dirs = append(dirs, d)
		}
		byDir[d] = append(byDir[d], p)
	}
	sort.Strings(dirs)
	var out []string
	shown := 0
	for _, d := range dirs {
		if shown >= 8 {
			out = append(out, sDim.Render(fmt.Sprintf("+%d more", len(m.changeOrder)-shown)))
			break
		}
		if d != "." {
			out = append(out, sDim.Render(truncLeft(d+"/", cw)))
		}
		fs := byDir[d]
		for i, p := range fs {
			if shown >= 8 {
				break
			}
			branch := "├ "
			if i == len(fs)-1 {
				branch = "└ "
			}
			if d == "." {
				branch = ""
			}
			c := m.changes[p]
			stat := fg(cAdd).Render(fmt.Sprintf("+%d", c.add)) + " " + fg(cDel).Render(fmt.Sprintf("−%d", c.del))
			dot := ""
			if n := m.lspErrFor(p); n > 0 {
				dot = sErr.Render(" ●")
			}
			name := filepath.Base(p)
			room := cw - lipgloss.Width(branch) - lipgloss.Width(stat) - lipgloss.Width(dot) - 1
			out = append(out, sDim.Render(branch)+m.fileLink(p, sText.Render(truncRight(name, room)))+dot+strings.Repeat(" ", max(1, room-lipgloss.Width(truncRight(name, room))+1))+stat)
			shown++
		}
	}
	return out
}

func (m *Model) lspErrFor(rel string) int {
	for p, n := range m.lspErr {
		if m.rel(p) == rel {
			return n
		}
	}
	return 0
}

// ---------- links + clipboard ----------

// fileLink makes text a clickable file:// link (OSC 8) in terminals that
// support it; others just show the text.
func (m *Model) fileLink(rel, text string) string {
	if m.plain {
		return text
	}
	abs := rel
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(m.dir, rel)
	}
	return lipgloss.NewStyle().Hyperlink("file://" + filepath.ToSlash(abs)).Render(text)
}

var codeBlockRe = regexp.MustCompile("(?s)```[^\n]*\n(.*?)```")

// cmdCopy copies the last answer (or its n-th code block) to the system
// clipboard through the terminal (OSC 52), which also works over SSH.
func cmdCopy(m *Model, arg string) tea.Cmd {
	var last string
	for i := len(m.entries) - 1; i >= 0; i-- {
		if m.entries[i].kind == kText {
			last = m.entries[i].text
			break
		}
	}
	if last == "" {
		m.info("nothing to copy yet")
		return nil
	}
	what := "the last answer"
	if arg != "" {
		blocks := codeBlockRe.FindAllStringSubmatch(last, -1)
		n := 0
		fmt.Sscan(arg, &n)
		if n < 1 || n > len(blocks) {
			m.errorf(fmt.Sprintf("the last answer has %s", plural(len(blocks), "code block")))
			return nil
		}
		last, what = blocks[n-1][1], fmt.Sprintf("code block %d", n)
	}
	m.toast("copied " + what)
	return tea.SetClipboard(last)
}

// ---------- splash ----------

var tips = []string{
	"ctrl+k opens the command palette",
	"/goal $ go test ./... keeps going until the tests pass",
	"/review asks the model for an evidence-based review",
	"/compact replaces the history with a summary when the context fills",
	"alt+enter queues a message for the turn that is running",
	"/diff opens every change of this session side by side",
	"ctrl+x then d, r, f … are leader shortcuts · /keys",
	"/handoff starts a fresh thread that keeps the essentials",
	"@file.go puts a whole file into the prompt",
	"/doctor checks llama-server, models and MCP",
}

// renderSplash is the empty state: the wordmark scrambling into place,
// recent sessions and a tip.
func (m *Model) renderSplash(w int) string {
	word := "N Ø X"
	reveal := min(m.frame, 12)
	var mark string
	if reveal < 12 && !m.plain {
		rs := []rune(word)
		var b strings.Builder
		for i, r := range rs {
			if r == ' ' || i*3 < reveal {
				b.WriteRune(r)
			} else {
				b.WriteRune(rune(scrambleRunes[(m.frame*7+i*13)%len(scrambleRunes)]))
			}
		}
		mark = gradient(b.String(), aurora, 0, false, true)
	} else {
		mark = shimmer(word, m.frame/2, aurora)
	}
	rows := []string{"", mark, sDim.Render("terminal coding agent · " + m.prov.Name() + " · " + m.modelName()), ""}
	if m.store != nil && time.Since(m.splashAt) > 10*time.Second {
		m.splashList, _ = m.store.List(m.dir) // reading every session per frame is slow
		m.splashAt = time.Now()
	}
	if m.store != nil {
		if list := m.splashList; len(list) > 0 {
			rows = append(rows, sDim.Render(spaced("RECENT")))
			for i, s := range list {
				if i == 4 {
					break
				}
				rows = append(rows, sMuted.Render("↺ "+truncRight(s.Title(), w-20))+sDim.Render("  "+s.Updated.Format("Jan 2")))
			}
			rows = append(rows, sDim.Render("/resume or nox -c continues one"), "")
		}
	}
	tip := tips[int(time.Now().Unix()/3600)%len(tips)]
	rows = append(rows, sDim.Render("tip · ")+sText.Render(tip))
	if _, err := os.Stat(filepath.Join(m.dir, "NOX.md")); err != nil {
		rows = append(rows, sDim.Render("no NOX.md here yet · /init writes one"))
	}
	return strings.Join(rows, "\n")
}
