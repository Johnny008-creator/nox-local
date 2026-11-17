package tui

import (
	"fmt"
	"github.com/Johnny008-creator/N0X-Local/internal/checkpoint"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Johnny008-creator/N0X-Local/internal/agent"
	"github.com/Johnny008-creator/N0X-Local/internal/config"
	"github.com/Johnny008-creator/N0X-Local/internal/local"
	"github.com/Johnny008-creator/N0X-Local/internal/paths"
	"github.com/Johnny008-creator/N0X-Local/internal/trust"
)

type command struct {
	name, args, desc string
	soon             string // milestone when not implemented yet
	run              func(m *Model, arg string) tea.Cmd
}

var commands []command

func init() {
	commands = []command{
		{name: "help", desc: "All commands (pick one to run it)", run: func(m *Model, _ string) tea.Cmd { return m.openCommandPicker() }},
		{name: "model", args: "[name]", desc: "Pick a downloaded model", run: cmdModel},
		{name: "effort", args: "[level]", desc: "How much the model may think: low, medium, high", run: cmdEffort},
		{name: "mode", args: "[mode]", desc: "Permission mode: manual, plan, auto, yolo", run: cmdMode},
		{name: "plan", args: "[toggle]", desc: "Plan first: the agent plans read-only, /go runs the plan", run: cmdPlanFirst},
		{name: "research", args: "<question>", desc: "Deep research: rounds of searching and reading, then a sourced report", run: cmdResearch},
		{name: "btw", args: "<question>", desc: "Side question in parallel: does not touch the turn in flight", run: cmdBtw},
		{name: "artifacts", args: "[n]", desc: "Charts, graphics, diagrams and documents this conversation made", run: cmdArtifacts},
		{name: "go", desc: "Run the plan the agent wrote in plan-first mode", run: cmdGo},
		{name: "usage", desc: "Context window and token totals for this session", run: cmdUsage},
		{name: "context", desc: "What fills the context window", run: cmdContext},
		{name: "status", desc: "Provider, model, mode, session", run: cmdStatus},
		{name: "compact", args: "[focus]", desc: "Summarise the conversation to free context", run: cmdCompact},
		{name: "handoff", args: "[goal]", desc: "Fresh thread seeded with a summary of this one", run: cmdHandoff},
		{name: "clear", desc: "Start a fresh conversation", run: cmdClear},
		{name: "resume", desc: "Resume a saved session", run: cmdResume},
		{name: "find", args: "<text>", desc: "Search this conversation and jump to a hit (ctrl+f)", run: cmdFind},
		{name: "search", args: "<text>", desc: "Search all saved sessions (prompts, answers, files)", run: cmdSearch},
		{name: "export", args: "[file]", desc: "Export the conversation to Markdown", run: cmdExport},
		{name: "init", desc: "Have NØX write NOX.md for this project", run: cmdInit},
		{name: "memory", desc: "Show which NOX.md files are loaded", run: cmdMemory},
		{name: "review", args: "[commit <sha>|branch <base>]", desc: "Independent review by another model (evidence required)", run: cmdReview},
		{name: "sessions", desc: "Every running NØX window and what it is doing", run: cmdSessions},
		{name: "send", args: "<pid> <text>", desc: "Pass a note to another NØX window", run: cmdSend},
		{name: "loop", args: "<interval> <prompt> | off", desc: "Re-send a prompt on an interval while this window is open", run: cmdLoop},
		{name: "goal", args: "<condition> | $ <cmd> | off", desc: "Keep working until a condition holds (checked by command or judge)", run: cmdGoal},
		{name: "fork", desc: "Branch this conversation into a new session (original kept)", run: cmdFork},
		{name: "trust", args: "[off]", desc: "Allow this project's own settings (.mcp.json, .nox/config.toml)", run: cmdTrust},
		{name: "skills", desc: "Skills the model can use (pick one to use it)", run: cmdSkills},
		{name: "mcp", desc: "MCP servers this project can switch on", run: cmdMCP},
		{name: "worktree", args: "<name>", desc: "Continue in a new git worktree on branch nox/<name>", run: cmdWorktree},
		{name: "hooks", desc: "Configured hooks and auto-commit", run: cmdHooks},
		{name: "twin", desc: "Split view: this model | another", run: cmdTwin},
		{name: "rewind", desc: "Undo file changes back to before an earlier turn", run: cmdRewind},
		{name: "diff", args: "[file]", desc: "Full diff viewer for this session's changes (split / unified)", run: cmdDiff},
		{name: "redo", desc: "Undo the last /rewind (valid until the next prompt)", run: cmdRedo},
		{name: "revert", args: "<file>", desc: "Put one file back to how it was before this turn", run: cmdRevert},
		{name: "permissions", desc: "Tools you allowed for this session", run: cmdPermissions},
		{name: "theme", args: "[name]", desc: "Colour theme: halo, ember, ledger, frost, mono", run: cmdTheme},
		{name: "keys", desc: "Keyboard shortcuts", run: cmdKeys},
		{name: "doctor", desc: "Check llama-server, models and MCP", run: cmdDoctor},
		{name: "open", args: "<file|url>", desc: "Open a file, image or URL in the system viewer", run: cmdOpen},
		{name: "copy", args: "[n]", desc: "Copy the last answer (or its n-th code block) to the clipboard", run: cmdCopy},
		{name: "plain", desc: "Toggle plain mode: no animation (also NOX_PLAIN=1)", run: func(m *Model, _ string) tea.Cmd {
			m.plain = !m.plain
			m.info(map[bool]string{true: "plain mode on: no animation", false: "plain mode off"}[m.plain])
			return nil
		}},
		{name: "vim", desc: "Toggle vim keys in the prompt (esc normal, i insert)", run: cmdVim},
		{name: "todos", desc: "The agent's task list", run: cmdTodos},
		{name: "quit", desc: "Exit NØX", run: func(m *Model, _ string) tea.Cmd { return m.quit() }},
		{name: "exit", desc: "Exit NØX", run: func(m *Model, _ string) tea.Cmd { return m.quit() }},

		{name: "config", args: "[project]", desc: "Edit ~/.nox/config.toml (or .nox/config.toml) in $EDITOR", run: cmdConfig},
	}
}

