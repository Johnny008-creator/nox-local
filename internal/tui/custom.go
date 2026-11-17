package tui

import (
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Johnny008-creator/N0X-Local/internal/paths"
)

// loadCustomCommands registers prompt templates from ~/.nox/commands/*.md and
// <project>/.nox/commands/*.md (project wins). The file name is the command;
// the first line starting with "description:" becomes its help text, and
// $ARGUMENTS is replaced by whatever follows the command.
func loadCustomCommands(dir string) {
	seen := map[string]bool{}
	for _, c := range commands {
		seen[c.name] = true
	}
	var custom []command
	for _, d := range []string{filepath.Join(dir, ".nox", "commands"), paths.Sub("commands")} {
		files, _ := filepath.Glob(filepath.Join(d, "*.md"))
		for _, f := range files {
			name := strings.ToLower(strings.TrimSuffix(filepath.Base(f), ".md"))
			if seen[name] {
				continue
			}
			b, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			body, desc := string(b), "custom command"
			if first, rest, ok := strings.Cut(body, "\n"); ok && strings.HasPrefix(strings.ToLower(first), "description:") {
				desc, body = strings.TrimSpace(first[len("description:"):]), rest
			}
			tmpl := strings.TrimSpace(body)
			seen[name] = true
			custom = append(custom, command{name: name, args: "[args]", desc: desc + " · " + shortPath(f), run: func(m *Model, arg string) tea.Cmd {
				prompt := strings.ReplaceAll(tmpl, "$ARGUMENTS", arg)
				if arg != "" && !strings.Contains(tmpl, "$ARGUMENTS") {
					prompt += "\n\n" + arg
				}
				if m.busy {
					return m.onSubmitWhileBusy(prompt)
				}
				return m.startTurn(prompt)
			}})
		}
	}
	// custom commands go right after the built-in ones that work today
	i := 0
	for i < len(commands) && commands[i].soon == "" {
		i++
	}
	commands = append(commands[:i], append(custom, commands[i:]...)...)
}

func shortPath(p string) string {
	if h, err := os.UserHomeDir(); err == nil && strings.HasPrefix(p, h) {
		return "~" + strings.TrimPrefix(p, h)
	}
	return p
}
