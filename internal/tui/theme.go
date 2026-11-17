package tui

import (
	"image/color"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"charm.land/lipgloss/v2"

	"github.com/Johnny008-creator/N0X-Local/internal/agent"
)

// theme is one palette. a1→a2→a3 is the signature gradient (logo, prompt
// bar, context meter); a1 is also the main accent. All look-book directions
// the user liked are here; "halo" (design #12) is the default.
type theme struct {
	name, desc                          string
	fg, white, dim, muted, card, panel  string
	a1, a2, a3                          string
	add, addBg, del, delBg, err, border string
}

var themes = []theme{
	{"halo", "aurora: teal → lime → amber (default)", "#E3ECEF", "#FFFFFF", "#5D6B75", "#8A99A3", "#24323D", "#111A21",
		"#2DD4BF", "#A3E635", "#FBBF24", "#9FF5D2", "#0D2A24", "#FF9F9F", "#2A1616", "#FF6B6B", "#2B3A46"},
	{"ember", "warm: amber → coral → rose", "#F1E7E2", "#FFFFFF", "#7A6660", "#A08A83", "#3A2B27", "#1C1513",
		"#FB923C", "#FB7185", "#F472B6", "#C7F0A8", "#1D2A17", "#FFAAB0", "#331718", "#FF6B6B", "#3A2B27"},
	{"ledger", "gold on charcoal", "#E6E1D6", "#FFFFFF", "#6F6A61", "#9A948A", "#2E2B27", "#1B1A18",
		"#F2C46D", "#D7A35F", "#8FCF9A", "#A8E0B0", "#16261A", "#E8A5A5", "#2A1717", "#E8726B", "#34312C"},
	{"frost", "cool blue, Monolith style", "#E8E8EA", "#FFFFFF", "#6B6F7E", "#9599A8", "#30343D", "#1B1D23",
		"#9AB4FF", "#7DD3FC", "#C4B5FD", "#B9F5C8", "#12301D", "#FFB3B8", "#3A1518", "#FF6B6B", "#363946"},
	{"mono", "greys only, for limited terminals", "#E0E0E0", "#FFFFFF", "#707070", "#9A9A9A", "#3A3A3A", "#1A1A1A",
		"#FFFFFF", "#C8C8C8", "#909090", "#D0D0D0", "#262626", "#A0A0A0", "#1E1E1E", "#FF6B6B", "#444444"},
}

var (
	cFg, cWhite, cDim, cMuted, cCard, cPanel     color.Color
	cTeal, cLime, cAmber                         color.Color
	cAdd, cAddBg, cDel, cDelBg, cErr, cInputLine color.Color
	aurora                                       []color.Color
	current                                      = themes[0]
)

var (
	rainbow = hexes("#FF5F6D", "#FFB56B", "#F9F871", "#6BFFB0", "#5FD7FF", "#A78BFA", "#FF7AD9")
	ultra   = hexes("#6D28D9", "#A855F7", "#E9B8FF", "#A855F7")
)

func init() { applyTheme("halo") }

// loadUserThemes appends themes from ~/.nox/themes.
func loadUserThemes(dir string) {
	files, _ := filepath.Glob(filepath.Join(dir, "*.toml"))
	for _, f := range files {
		var u map[string]string
		if _, err := toml.DecodeFile(f, &u); err != nil {
			continue
		}
		base := themes[0]
		t := base
		t.name = strings.TrimSuffix(filepath.Base(f), ".toml")
		t.desc = "custom · " + u["description"]
		pick := func(k string, def string) string {
			if v := u[k]; v != "" {
				return v
			}
			return def
		}
		t.fg, t.white, t.dim, t.muted = pick("fg", base.fg), pick("white", base.white), pick("dim", base.dim), pick("muted", base.muted)
		t.card, t.panel = pick("card", base.card), pick("panel", base.panel)
		t.a1, t.a2, t.a3 = pick("accent1", base.a1), pick("accent2", base.a2), pick("accent3", base.a3)
		t.add, t.addBg, t.del, t.delBg = pick("add", base.add), pick("add_bg", base.addBg), pick("del", base.del), pick("del_bg", base.delBg)
		t.err, t.border = pick("error", base.err), pick("border", base.border)
		replaced := false
		for i := range themes {
			if themes[i].name == t.name {
				themes[i], replaced = t, true
			}
		}
		if !replaced {
			themes = append(themes, t)
		}
	}
}