func (m *Model) suggestions() []command {
	v := m.ta.Value()
	if !strings.HasPrefix(v, "/") || strings.ContainsAny(v, " \n") {
		return nil
	}
	p := strings.ToLower(v[1:])
	var out []command
	for _, c := range commands {
		if strings.HasPrefix(c.name, p) {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].soon == "" && out[j].soon != "" })
	return out
}

func (m *Model) runCommand(text string) tea.Cmd {
	name, arg, _ := strings.Cut(strings.TrimPrefix(text, "/"), " ")
	arg = strings.TrimSpace(arg)
	for _, c := range commands {
		if c.name == strings.ToLower(name) {
			if c.run == nil {
				m.info(fmt.Sprintf("/%s lands in milestone %s (docs/ROADMAP.md)", c.name, c.soon))
				return nil
			}
			return c.run(m, arg)
		}
	}
	m.errorf("unknown command /" + name + " · /help lists everything")
	return nil
}

func cmdEffort(m *Model, arg string) tea.Cmd {
	if arg == "" {
		return m.openEffortPicker()
	}
	e, ok := agent.ParseEffort(arg)
	if !ok {
		m.errorf("effort must be one of low, medium, high")
		return nil
	}
	m.effort = e
	m.info("effort → " + e.String())
	return nil
}

func cmdMode(m *Model, arg string) tea.Cmd {
	if arg == "" {
		return m.openModePicker()
	}
	md, ok := agent.ParseMode(arg)
	if !ok {
		m.errorf("mode must be one of manual, plan, auto, yolo")
		return nil
	}
	m.mode = md
	m.info("mode → " + md.Label())
	return nil
}

// planFirstGuide asks for a plan to approve instead of changes. The desktop
// has the same text: one product, one instruction.
const planFirstGuide = `# Plan first
The user wants to approve a plan before anything changes. Investigate
read-only as much as you need, then answer with the plan: a short goal, a
numbered list of concrete steps (which files, what changes), risks or open
questions, and how you will check the result. Change nothing yet; the user
will approve the plan or ask for changes to it.`

