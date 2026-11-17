package tui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/BurntSushi/toml"

	"github.com/Johnny008-creator/nox-local/internal/agent"
	"github.com/Johnny008-creator/nox-local/internal/paths"
)

// Command palette (ctrl+k): one fuzzy list over commands, providers,
// themes, skills, recent sessions and project files.
func (m *Model) openPalette() tea.Cmd {
	var items []pickItem
	add := func(kind, value, label, desc string) {
		items = append(items, pickItem{value: kind + ":" + value, label: label, desc: kind + " · " + desc})
	}
	for _, c := range commands {
		if c.run != nil {
			add("cmd", c.name, "/"+c.name, c.desc)
		}
	}
	for _, t := range themes {
		add("theme", t.name, "◐ theme "+t.name, t.desc)
	}
	for _, s := range agent.Skills(m.dir) {
		add("skill", s.Path, "✦ "+s.Name, s.Desc)
	}
	if m.store != nil {
		if list, err := m.store.List(m.dir); err == nil {
			for i, s := range list {
				if i == 8 {
					break
				}
				add("session", s.ID, "↺ "+truncRight(s.Title(), 50), s.Updated.Format("Jan 2 15:04"))
			}
		}
	}
	files := m.projectFiles()
	for i, f := range files {
		if i == 3000 {
			break
		}
		add("file", f, "@"+f, "attach to the prompt")
	}
	m.picker = newPicker("Command palette · type to search", items, true, func(m *Model, it pickItem) tea.Cmd {
		kind, val, _ := strings.Cut(it.value, ":")
		switch kind {
		case "cmd":
			return m.runCommand("/" + val)
		case "provider":
			return m.switchProvider(val)
		case "theme":
			return cmdTheme(m, val)
		case "skill":
			m.ta.SetValue("Use the skill at " + val + " to " + m.ta.Value())
			m.ta.CursorEnd()
		case "session":
			if s, err := m.store.Load(val); err == nil {
				m.save()
				m.resetConversation()
				m.restore(s)
			}
		case "file":
			v := m.ta.Value()
			if v != "" && !strings.HasSuffix(v, " ") {
				v += " "
			}
			m.ta.SetValue(v + "@" + val + " ")
			m.ta.CursorEnd()
		}
		return nil
	})
	return nil
}

// ---------- key map + leader ----------

// keymap binds keys to commands, from ~/.nox/keys.toml:
//
//	"ctrl+g" = "/goal"
//	"leader d" = "/diff"
//
// The leader is ctrl+x: press it, then the second key.
var defaultKeys = map[string]string{
	"leader d": "/diff", "leader r": "/review", "leader f": "/find", "leader s": "/search",
	"leader t": "/todos", "leader c": "/compact", "leader o": "/doctor",
	"leader h": "/handoff", "leader k": "/keys", "leader p": "palette", "leader v": "/vim",
	"leader u": "/usage", "leader w": "/rewind",
}

// keymapErr is the last keys.toml problem, shown once.
var keymapErr string

func loadKeymap() map[string]string {
	km := map[string]string{}
	for k, v := range defaultKeys {
		km[k] = v
	}
	var user map[string]string
	if b, err := os.ReadFile(paths.File("keys.toml")); err == nil {
		if err := toml.Unmarshal(b, &user); err != nil {
			keymapErr = "keys.toml: " + err.Error() + " · default keys are used"
		}
		for k, v := range user {
			km[strings.ToLower(strings.TrimSpace(k))] = v
		}
	}
	return km
}

// runBinding executes what a key is bound to.
func (m *Model) runBinding(action string) tea.Cmd {
	switch {
	case action == "palette":
		return m.openPalette()
	case strings.HasPrefix(action, "/"):
		return m.runCommand(action)
	}
	m.ta.SetValue(action)
	m.ta.CursorEnd()
	return nil
}

// leaderKey handles the key after ctrl+x.
func (m *Model) leaderKey(k string) tea.Cmd {
	m.leader = time.Time{}
	if a, ok := m.keymap["leader "+k]; ok {
		return m.runBinding(a)
	}
	m.info("leader + " + k + " is not bound · /keys lists bindings")
	return nil
}

func keymapHelp(km map[string]string) string {
	var ks []string
	for k := range km {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	var b strings.Builder
	for _, k := range ks {
		fmt.Fprintf(&b, "  %-12s %s\n", strings.Replace(k, "leader ", "ctrl+x ", 1), km[k])
	}
	return strings.TrimRight(b.String(), "\n")
}