// applyTheme switches the palette and rebuilds every derived style.
func applyTheme(name string) bool {
	var t *theme
	for i := range themes {
		if themes[i].name == name {
			t = &themes[i]
		}
	}
	if t == nil {
		return false
	}
	current = *t
	c := lipgloss.Color
	cFg, cWhite, cDim, cMuted, cCard, cPanel = c(t.fg), c(t.white), c(t.dim), c(t.muted), c(t.card), c(t.panel)
	cTeal, cLime, cAmber = c(t.a1), c(t.a2), c(t.a3)
	cAdd, cAddBg, cDel, cDelBg, cErr, cInputLine = c(t.add), c(t.addBg), c(t.del), c(t.delBg), c(t.err), c(t.border)
	aurora = []color.Color{cTeal, cLime, cAmber}
	sDim, sMuted, sText = fg(cDim), fg(cMuted), fg(cFg)
	sBold, sTeal, sLime, sErr = fg(cWhite).Bold(true), fg(cTeal), fg(cLime), fg(cErr)
	cardStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cCard).Padding(0, 1)
	resetMarkdown()
	return true
}

var modeColor = map[agent.Mode]color.Color{
	agent.ModeManual: lipgloss.Color("#A7ACBD"),
	agent.ModePlan:   lipgloss.Color("#4FD1C5"),
	agent.ModeAuto:   lipgloss.Color("#F0B35A"),
	agent.ModeYolo:   lipgloss.Color("#FF6B6B"),
}

func hexes(hs ...string) []color.Color {
	cs := make([]color.Color, len(hs))
	for i, h := range hs {
		cs[i] = lipgloss.Color(h)
	}
	return cs
}

func fg(c color.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

var sDim, sMuted, sText, sBold, sTeal, sLime, sErr lipgloss.Style

// gradient colours every rune of s along stops. cyclic wraps the palette so
// that increasing offset makes the colours flow (used for max / ultracode).
func gradient(s string, stops []color.Color, offset int, cyclic, bold bool) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	var pal []color.Color
	if cyclic {
		loop := append(append([]color.Color{}, stops...), stops[0])
		pal = lipgloss.Blend1D(max(len(runes), 24), loop...)
	} else {
		pal = lipgloss.Blend1D(max(len(runes), 2), stops...)
	}
	n := len(pal)
	var b strings.Builder
	for i, r := range runes {
		if r == ' ' {
			b.WriteRune(r)
			continue
		}
		st := lipgloss.NewStyle().Foreground(pal[((i+offset)%n+n)%n])
		if bold {
			st = st.Bold(true)
		}
		b.WriteString(st.Render(string(r)))
	}
	return b.String()
}

// fxStops returns an animated palette for efforts that deserve one. Local
// models use three flat levels, so nothing animates there — the hook is kept
// for themes that want it.
func fxStops(e agent.Effort) []color.Color {
	return nil
}

// effortLabel renders the effort name: low dim, medium muted, high bold teal.
func effortLabel(e agent.Effort, _ int) string {
	switch e {
	case agent.EffortLow:
		return sDim.Render("low")
	case agent.EffortMedium:
		return sMuted.Render("medium")
	}
	return sTeal.Bold(true).Render("high")
}

func modeLabel(m agent.Mode) string {
	return fg(modeColor[m]).Render(m.Glyph() + " " + m.Label())
}