// cmdPlanFirst toggles plan-first mode. While it is on, every prompt runs
// read-only and ends as a plan; /go then runs that plan for real.
func cmdPlanFirst(m *Model, arg string) tea.Cmd {
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "off", "no", "false":
		m.planFirst = false
	case "on", "yes", "true":
		m.planFirst = true
	default:
		m.planFirst = !m.planFirst
	}
	if !m.planFirst {
		m.planPending, m.planText = false, ""
		m.info("plan first off · changes run directly")
		return nil
	}
	m.info("plan first on · the agent plans read-only, /go runs the plan")
	return nil
}

// cmdGo approves the pending plan: the same plan goes back in the current
// mode, so the agent executes what it proposed.
func cmdGo(m *Model, note string) tea.Cmd {
	if !m.planPending || strings.TrimSpace(m.planText) == "" {
		m.errorf("no plan waiting · /plan turns plan-first on")
		return nil
	}
	plan := m.planText
	m.planPending, m.planText = false, ""
	prompt := "# Approved plan — carry it out\n\nThe plan below was approved. Execute it the way it is written; say so if the code makes a step impossible.\n\n" + plan
	if s := strings.TrimSpace(note); s != "" {
		prompt += "\n\n# The user's note\n\n" + s
	}
	// m.mode is whatever it was before planning: /plan never changed it.
	// This one turn runs for real, even though plan-first stays on.
	m.skipPlanOnce = true
	return m.startTurnOn(m.prov, prompt, m.mode, false)
}

func cmdModel(m *Model, arg string) tea.Cmd {
	if arg == "" {
		return m.openModelPicker()
	}
	m.setModel(arg)
	if list, ok := m.modelCache[m.prov.ID()]; ok && len(list) > 0 {
		known := false
		for _, mi := range list {
			if mi.ID == arg {
				known = true
				break
			}
		}
		if !known {
			m.info("model → " + arg + " · not in " + m.prov.Name() + "'s list (ctrl+o shows it); used as typed")
			return nil
		}
	}
	m.info("model → " + arg)
	return nil
}

func cmdUsage(m *Model, _ string) tea.Cmd {
	ctx := m.ctxTokens
	mx := max(m.contextMax(), 1)
	m.info(fmt.Sprintf("usage · %s (%s)\n  context   %s / %s tokens (%d%%)\n  output    %s tokens this session\n  turns     %d\n  speed     %s",
		m.prov.Name(), m.modelName(), fmtInt(ctx), fmtInt(mx), ctx*100/mx,
		fmtInt(m.outTokens), m.turns, m.tokPerSec()))
	return nil
}

func cmdContext(m *Model, _ string) tea.Cmd {
	mem := []string{}
	for _, p := range []string{agent.GlobalMemory(), agent.ProjectMemory(m.dir)} {
		if p != "" {
			mem = append(mem, p)
		}
	}
	memStr := "none"
	if len(mem) > 0 {
		memStr = strings.Join(mem, ", ")
	}
	head := fmt.Sprintf("context · %s of %s tokens (%d%%) · %s", fmtInt(m.ctxTokens), fmtInt(m.contextMax()), m.ctxTokens*100/max(m.contextMax(), 1), m.modelName())
	var user, text, tools int
	for _, e := range m.entries {
		switch e.kind {
		case kUser:
			user += len(e.text)
		case kText:
			text += len(e.text)
		case kTool:
			tools += len(e.result)
		}
	}
	m.info(fmt.Sprintf("%s\n  your messages   ~%s tok\n  answers         ~%s tok\n  tool output     ~%s tok\n  memory          %s\n  (estimate from this window; the model keeps its own history)",
		head, fmtInt(user/4), fmtInt(text/4), fmtInt(tools/4), memStr))
	return nil
}

func cmdStatus(m *Model, _ string) tea.Cmd {
	sid := m.sessFor[m.prov.ID()]
	if sid == "" {
		sid = "new"
	}
	m.info(fmt.Sprintf("status\n  backend   %s\n  model     %s · %s ctx\n  mode      %s\n  effort    %s\n  session   %s\n  dir       %s",
		m.prov.Name(), m.modelName(), fmtK(m.contextMax()), m.mode, m.effort, sid, m.dir))
	return nil
}

