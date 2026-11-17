package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Johnny008-creator/nox-local/internal/agent"
	"github.com/Johnny008-creator/nox-local/internal/checkpoint"
	"github.com/Johnny008-creator/nox-local/internal/config"
	"github.com/Johnny008-creator/nox-local/internal/hooks"
)

type (
	hookMsg struct {
		event string
		res   []hooks.Result
	}
	promptHookMsg struct {
		text string
		res  []hooks.Result
	}
	commitMsg struct {
		hash string
		err  error
	}
)

// onHooks reports hook results; it returns true when one failed.
func (m *Model) onHooks(h hookMsg) bool {
	failed := false
	for _, r := range h.res {
		switch {
		case r.Err != nil:
			failed = true
			msg := fmt.Sprintf("%s hook failed: %s (%v)", h.event, r.Hook.Command, r.Err)
			if r.Output != "" {
				msg += "\n" + r.Output
			}
			m.errorf(msg)
		case h.event == "stop" && r.Output != "":
			m.info(fmt.Sprintf("stop hook · %s\n%s", r.Hook.Command, r.Output))
		}
	}
	return failed
}

// postToolHooks queues post_tool hooks for a finished tool call; they run
// in the background and only speak up when they fail.
func (m *Model) postToolHooks(t agent.Tool) {
	hs := hooks.Select(m.hooks, "post_tool", t.Name)
	if len(hs) == 0 {
		return
	}
	file, _ := t.Input["file_path"].(string)
	v := hooks.Vars{Event: "post_tool", Tool: t.Name, File: file, Prompt: m.lastPrompt, Dir: m.dir}
	m.pending = append(m.pending, func() tea.Msg { return hookMsg{"post_tool", hooks.Run(hs, v)} })
}

func (m *Model) onCommit(c commitMsg) {
	switch {
	case c.err != nil:
		m.errorf("auto-commit failed: " + c.err.Error())
	case c.hash != "":
		m.info("committed " + c.hash)
	}
}

func configPath() string { return config.Path() }

func (m *Model) stopHooks() tea.Cmd {
	hs := hooks.Select(m.hooks, "stop", "")
	if len(hs) == 0 || m.lastFailed {
		return nil
	}
	v := hooks.Vars{Event: "stop", Prompt: m.lastPrompt, Dir: m.dir}
	return func() tea.Msg { return hookMsg{"stop", hooks.Run(hs, v)} }
}

// maybeCommit commits after a turn that changed files when auto_commit is on.
func (m *Model) maybeCommit() tea.Cmd {
	if !m.autoCommit || m.repo == nil || m.lastFailed || m.reviewing || len(m.changeOrder) <= m.turnChanged {
		return nil
	}
	repo := m.repo
	msg := "NØX: " + truncRight(strings.Join(strings.Fields(m.lastPrompt), " "), 68) +
		"\n\nCo-Authored-By: NØX via " + m.turnProv.Name()
	return func() tea.Msg {
		h, err := repo.Commit(msg)
		return commitMsg{h, err}
	}
}

func cmdWorktree(m *Model, arg string) tea.Cmd {
	switch {
	case m.repo == nil:
		m.errorf("/worktree needs a git repository")
		return nil
	case arg == "" || strings.ContainsAny(arg, " /\\:"):
		m.errorf("usage: /worktree <name>  (letters, digits, - and _)")
		return nil
	case m.busy:
		m.errorf("wait for the current turn to finish first")
		return nil
	}
	path, err := m.repo.Worktree(arg)
	if err != nil {
		m.errorf("worktree: " + err.Error())
		return nil
	}
	m.save()
	m.dir = path
	m.repo = checkpoint.Open(path)
	m.points, m.sessFor, m.saved = nil, map[string]string{}, nil
	m.changes, m.changeOrder = map[string]change{}, nil
	m.fidx = fileIndex{}
	m.info("now working in " + path + " on branch nox/" + arg + "\nthe model starts a fresh session there; the conversation is handed over")
	return nil
}

func cmdHooks(m *Model, _ string) tea.Cmd {
	var rows []string
	for _, h := range m.hooks {
		match := ""
		if h.Match != "" {
			match = " [" + h.Match + "]"
		}
		rows = append(rows, fmt.Sprintf("  %-9s%s  %s", h.Event, match, h.Command))
	}
	if len(rows) == 0 {
		rows = append(rows, "  none · add [[hooks]] to "+filepath.ToSlash(configPath()))
	}
	ac := "off"
	if m.autoCommit {
		ac = "on"
	}
	m.info("hooks\n" + strings.Join(rows, "\n") + "\nauto_commit " + ac)
	return nil
}
