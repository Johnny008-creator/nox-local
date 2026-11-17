package agent

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Johnny008-creator/N0X-Local/internal/paths"
)

// ProjectMemory returns the path of the project's NOX.md, walking up from dir
// to the filesystem root, or "" if there is none.
func ProjectMemory(dir string) string {
	for d := dir; ; {
		p := filepath.Join(d, "NOX.md")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

// GlobalMemory is ~/.nox-local/NOX.md (user-wide instructions), if present.
func GlobalMemory() string {
	p := paths.File("NOX.md")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

const maxMemory = 40_000 // bytes per file; keeps prompts sane

func readCapped(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	if len(b) > maxMemory {
		b = b[:maxMemory]
	}
	return strings.TrimSpace(string(b))
}

// leanMemory caps NOX.md for small local models, whose whole window may be
// 8–32K tokens.
const leanMemory = 6_000

// SystemPromptLean is the prompt for local models: identity, NOX.md (capped),
// remembered facts and a compact skill index.
func SystemPromptLean(dir string) string {
	var b strings.Builder
	b.WriteString(Identity)
	for _, p := range []string{GlobalMemory(), ProjectMemory(dir)} {
		if p == "" {
			continue
		}
		if s := readCapped(p); s != "" {
			if len(s) > leanMemory {
				s = s[:leanMemory] + "\n[…truncated]"
			}
			b.WriteString("\n\n# Project instructions (" + filepath.Base(filepath.Dir(p)) + "/NOX.md)\n\n" + s)
		}
	}
	b.WriteString(factsSection(dir))
	b.WriteString(skillIndex(dir))
	return b.String()
}