func cmdCompact(m *Model, arg string) tea.Cmd {
	if len(m.entries) == 0 {
		m.info("nothing to compact yet")
		return nil
	}
	if m.busy {
		m.errorf("wait for the current turn to finish (or esc) before /compact")
		return nil
	}
	m.shownAs = strings.TrimSpace("/compact " + arg)
	m.compacting = true
	return m.startTurn(compactPrompt + arg)
}

func cmdResearch(m *Model, arg string) tea.Cmd {
	if strings.TrimSpace(arg) == "" {
		m.info("usage: /research <question> · works best on high effort (bigger context)")
		return nil
	}
	if m.busy {
		m.errorf("wait for the current turn to finish (or esc) before /research")
		return nil
	}
	m.research = true
	return m.startTurn(arg)
}

const compactPrompt = "Summarise our conversation so far so a fresh session can continue it without the history: goals, decisions and why, files read or changed, commands and results, open tasks. Dense bullet points, no preamble, do not use tools. "

// cmdHandoff starts a fresh thread carrying over a summary of this one plus an
// optional goal (Amp-style handoff).
func cmdHandoff(m *Model, arg string) tea.Cmd {
	if len(m.entries) == 0 {
		m.info("nothing to hand off yet")
		return nil
	}
	if m.busy {
		m.errorf("wait for the current turn to finish (or esc) before /handoff")
		return nil
	}
	goal := arg
	m.handoffGoal = &goal
	m.shownAs = strings.TrimSpace("/handoff " + arg)
	m.compacting = true
	return m.startTurn(compactPrompt)
}

// finishCompact runs after the summary turn: the provider session is dropped
// and the summary seeds the next one, so the old history stops filling the
// window. /handoff uses the same path and adds a goal to the seed.
func (m *Model) finishCompact() {
	if !m.compacting {
		return
	}
	m.compacting = false
	if m.lastFailed {
		m.handoffGoal = nil
		return
	}
	sum := ""
	for i := len(m.entries) - 1; i >= 0; i-- {
		e := &m.entries[i]
		if e.kind == kCompact {
			sum = e.text
			break
		}
		if e.kind == kText {
			sum = e.text
			e.kind, e.result, e.done = kCompact, fmtK(m.ctxTokens)+"→"+fmtK(len(sum)/4), true
			break
		}
		if e.kind == kUser {
			break
		}
	}
	if sum == "" {
		m.handoffGoal = nil
		m.errorf("no summary came back · the conversation is unchanged")
		return
	}
	if m.turnProv != nil {
		delete(m.sessFor, m.turnProv.ID())
	}
	m.seed = "Context carried over from the earlier part of this NØX session (the full history was dropped to save context):\n<summary>\n" + sum + "\n</summary>"
	if m.handoffGoal != nil && *m.handoffGoal != "" {
		m.seed += "\nGoal for this thread: " + *m.handoffGoal
	}
	m.handoffGoal = nil
	m.ctxTokens = len(sum) / 4
	m.hinted = false
	m.todos = nil
	m.toast("context compacted · the next message starts from this summary")
}

func cmdClear(m *Model, _ string) tea.Cmd {
	m.save()
	m.resetConversation()
	return nil
}

func cmdResume(m *Model, _ string) tea.Cmd {
	if m.store == nil {
		m.info("sessions are not saved in this mode")
		return nil
	}
	list, err := m.store.List(m.dir)
	if err != nil || len(list) == 0 {
		m.info("no saved sessions for this directory yet")
		return nil
	}
	var items []pickItem
	for _, s := range list {
		if m.saved != nil && s.ID == m.saved.ID {
			continue
		}
		items = append(items, pickItem{value: s.ID, label: s.Title(),
			desc: fmt.Sprintf("%s · %s · %s", s.Updated.Format("Jan 2 15:04"), s.Provider, plural(s.Turns, "turn"))})
	}
	if len(items) == 0 {
		m.info("this is the only session here")
		return nil
	}
	m.picker = newPicker("Resume session", items, true, func(m *Model, it pickItem) tea.Cmd {
		s, err := m.store.Load(it.value)
		if err != nil {
			m.errorf(err.Error())
			return nil
		}
		m.save()
		m.resetConversation()
		m.restore(s)
		return nil
	})
	return nil
}

