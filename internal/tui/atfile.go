package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// project file index for @-mentions, refreshed at most every 20s.
type fileIndex struct {
	files []string
	built time.Time
}

var skipIndex = map[string]bool{".git": true, "node_modules": true, "vendor": true, "dist": true, "build": true, ".venv": true, "__pycache__": true, "target": true, ".next": true}

func (m *Model) projectFiles() []string {
	if time.Since(m.fidx.built) < 20*time.Second && m.fidx.files != nil {
		return m.fidx.files
	}
	var out []string
	_ = filepath.WalkDir(m.dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != m.dir && (skipIndex[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if rel, err := filepath.Rel(m.dir, p); err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		if len(out) > 20000 {
			return filepath.SkipAll
		}
		return nil
	})
	m.fidx = fileIndex{files: out, built: time.Now()}
	return out
}

// atWord returns the @token being typed at the end of the input.
func (m *Model) atWord() (string, bool) {
	v := m.ta.Value()
	i := strings.LastIndexAny(v, " \n\t")
	w := v[i+1:]
	if !strings.HasPrefix(w, "@") || strings.HasPrefix(v, "/") {
		return "", false
	}
	return w[1:], true
}

type atItem struct{ insert, label, desc string }

// atSuggestions offers providers (only as the first word) and project files.
func (m *Model) atSuggestions() []atItem {
	q, ok := m.atWord()
	if !ok {
		return nil
	}
	ql := strings.ToLower(q)
	var out []atItem
	if !strings.ContainsAny(strings.TrimSpace(m.ta.Value()), " \n") {
		for _, p := range m.provs {
			if strings.HasPrefix(p.ID(), ql) && p != m.prov {
				out = append(out, atItem{"@" + p.ID() + " ", "@" + p.ID(), "send this prompt to " + p.Name()})
			}
		}
	}
	type scored struct {
		f     string
		score int
	}
	var hits []scored
	for _, f := range m.projectFiles() {
		fl := strings.ToLower(f)
		base := strings.ToLower(filepath.Base(f))
		switch {
		case ql == "":
			hits = append(hits, scored{f, 3})
		case strings.HasPrefix(base, ql):
			hits = append(hits, scored{f, 0})
		case strings.Contains(base, ql):
			hits = append(hits, scored{f, 1})
		case strings.Contains(fl, ql):
			hits = append(hits, scored{f, 2})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score < hits[j].score
		}
		return len(hits[i].f) < len(hits[j].f)
	})
	for i, h := range hits {
		if i == 30 {
			break
		}
		out = append(out, atItem{"@" + h.f + " ", "@" + h.f, ""})
	}
	return out
}

// completeAt replaces the @token being typed with the chosen item.
func (m *Model) completeAt(it atItem) {
	v := m.ta.Value()
	i := strings.LastIndexAny(v, " \n\t")
	m.ta.SetValue(v[:i+1] + it.insert)
	m.ta.CursorEnd()
}

const maxAttach = 100_000

// expandFiles appends the content of every @path that names a project file.
func (m *Model) expandFiles(text string) string {
	var b strings.Builder
	seen := map[string]bool{}
	for _, w := range strings.Fields(text) {
		if !strings.HasPrefix(w, "@") || len(w) < 2 {
			continue
		}
		rel := strings.TrimRight(w[1:], ".,;:!?)")
		if seen[rel] || m.provider(rel) != nil {
			continue
		}
		p := filepath.Join(m.dir, filepath.FromSlash(rel))
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		seen[rel] = true
		note := ""
		if len(data) > maxAttach {
			data, note = data[:maxAttach], " truncated=\"true\""
		}
		fmt.Fprintf(&b, "\n\n<file path=%q%s>\n%s\n</file>", rel, note, data)
	}
	if b.Len() == 0 {
		return text
	}
	return text + "\n\nAttached files:" + b.String()
}

func (m *Model) renderAtSuggestions(w int) string {
	s := m.atSuggestions()
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
		st, cur := sMuted, "  "
		if i == sel {
			st, cur = sTeal.Bold(true), sTeal.Render("❯ ")
		}
		row := cur + st.Render(truncRight(s[i].label, w-30))
		if s[i].desc != "" {
			row += "  " + sDim.Render(s[i].desc)
		}
		rows = append(rows, row)
	}
	rows = append(rows, sDim.Render(fmt.Sprintf("  %d/%d · ↑↓ · tab/enter insert", sel+1, len(s))))
	return strings.Join(rows, "\n")
}