func cmdExport(m *Model, arg string) tea.Cmd {
	m.save()
	if m.saved == nil {
		m.info("nothing to export yet")
		return nil
	}
	path := arg
	if path == "" {
		path = "nox-" + m.saved.ID + ".md"
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(m.dir, path)
	}
	if err := os.WriteFile(path, []byte(m.saved.Markdown()), 0o644); err != nil {
		m.errorf(err.Error())
		return nil
	}
	m.info("exported → " + path)
	return nil
}

const initPrompt = `Create (or update) a NOX.md file in the project root. It is the instruction file NØX loads into every session, for every provider. Inspect the repository first, then write concise sections:
- Project: what it is, in two or three sentences.
- Stack and layout: languages, frameworks, the key directories.
- Commands: how to build, test, lint and run (exact commands).
- Conventions: code style, naming, patterns to follow, things to avoid.
- Notes for agents: anything non-obvious that would trip up a newcomer.
Keep it under 120 lines. If CLAUDE.md or AGENTS.md exist, reuse what is still accurate.`

func cmdInit(m *Model, _ string) tea.Cmd {
	if m.busy {
		m.errorf("wait for the current turn to finish first")
		return nil
	}
	if m.mode == agent.ModePlan || m.mode == agent.ModeManual {
		m.info("switching to auto mode so NØX can write NOX.md")
		m.mode = agent.ModeAuto
	}
	m.shownAs = "/init · write NOX.md for this project"
	return m.startTurn(initPrompt)
}

func cmdMemory(m *Model, _ string) tea.Cmd {
	g, p := agent.GlobalMemory(), agent.ProjectMemory(m.dir)
	show := func(s string) string {
		if s == "" {
			return "—"
		}
		return s
	}
	m.info(fmt.Sprintf("memory (sent with every turn)\n  global   %s\n  project  %s\n  /init writes NOX.md for this project; ~/.nox/NOX.md applies everywhere", show(g), show(p)))
	return nil
}

func cmdPermissions(m *Model, _ string) tea.Cmd {
	var names []string
	for k := range m.allow {
		names = append(names, k)
	}
	sort.Strings(names)
	s := "none"
	if len(names) > 0 {
		s = strings.Join(names, ", ")
	}
	m.info("always allowed this session: " + s + "\nmode " + m.mode.String() + " · manual asks for everything, auto asks only for risky commands, yolo never asks")
	return nil
}

func cmdKeys(m *Model, _ string) tea.Cmd {
	m.info(`keys
  enter          send            shift+enter / ctrl+j  newline
  shift+tab      cycle mode      ctrl+t                cycle effort
  ctrl+o         pick model
  tab / ↑↓       complete /cmd   ↑↓ (empty input)       history
  esc            interrupt (queue continues)   ctrl+c  stop all / clear / quit
  pgup/pgdn      scroll          ctrl+home/end          top / bottom
  wheel          scroll          shift+drag            select text
  ctrl+k         command palette ctrl+f               find in conversation
  alt+enter      queue a message for the running turn
leader: ctrl+x, then a key · change or add bindings in ~/.nox/keys.toml
` + keymapHelp(m.keymap))
	return nil
}

func cmdDoctor(m *Model, _ string) tea.Cmd {
	bin := local.Binary()
	if bin == "" {
		bin = "not found · build llama.cpp with Vulkan or set NOX_LLAMA_SERVER"
	}
	n := 0
	for _, p := range local.Catalog {
		if p.Installed() {
			n++
		}
	}
	models := fmt.Sprintf("%d downloaded · active %s", n, m.modelName())
	if n == 0 {
		models = "none downloaded · nox-local pull qwen3.5-4b"
	}
	mcp := "none configured"
	if m.mcp != nil {
		var names []string
		for _, s := range m.mcp.Servers() {
			names = append(names, s.Name)
		}
		if len(names) > 0 {
			mcp = strings.Join(names, ", ")
		}
	}
	m.info(strings.Join([]string{
		"doctor",
		fmt.Sprintf("  %-12s %s", "llama-server", bin),
		fmt.Sprintf("  %-12s %s", "models", models),
		fmt.Sprintf("  %-12s %s", "MCP", mcp),
		fmt.Sprintf("  %-12s %s", "config", "~/.nox/config.toml"),
	}, "\n"))
	return nil
}

func cmdRewind(m *Model, _ string) tea.Cmd {
	switch {
	case m.repo == nil:
		m.errorf("/rewind needs a git repository here (run git init once; NØX never commits for you)")
		return nil
	case len(m.points) == 0:
		m.info("no checkpoints yet · one is taken before every turn")
		return nil
	case m.busy:
		m.errorf("wait for the current turn to finish (or esc) before rewinding")
		return nil
	}
	var items []pickItem
	for i := len(m.points) - 1; i >= 0; i-- {
		p := m.points[i]
		label := strings.Join(strings.Fields(p.Label), " ")
		items = append(items, pickItem{value: fmt.Sprint(i), label: truncRight(label, 60),
			desc: "before turn · " + p.Taken.Format("15:04:05")})
	}
	m.picker = newPicker("Rewind files to before…", items, true, func(m *Model, it pickItem) tea.Cmd {
		var i int
		fmt.Sscan(it.value, &i)
		now, snapErr := m.repo.Snapshot("before rewind")
		touched, err := m.repo.Restore(m.points[i])
		if err != nil {
			m.errorf("rewind failed: " + err.Error())
			return nil
		}
		// remember enough to undo the rewind: the tree, the dropped
		// checkpoints and the change list
		if snapErr == nil {
			m.redo = &now
			m.redoPoints = append([]checkpoint.Point(nil), m.points[i:]...)
			m.redoChanges, m.redoOrder = maps.Clone(m.changes), slices.Clone(m.changeOrder)
		}
		m.points = m.points[:i]
		for _, t := range touched { // only the files the rewind put back
			delete(m.changes, t)
			m.changeOrder = slices.DeleteFunc(m.changeOrder, func(p string) bool { return p == t })
		}
		if len(touched) == 0 {
			m.info("nothing changed since then")
			return nil
		}
		m.info(fmt.Sprintf("rewound %s: %s\nthe conversation is kept; tell NØX what to do differently",
			plural(len(touched), "file"), strings.Join(touched, ", ")))
		return nil
	})
	return nil
}

func cmdMCP(m *Model, _ string) tea.Cmd {
	st := m.mcp.Servers()
	if len(st) == 0 {
		m.info("no MCP servers\n  add [mcp.<name>] with command/args or url to " + config.Path() + "\n  or put a .mcp.json in the project (used after /trust)")
		return nil
	}
	var rows []string
	for _, s := range st {
		switch {
		case s.Err != nil:
			rows = append(rows, fmt.Sprintf("  ✗ %-14s %s", s.Name, s.Err))
		case !s.Enabled:
			rows = append(rows, fmt.Sprintf("  ○ %-14s off · /mcp switches it on", s.Name))
		default:
			rows = append(rows, fmt.Sprintf("  ● %-14s %s", s.Name, plural(s.Tools, "tool")))
		}
	}
	m.info("MCP servers (small models cope with a few tools, not dozens)\n" + strings.Join(rows, "\n"))
	return nil
}

func cmdSkills(m *Model, _ string) tea.Cmd {
	sk := agent.Skills(m.dir)
	if len(sk) == 0 {
		m.info("no skills · put folders with a SKILL.md in ~/.nox/skills or .nox/skills (~/.claude/skills is shared too)")
		return nil
	}
	var items []pickItem
	for _, s := range sk {
		items = append(items, pickItem{value: s.Path, label: s.Name, desc: s.Source + " · " + s.Desc})
	}
	title := fmt.Sprintf("Skills · %s available to the model", plural(len(sk), "skill"))
	m.picker = newPicker(title, items, true, func(m *Model, it pickItem) tea.Cmd {
		m.ta.SetValue("Use the skill at " + it.value + " to " + m.ta.Value())
		m.ta.CursorEnd()
		return nil
	})
	return nil
}

func themePath() string { return paths.File("state", "theme") }

// loadTheme restores the theme picked last time.
func loadTheme() {
	if b, err := os.ReadFile(themePath()); err == nil {
		applyTheme(strings.TrimSpace(string(b)))
	}
}

func cmdTheme(m *Model, arg string) tea.Cmd {
	set := func(name string) {
		if !applyTheme(name) {
			m.errorf("unknown theme " + name)
			return
		}
		_ = os.WriteFile(themePath(), []byte(name), 0o644)
		m.dropRenderCache()
	}
	if arg != "" {
		set(arg)
		return nil
	}
	var items []pickItem
	for _, t := range themes {
		t := t
		items = append(items, pickItem{value: t.name, label: t.name, desc: t.desc, current: t.name == current.name,
			render: func(int) string { return gradient(t.name, hexes(t.a1, t.a2, t.a3), 0, false, true) }})
	}
	before := current.name
	m.picker = newPicker("Theme · ↑↓ previews live", items, false, func(m *Model, it pickItem) tea.Cmd {
		set(it.value)
		return nil
	})
	m.picker.move = func(m *Model, it pickItem) { applyTheme(it.value); m.dropRenderCache() }
	m.picker.cancel = func(m *Model) { applyTheme(before); m.dropRenderCache() }
	return nil
}

func cmdTodos(m *Model, _ string) tea.Cmd {
	if len(m.todos) == 0 {
		m.info("no task list yet · the agent writes one (TodoWrite) for multi-step work")
		return nil
	}
	done, total := todoCounts(m.todos)
	rows := []string{fmt.Sprintf("todos · %d/%d done", done, total)}
	for _, t := range m.todos {
		mark := "○"
		switch t.Status {
		case "completed":
			mark = "✓"
		case "in_progress":
			mark = "●"
		}
		rows = append(rows, "  "+mark+" "+t.Content)
	}
	m.info(strings.Join(rows, "\n"))
	return nil
}

// cmdFork copies the conversation (and API provider histories) into a new
// session and continues there; the original stays resumable as it was.
func cmdFork(m *Model, _ string) tea.Cmd {
	if m.busy {
		m.errorf("wait for the current turn to finish first")
		return nil
	}
	m.save()
	if m.saved == nil {
		m.info("nothing to fork yet")
		return nil
	}
	orig := m.saved.ID
	m.sessFor = map[string]string{} // the fork starts a fresh model session
	m.saved = nil
	m.save()
	m.info("forked · now in a new session; the original (" + orig + ") is unchanged and in /resume")
	return nil
}

func cmdRedo(m *Model, _ string) tea.Cmd {
	if m.redo == nil || m.repo == nil {
		m.info("nothing to redo · /redo works right after /rewind")
		return nil
	}
	touched, err := m.repo.Restore(*m.redo)
	if err != nil {
		m.errorf("redo failed: " + err.Error())
		return nil
	}
	m.redo = nil
	m.points = append(m.points, m.redoPoints...)
	if m.redoChanges != nil {
		m.changes, m.changeOrder = m.redoChanges, m.redoOrder
	}
	m.redoPoints, m.redoChanges, m.redoOrder = nil, nil, nil
	m.info("redo · restored " + plural(len(touched), "file") + " to before the rewind")
	return nil
}

// cmdRevert restores one file to the checkpoint taken before the latest turn.
func cmdRevert(m *Model, arg string) tea.Cmd {
	if m.repo == nil || len(m.points) == 0 {
		m.errorf("/revert needs a git repository and at least one turn")
		return nil
	}
	if arg == "" {
		var items []pickItem
		for _, p := range m.changeOrder {
			c := m.changes[p]
			items = append(items, pickItem{value: p, label: p, desc: fmt.Sprintf("+%d −%d", c.add, c.del)})
		}
		if len(items) == 0 {
			m.info("no changed files this session")
			return nil
		}
		m.picker = newPicker("Revert which file to before the turn that changed it?", items, true, func(m *Model, it pickItem) tea.Cmd {
			return cmdRevert(m, it.value)
		})
		return nil
	}
	abs := arg
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(m.dir, arg)
	}
	abs = filepath.Clean(abs)
	// the newest checkpoint where the file was different = before the last
	// turn that changed it
	at := -1
	for i := len(m.points) - 1; i >= 0; i-- {
		if m.repo.FileDiffers(m.points[i], abs) {
			at = i
			break
		}
	}
	if at < 0 {
		m.info(arg + " has not changed since the session started · nothing to revert")
		return nil
	}
	if err := m.repo.RestoreFile(m.points[at], abs); err != nil {
		m.errorf("revert: " + err.Error())
		return nil
	}
	rel := m.rel(abs)
	delete(m.changes, rel)
	for i, p := range m.changeOrder {
		if p == rel {
			m.changeOrder = append(m.changeOrder[:i:i], m.changeOrder[i+1:]...)
			break
		}
	}
	m.info(fmt.Sprintf("reverted %s to before the turn that changed it (%s)", rel, m.points[at].Taken.Format("15:04:05")))
	return nil
}

func (m *Model) dropRenderCache() {
	m.renderGen++
	for i := range m.entries {
		m.entries[i].cache, m.entries[i].prefixOut, m.entries[i].prefixLen = "", "", 0
	}
}

// cmdOpen hands a file (screenshot, HTML report, PDF) or URL to the desktop.
func cmdOpen(m *Model, arg string) tea.Cmd {
	if arg == "" {
		m.info("usage: /open <file or url> · browser screenshots are in ~/.nox/cache/shots")
		return nil
	}
	target := arg
	if !strings.Contains(arg, "://") && !filepath.IsAbs(arg) {
		target = filepath.Join(m.dir, arg)
	}
	opener := "xdg-open"
	switch runtime.GOOS {
	case "darwin":
		opener = "open"
	case "windows":
		opener = "explorer"
	}
	if strings.HasPrefix(arg, "-") {
		m.errorf("open: " + arg + " looks like an option, not a file")
		return nil
	}
	c := exec.Command(opener, target)
	if err := c.Start(); err != nil {
		m.errorf("open: " + err.Error())
		return nil
	}
	go c.Wait() // reap it
	m.info("opened " + arg)
	return nil
}

// cmdConfig opens the config in $EDITOR and reloads what can change live.
func cmdConfig(m *Model, arg string) tea.Cmd {
	path := config.Path()
	if strings.TrimSpace(arg) == "project" {
		path = filepath.Join(m.dir, ".nox", "config.toml")
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
	}
	ed := os.Getenv("VISUAL")
	if ed == "" {
		ed = os.Getenv("EDITOR")
	}
	if ed == "" {
		for _, c := range []string{"nvim", "vim", "nano", "vi"} {
			if _, err := exec.LookPath(c); err == nil {
				ed = c
				break
			}
		}
	}
	if ed == "" {
		m.info("config: " + path + " · set $EDITOR to edit it from here")
		return nil
	}
	parts := strings.Fields(ed)
	c := exec.Command(parts[0], append(parts[1:], path)...)
	return tea.ExecProcess(c, func(err error) tea.Msg { return configEditedMsg{err} })
}

type configEditedMsg struct{ err error }

func (m *Model) onConfigEdited(msg configEditedMsg) {
	if msg.err != nil {
		m.errorf("editor: " + msg.err.Error())
		return
	}
	m.keymap = loadKeymap()
	m.info("config saved · keys apply now; model, MCP and hooks after restarting nox-local")
}

func cmdTrust(m *Model, arg string) tea.Cmd {
	if strings.TrimSpace(arg) == "off" {
		if err := trust.Revoke(m.dir); err != nil {
			m.errorf(err.Error())
			return nil
		}
		m.info("project settings are off again from the next start")
		return nil
	}
	files := trust.Files(m.dir)
	if len(files) == 0 {
		m.info("this project has no NØX settings of its own · nothing to trust")
		return nil
	}
	if err := trust.Grant(m.dir); err != nil {
		m.errorf(err.Error())
		return nil
	}
	m.info("trusted · these files are used from the next start of nox (they can run commands):\n  " + strings.Join(files, "\n  ") + "\n  when they change, NØX asks again · /trust off withdraws it")
	return nil
}

func rulesOrNone(s string) string {
	if s == "" {
		return ": none"
	}
	return s
}
