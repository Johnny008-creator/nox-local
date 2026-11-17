// Package tui is the NØX terminal UI (design #12 "Halo").
package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Johnny008-creator/nox-local/internal/agent"
	"github.com/Johnny008-creator/nox-local/internal/checkpoint"
	"github.com/Johnny008-creator/nox-local/internal/hooks"
	"github.com/Johnny008-creator/nox-local/internal/mcpclient"
	"github.com/Johnny008-creator/nox-local/internal/paths"
	"github.com/Johnny008-creator/nox-local/internal/session"
	"github.com/Johnny008-creator/nox-local/internal/trust"
)

type entryKind int

const (
	kUser entryKind = iota
	kText
	kThinking
	kTool
	kInfo
	kError
	kCompact // context compacted: text = summary, result = "before→after"
	kBtw     // a /btw side answer: shown, but not part of the conversation
)

type entry struct {
	kind   entryKind
	text   string
	tool   agent.Tool
	result string
	isErr  bool
	done   bool
	start  time.Time
	dur    time.Duration
	prov   string // provider that produced it (user entries: who it was sent to)

	children  int    // sub-agent steps under this tool call
	lastChild string // the latest of them, for display

	cacheW    int // rendered-markdown cache for kText
	cacheLen  int
	cache     string
	expanded  bool   // tool row clicked open
	prefixLen int    // streaming: bytes of text rendered into prefixOut
	prefixOut string // rendered finished blocks

	// Rendered row, reused while nothing it depends on has changed (rowKey).
	// Without it every frame re-renders the whole conversation, which makes
	// long sessions stutter.
	row    string
	rowH   int
	rowKey rowKey
}

// rowKey is everything a finished row's rendering depends on.
type rowKey struct {
	w, gen                int
	done, isErr, expanded bool
	children              int
	lastChild, prov       string
	textLen, resultLen    int
	dur                   time.Duration
	first                 bool
}

type (
	// ch identifies the turn an event belongs to: events of a turn that was
	// abandoned (/clear, /resume while busy) are drained and ignored.
	eventMsg struct {
		ev agent.Event
		ch <-chan agent.Event
	}
	streamEndMsg struct{ ch <-chan agent.Event }
	tickMsg      time.Time
	modelsMsg    struct {
		prov   string
		models []agent.ModelInfo
		err    error
	}
)

type change struct{ add, del int }

var customOnce sync.Once

// Options configure a new TUI session.
type Options struct {
	Providers []agent.Provider // in display order
	Active    string           // provider id to start with
	Model     string
	Mode      agent.Mode
	Effort    agent.Effort
	Version   string
	Dir       string
	Resume    *session.Session // continue a saved session
	Store     *session.Store
	MCP       *mcpclient.Manager
	Hooks     []hooks.Hook
	// AutoCommit commits after every turn that changed files.
	AutoCommit bool
	// Interrupted is the id of a session whose NØX crashed or was killed.
	Interrupted string
	// Safe disables hooks, MCP, skills and custom commands (nox --safe).
	Safe bool
	// ConfigProblems are config.toml errors and warnings to show at start.
	ConfigProblems []string
}

// keyed marks a provider that authenticates with an API key. Local models
// need none, so every check for it is simply false in this build.
type keyed interface{ KeyEnv() string }

type Model struct {
	sel       selection  // mouse selection (select.go)
	reviewAsk *reviewAsk // pending ultracode review question
	screen    []string   // last rendered screen, for copying the selection
	w, h      int

	provs     []agent.Provider
	prov      agent.Provider
	model     string            // requested model for the active provider
	modelFor  map[string]string // remembered model per provider
	sessFor   map[string]string // provider session id per provider
	realModel string
	mode      agent.Mode
	effort    agent.Effort
	version   string
	dir       string

	entries  []entry
	vp       viewport.Model
	ta       textarea.Model
	loginFor keyed // non-nil while the input is a masked API-key field
	follow   bool

	busy   bool
	cancel context.CancelFunc
	events <-chan agent.Event
	queue  []string

	turnProv    agent.Provider // provider running the current turn
	reviewing   bool           // current turn is the ultracode review pass
	turnChanged int            // len(changeOrder) when the turn started
	turnSpoke   bool           // the turn produced text the user can read
	lastFailed  bool

	twin     bool // split view, see twin.go
	picker   *picker
	sugIdx   int
	approval *agent.Approval
	allow    map[string]bool // tools approved for the rest of the session
	// planFirst: every prompt plans read-only until /go approves the plan.
	planFirst    bool
	planPending  bool   // a plan is waiting for /go
	planText     string // the plan the agent wrote, sent back on /go
	turnPlanned  bool   // the running turn was a plan-first turn
	webOff       bool   // WebSearch and WebFetch are left out of every turn
	skipPlanOnce bool   // /go runs the approved plan without planning again

	turnStart  time.Time
	outChars   int  // characters generated this turn (answer, thinking, tool input)
	turnOutTok int  // exact output tokens reported by the provider this turn
	estFrom    int  // outChars already covered by turnOutTok
	argDeltas  bool // this turn's provider streams tool input (EvOutput)
	thinking   bool
	thinkStart time.Time
	thinkDur   time.Duration

	ctxTokens   int
	ctxWindow   int // reported by provider; 0 = use model info
	outTokens   int
	costUSD     float64
	turns       int
	limitUsed   float64
	limitWindow string
	sessStart   time.Time
	changes     map[string]change
	changeOrder []string
	modelCache  map[string][]agent.ModelInfo

	repo   *checkpoint.Repo   // nil outside git repos
	points []checkpoint.Point // working-tree snapshots, one per turn

	mcp           *mcpclient.Manager
	hooks         []hooks.Hook
	autoCommit    bool
	pending       []tea.Cmd // commands queued by event handlers (post_tool hooks)
	lastPrompt    string
	store         *session.Store
	saved         *session.Session
	history       []string
	histIdx       int
	frame         int
	fidx          fileIndex
	todos         []todo     // the agent's task list (EvTodos / TodoWrite)
	artifactsMade []artifact // artifacts the backend reported (EvArtifact)
	lastSave      time.Time
	compacting    bool    // the running turn writes the /compact summary
	research      bool    // the next turn runs as deep research
	hinted        bool    // the "time to /compact" note was shown for this conversation
	handoffGoal   *string // set while /handoff's summary turn runs
	seed          string  // context injected into the next fresh provider session
	lastSummary   string  // summary from the latest compaction
	shownAs       string  // label for the next user entry instead of the raw prompt
	toasts        []toastT
	goal          *goalT
	lspErr        map[string]int // file → LSP errors after its last edit
	turnFiles     []string       // files changed in the current turn (for review / auto-commit)
	redo          *checkpoint.Point
	cardRows      int    // diff card height limit while rendering an approval
	stash         string // draft kept aside while a command borrows the prompt
	splashAt      time.Time
	splashList    []*session.Session
	redoPoints    []checkpoint.Point
	redoChanges   map[string]change
	redoOrder     []string
	diff          *diffView // full-screen /diff viewer
	keymap        map[string]string
	vim           vimState
	plain         bool   // no animation, borders kept simple (/plain, NOX_PLAIN=1)
	batonFrom     string // provider we just switched from (header colour morph)
	batonStart    int
	rates         []int              // output chars per tick, for the sparkline
	entryRows     []int              // first rendered line of each entry (-1 = hidden)
	provNote      map[string]string  // provider → problem seen ("no credits")
	costBy        map[string]float64 // provider · model → cost this session
	lastState     string
	lastStatus    time.Time
	inbox         []string
	loop          *loopT
	plainTicks    int
	budgetWarn    bool    // session warning shown
	dayWarn       bool    // daily warning shown
	billedUSD     float64 // what this session really paid (no subscriptions / free)
	vpTop         int     // screen row where the conversation viewport starts
	renderGen     int     // bumped when every cached row must be redrawn (theme change)
	vpContent     string  // what the viewport currently shows, to skip resetting it
	lastOut       int
	leader        time.Time   // ctrl+x pressed; the next key is a leader binding
	steer         chan string // messages for the running turn (alt+enter)
	turnFrame     int         // m.frame when the turn started (staggered spinner entrance)
}

func New(o Options) *Model {
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.Placeholder = "Ask NØX anything · / for commands"
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("shift+enter", "ctrl+j"))
	st := textarea.DefaultDarkStyles()
	for _, s := range []*textarea.StyleState{&st.Focused, &st.Blurred} {
		s.Base = lipgloss.NewStyle()
		s.CursorLine = lipgloss.NewStyle()
		s.Text = sText
		s.Placeholder = sDim
		s.EndOfBuffer = sDim
	}
	ta.SetStyles(st)
	ta.Focus()

	m := &Model{
		provs:      o.Providers,
		model:      o.Model,
		modelFor:   map[string]string{},
		sessFor:    map[string]string{},
		mode:       o.Mode,
		effort:     o.Effort,
		version:    o.Version,
		dir:        o.Dir,
		vp:         viewport.New(),
		ta:         ta,
		follow:     true,
		sessStart:  time.Now(),
		changes:    map[string]change{},
		allow:      map[string]bool{},
		modelCache: map[string][]agent.ModelInfo{},
		keymap:     loadKeymap(),
		plain:      os.Getenv("NOX_PLAIN") != "",
		store:      o.Store,
		mcp:        o.MCP,
		hooks:      o.Hooks,
		autoCommit: o.AutoCommit,
	}
	customOnce.Do(func() {
		if !o.Safe {
			loadCustomCommands(o.Dir)
		}
		loadUserThemes(paths.Sub("themes"))
		loadTheme()
	})
	m.prov = m.provider(o.Active)
	if m.prov == nil {
		m.prov = o.Providers[0]
	}
	for _, p := range o.Providers {
		if k, ok := p.(keyed); ok {
			keyEnvOf[p.ID()] = k.KeyEnv()
		}
	}
	m.turnProv = m.prov
	m.repo = checkpoint.Open(o.Dir)
	if !m.prov.Available() {
		m.info("no model on this machine yet · nox-local pull qwen3.5-4b downloads one (2.9 GB) · /doctor checks the setup")
	}
	for _, p := range o.ConfigProblems {
		m.errorf("config: " + p + " · /config to fix")
	}
	if keymapErr != "" {
		m.errorf(keymapErr)
	}
	if trust.Needed(o.Dir) {
		m.info("this project ships NØX settings that can run commands (" + strings.Join(relAll(o.Dir, trust.Files(o.Dir)), ", ") + ") · they are off until you /trust it")
	}
	if o.Resume != nil {
		m.restore(o.Resume)
	} else if o.Interrupted != "" {
		m.info("the last NØX session here ended unexpectedly · /resume (or nox -c) picks it up again")
	}
	return m
}

func (m *Model) provider(id string) agent.Provider {
	for _, p := range m.provs {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

func tick() tea.Cmd {
	return tea.Tick(110*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func waitEvent(ch <-chan agent.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return streamEndMsg{ch}
		}
		return eventMsg{ev, ch}
	}
}

func fetchModels(p agent.Provider) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ms, err := p.Models(ctx)
		return modelsMsg{p.ID(), ms, err}
	}
}

// Init also asks every provider that is set up for the models this account
// can use, so a configured model that is gone (an expired plan, a removed
// model) is replaced before the first prompt instead of failing on it.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tick(), textarea.Blink, fetchModels(m.prov)}
	for _, p := range m.provs {
		if p != m.prov && p.Available() {
			cmds = append(cmds, fetchModels(p))
		}
	}
	return tea.Batch(cmds...)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		if !m.plain {
			m.frame++
		} else {
			m.plainTicks++
		}
		m.stepToasts()
		m.sampleRate()
		m.publishStatus()
		inboxCmd := m.checkInbox()
		return m, tea.Batch(tick(), m.dueLoop(), inboxCmd)

	case modelsMsg:
		if msg.err == nil {
			m.modelCache[msg.prov] = msg.models
			m.checkModel(msg.prov)
		}
		if m.picker != nil && m.picker.loading == msg.prov {
			if msg.err != nil {
				m.picker = nil
				m.errorf("could not list models: " + msg.err.Error())
			} else {
				m.openModelPicker()
			}
		}
		return m, nil

	case eventMsg:
		if msg.ch != nil && msg.ch != m.events { // a turn we already left: drain, ignore
			if msg.ev.Kind == agent.EvApproval {
				msg.ev.Approval.Reply <- false
			}
			return m, waitEvent(msg.ch)
		}
		if msg.ev.Kind == agent.EvApproval {
			m.onApproval(msg.ev.Approval)
		} else {
			m.handleEvent(msg.ev)
		}
		cmds := append(m.pending, waitEvent(m.events))
		m.pending = nil
		return m, tea.Batch(cmds...)

	case hookMsg:
		m.onHooks(msg)
		return m, nil

	case goalCheckMsg:
		return m, m.onGoalCheck(msg)

	case configEditedMsg:
		m.onConfigEdited(msg)
		return m, nil

	case btwDoneMsg:
		m.onBtw(msg)
		return m, nil

	case commitMsg:
		m.onCommit(msg)
		return m, nil

	case promptHookMsg:
		if failed := m.onHooks(hookMsg{"prompt", msg.res}); failed {
			m.info("prompt not sent: a prompt hook failed")
			return m, nil
		}
		return m, m.dispatch(msg.text)

	case streamEndMsg:
		if msg.ch != nil && msg.ch != m.events {
			return m, nil // end of an abandoned turn
		}
		long := time.Since(m.turnStart) > 20*time.Second
		m.finishTurn()
		m.onReviewDone()
		m.save()
		var bell tea.Cmd
		if long {
			bell = tea.Raw("\a") // flashes the taskbar / tab in Windows Terminal
		}
		bell = tea.Batch(bell, m.stopHooks(), m.maybeCommit())
		if cmd := m.afterTurn(); cmd != nil {
			return m, tea.Batch(cmd, bell)
		}
		if cmd := m.goalAfterTurn(); cmd != nil {
			return m, tea.Batch(cmd, bell)
		}
		if len(m.queue) == 0 && bell != nil {
			return m, bell
		}
		if len(m.queue) > 0 {
			next := m.queue[0]
			m.queue = m.queue[1:]
			if p, rest, ok := m.mention(next); ok {
				return m, tea.Batch(m.startTurnOn(p, rest, m.mode, false), bell)
			}
			return m, tea.Batch(m.startTurn(next), bell) // stop hooks and auto-commit still run
		}
		return m, bell

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && m.diff == nil && m.picker == nil {
			m.sel.press(msg.X, msg.Y)
		}
		return m, nil

	case tea.MouseMotionMsg:
		m.sel.drag(msg.X, msg.Y)
		return m, nil

	case tea.MouseReleaseMsg:
		if !m.sel.down {
			return m, nil
		}
		m.sel.down = false
		if !m.sel.moved { // a plain click
			m.clickAt(msg.X, msg.Y)
			return m, nil
		}
		if text := m.selectedText(); text != "" {
			m.toast(fmt.Sprintf("copied %d characters", len([]rune(text))))
			return m, tea.SetClipboard(text)
		}
		return m, nil

	case tea.MouseWheelMsg:
		if m.diff != nil {
			var cmd tea.Cmd
			m.diff.vp, cmd = m.diff.vp.Update(msg)
			return m, cmd
		}
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		m.follow = m.vp.AtBottom()
		return m, cmd

	case tea.KeyPressMsg:
		m.sel = selection{} // typing clears a mouse selection
		if m.reviewAsk != nil && m.approval == nil && m.diff == nil && m.picker == nil {
			if cmd, handled := m.answerReview(msg.String()); handled {
				return m, cmd
			}
		}
		if m.diff != nil && m.approval == nil {
			return m, m.diffKey(msg)
		}
		if m.approval != nil {
			return m, m.approvalKey(msg)
		}
		if m.picker != nil {
			return m, m.pickerKey(msg)
		}
		if cmd, handled := m.handleKey(msg); handled {
			return m, cmd
		}
	}

	var cmd tea.Cmd
	before := m.ta.Value()
	m.ta, cmd = m.ta.Update(msg)
	if m.ta.Value() != before {
		m.sugIdx = 0
	}
	m.ta.SetHeight(min(max(strings.Count(m.ta.Value(), "\n")+1, 1), 6))
	return m, cmd
}

func (m *Model) handleKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	if cmd, ok := m.vimKey(k); ok {
		return cmd, true
	}
	if !m.leader.IsZero() && time.Since(m.leader) < 3*time.Second {
		return m.leaderKey(k.String()), true
	}
	m.leader = time.Time{}
	switch k.String() {
	case "ctrl+x":
		m.leader = time.Now()
		return nil, true
	case "ctrl+k":
		return m.openPalette(), true
	}
	if a, ok := m.keymap[k.String()]; ok {
		return m.runBinding(a), true
	}
	if at := m.atSuggestions(); len(at) > 0 {
		switch k.String() {
		case "up":
			m.sugIdx = (m.sugIdx - 1 + len(at)) % len(at)
			return nil, true
		case "down":
			m.sugIdx = (m.sugIdx + 1) % len(at)
			return nil, true
		case "tab", "enter":
			m.completeAt(at[m.sugIdx%len(at)])
			m.sugIdx = 0
			return nil, true
		case "esc":
			m.completeAt(atItem{insert: ""})
			return nil, true
		}
	}
	sugg := m.suggestions()
	switch k.String() {
	case "ctrl+c":
		if m.busy {
			if len(m.queue) > 0 {
				m.info("stopped · " + plural(len(m.queue), "queued prompt") + " dropped")
			}
			m.stopAll()
			return nil, true
		}
		if m.ta.Value() != "" {
			m.ta.Reset()
			return nil, true
		}
		return m.quit(), true
	case "esc":
		if m.busy {
			m.interrupt()
		} else if len(sugg) > 0 {
			m.ta.Reset()
		}
		return nil, true
	case "shift+tab":
		m.mode = m.mode.Next()
		return nil, true
	case "tab":
		if len(sugg) > 0 {
			m.ta.SetValue("/" + sugg[m.sugIdx%len(sugg)].name + " ")
			m.ta.CursorEnd()
		}
		return nil, true
	case "enter":
		text := m.ta.Value()
		if len(sugg) > 0 {
			text = "/" + sugg[m.sugIdx%len(sugg)].name
		}
		m.ta.Reset()
		m.ta.SetHeight(1)
		m.histIdx, m.sugIdx = 0, 0
		return m.submit(text), true
	case "up":
		if len(sugg) > 0 {
			m.sugIdx = (m.sugIdx - 1 + len(sugg)) % len(sugg)
			return nil, true
		}
		if m.ta.Line() == 0 && len(m.history) > 0 {
			m.histIdx = min(m.histIdx+1, len(m.history))
			m.ta.SetValue(m.history[len(m.history)-m.histIdx])
			return nil, true
		}
	case "down":
		if len(sugg) > 0 {
			m.sugIdx = (m.sugIdx + 1) % len(sugg)
			return nil, true
		}
		if m.histIdx > 0 && m.ta.Line() == m.ta.LineCount()-1 {
			m.histIdx--
			if m.histIdx == 0 {
				m.ta.Reset()
			} else {
				m.ta.SetValue(m.history[len(m.history)-m.histIdx])
			}
			return nil, true
		}
	case "ctrl+t":
		m.effort = m.effort.Next()
		return nil, true
	case "ctrl+f":
		return cmdFind(m, ""), true
	case "ctrl+v":
		return nil, false // no image paste: local models take text only
	case "alt+enter":
		text := strings.TrimSpace(m.ta.Value())
		if !m.busy || text == "" {
			return nil, false
		}
		m.ta.Reset()
		return m.onSubmitWhileBusy(text), true
	case "ctrl+o":
		return m.openModelPicker(), true
	case "pgup":
		m.vp.PageUp()
		m.follow = m.vp.AtBottom()
		return nil, true
	case "pgdown":
		m.vp.PageDown()
		m.follow = m.vp.AtBottom()
		return nil, true
	case "ctrl+home":
		m.vp.GotoTop()
		m.follow = false
		return nil, true
	case "ctrl+end":
		m.vp.GotoBottom()
		m.follow = true
		return nil, true
	}
	return nil, false
}

func (m *Model) submit(text string) tea.Cmd {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	m.history = append(m.history, text)
	if strings.HasPrefix(text, "/") {
		cmd := m.runCommand(text)
		if m.stash != "" && m.ta.Value() == "" { // give back the borrowed draft
			m.ta.SetValue(m.stash)
			m.ta.CursorEnd()
		}
		m.stash = ""
		return cmd
	}
	if hs := hooks.Select(m.hooks, "prompt", ""); len(hs) > 0 {
		v := hooks.Vars{Event: "prompt", Prompt: text, Dir: m.dir}
		return func() tea.Msg { return promptHookMsg{text, hooks.Run(hs, v)} }
	}
	return m.dispatch(text)
}

// dispatch sends a prompt that passed the prompt hooks.
func (m *Model) dispatch(text string) tea.Cmd {
	m.lastPrompt = text
	if p, rest, ok := m.mention(text); ok {
		return m.sendMention(p, rest)
	}
	if m.busy {
		return m.onSubmitWhileBusy(text)
	}
	return m.startTurn(text)
}

// onSubmitWhileBusy queues a prompt sent while NØX is still working. It runs
// as soon as the current turn ends, whether it finishes or you interrupt it
// with Esc. Ctrl+C stops the turn and drops the queue.
func (m *Model) onSubmitWhileBusy(text string) tea.Cmd {
	m.queue = append(m.queue, text)
	m.info("queued · runs next (esc skips to it, ctrl+c drops the queue · " + plural(len(m.queue), "prompt") + " waiting)")
	return nil
}

func (m *Model) startTurn(text string) tea.Cmd {
	return m.startTurnOn(m.prov, text, m.mode, false)
}

// startTurnOn runs one turn on p (the active provider, an @mention, or the
// ultracode reviewer). review marks the automatic second-opinion pass.
func (m *Model) startTurnOn(p agent.Provider, text string, mode agent.Mode, review bool) tea.Cmd {
	// one-shot flags of a /compact or /handoff: a turn that cannot start
	// must not leave them for the next prompt
	clearOneShot := func() {
		m.shownAs, m.compacting = "", false
		m.handoffGoal = nil
	}
	shown := text
	if review {
		shown = "review by " + p.Name()
	}
	if m.shownAs != "" {
		shown = m.shownAs
	}
	if m.compacting {
		shown = "/compact"
	}
	prompt := m.expandFiles(text)
	// Plan first: the turn runs read-only and its answer is the plan. It is
	// stored so /go can send it back in the real mode — the same shape the
	// desktop uses, so both windows behave alike.
	planning := m.planFirst && !m.skipPlanOnce && !review && !m.compacting
	m.skipPlanOnce = false
	m.turnPlanned = planning
	if planning {
		// Provider-neutral: the guide goes into the prompt itself, which every
		// adapter sends, rather than into a system extra only some honour.
		mode = agent.ModePlan
		m.planPending = false
		prompt = planFirstGuide + "\n\n# The user's message\n" + prompt
		shown += "  [plan first]"
	}
	sid := m.sessFor[p.ID()]
	usedSeed := false
	if sid == "" && m.seed != "" {
		prompt, usedSeed = m.seed+"\n\n"+prompt, true
	}
	model := m.model
	if p != m.prov {
		model = m.modelFor[p.ID()]
	}
	var point *checkpoint.Point
	if m.repo != nil && !review {
		if pt, err := m.repo.Snapshot(text); err == nil {
			point = &pt
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	research := m.research
	m.research = false
	var without []string
	if m.webOff {
		without = []string{"WebSearch", "WebFetch"}
	}
	ch, err := p.Run(ctx, agent.Request{
		Prompt: prompt, Model: model, SessionID: sid,
		Dir: m.dir, Mode: mode, Effort: m.effort, Research: research, Without: without,
	})
	m.entries = append(m.entries, entry{kind: kUser, text: shown, prov: p.ID()})
	if err != nil {
		cancel()
		clearOneShot() // the seed stays for the next try
		if !p.Available() {
			m.errorf(p.Name() + " is not set up · " + setupHint(p.ID()))
		} else {
			m.errorf(err.Error())
		}
		return nil
	}
	// the turn runs: now consume what it took
	m.shownAs = ""
	if usedSeed {
		m.seed = ""
	}
	if point != nil {
		m.points = append(m.points, *point)
		if len(m.points) > 100 {
			m.points = m.points[1:]
		}
	}
	m.turnProv, m.reviewing = p, review
	if !review {
		m.redo = nil // a new prompt makes the rewound state final
	}
	m.turnChanged = len(m.changeOrder)
	m.turnFiles = nil
	m.cancel, m.events, m.busy, m.lastFailed = cancel, ch, true, false
	m.turnStart, m.outChars, m.thinkDur, m.thinking = time.Now(), 0, 0, false
	m.turnOutTok, m.estFrom, m.argDeltas = 0, 0, false
	m.turnSpoke = false
	m.turnFrame = m.frame
	m.rates, m.lastOut = nil, 0
	m.follow = true
	m.turns++
	return waitEvent(ch)
}

// handoff gives a provider that joins mid-conversation a transcript of what
// was said with other providers, so switching keeps the thread.
func (m *Model) handoff(pid string) string {
	var b strings.Builder
	for _, e := range m.entries { // called before the new prompt's entry is added
		if e.prov == pid {
			continue
		}
		switch e.kind {
		case kUser:
			fmt.Fprintf(&b, "USER: %s\n", e.text)
		case kText:
			fmt.Fprintf(&b, "NØX (%s): %s\n", e.prov, e.text)
		case kTool:
			fmt.Fprintf(&b, "  [tool %s %s]\n", e.tool.Name, m.toolTarget(e.tool))
		}
	}
	if b.Len() == 0 {
		return ""
	}
	t := b.String()
	if len(t) > 16000 {
		t = "…" + t[len(t)-16000:]
	}
	return "Earlier in this NØX session (handled by another provider):\n<transcript>\n" + t + "</transcript>\n\nContinue from there. New message:\n"
}

func (m *Model) interrupt() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.approval != nil {
		m.approval.Reply <- false
		m.approval = nil
	}
}

// stopAll interrupts the turn and forgets queued prompts.
func (m *Model) stopAll() {
	m.queue = nil
	m.interrupt()
}

func (m *Model) finishTurn() {
	m.endThinking()
	for i := range m.entries {
		if m.entries[i].kind == kTool && !m.entries[i].done {
			m.entries[i].done, m.entries[i].isErr = true, true
			m.entries[i].result = "interrupted"
		}
	}
	m.busy, m.cancel, m.events, m.approval = false, nil, nil, nil
	m.turnPlanned = false
}

func (m *Model) handleEvent(ev agent.Event) {
	if ev.ContextWindow > 0 {
		m.ctxWindow = ev.ContextWindow
	}
	pid := m.turnProv.ID()
	switch ev.Kind {
	case agent.EvSession:
		if ev.SessionID != "" {
			m.sessFor[pid] = ev.SessionID
		}
		if ev.Model != "" {
			m.realModel = ev.Model
		}
	case agent.EvText:
		m.endThinking()
		m.outChars += len(ev.Text)
		if strings.TrimSpace(ev.Text) != "" {
			m.turnSpoke = true
		}
		if n := len(m.entries); n > 0 && m.entries[n-1].kind == kText {
			m.entries[n-1].text += ev.Text
		} else {
			m.entries = append(m.entries, entry{kind: kText, text: strings.TrimLeft(ev.Text, "\n"), prov: pid})
		}
	case agent.EvThinking:
		m.outChars += len(ev.Text)
		if !m.thinking {
			m.thinking, m.thinkStart = true, time.Now()
			m.entries = append(m.entries, entry{kind: kThinking, start: m.thinkStart, prov: pid})
		}
	case agent.EvToolStart:
		if ev.Tool.Parent != "" { // a sub-agent step: fold it into its parent row
			for i := len(m.entries) - 1; i >= 0; i-- {
				if e := &m.entries[i]; e.kind == kTool && e.tool.ID == ev.Tool.Parent {
					e.children++
					e.lastChild = toolLabel(ev.Tool.Name) + " " + m.toolTarget(ev.Tool)
					break
				}
			}
			return
		}
		m.endThinking()
		m.noteTodos(ev.Tool)
		if !m.argDeltas { // CLI providers send the whole input at once: count it here
			if b, err := json.Marshal(ev.Tool.Input); err == nil {
				m.outChars += len(b)
			}
		}
		m.entries = append(m.entries, entry{kind: kTool, tool: ev.Tool, start: time.Now(), prov: pid})
	case agent.EvToolResult:
		for i := len(m.entries) - 1; i >= 0; i-- {
			e := &m.entries[i]
			if e.kind == kTool && e.tool.ID == ev.Tool.ID {
				e.result, e.isErr, e.done, e.dur = ev.Result, ev.IsError, true, time.Since(e.start)
				if !e.isErr {
					if p, ok := e.tool.Input["file_path"].(string); ok && (e.tool.Name == "Edit" || e.tool.Name == "Write" || e.tool.Name == "MultiEdit") {
						if p = m.rel(p); !slices.Contains(m.turnFiles, p) {
							m.turnFiles = append(m.turnFiles, p)
						}
					}
					m.trackChange(e.tool)
					m.postToolHooks(e.tool)
					if p, ok := e.tool.Input["file_path"].(string); ok && (e.tool.Name == "Edit" || e.tool.Name == "Write") {
						if m.lspErr == nil {
							m.lspErr = map[string]int{}
						}
						m.lspErr[p] = lspErrors(e.result)
					}
				}
				if time.Since(m.lastSave) > 3*time.Second { // crash safety mid-turn
					m.save()
				}
				break
			}
		}
	case agent.EvContext:
		if t := ev.Usage.ContextTokens(); t > 0 {
			m.ctxTokens = t
		}
		if ev.Usage.OutputTokens > 0 { // exact count for what was generated so far
			m.turnOutTok += ev.Usage.OutputTokens
			m.estFrom = m.outChars
		}
		if ev.Model != "" {
			m.realModel = ev.Model
		}
	case agent.EvToolInput:
		m.outChars += len(ev.Text)
		m.argDeltas = true
	case agent.EvTodos:
		m.todos = agentTodos(ev.Todos)
	case agent.EvArtifact:
		if a := ev.Artifact; a != nil {
			m.artifactsMade = append(m.artifactsMade, artifactOf(*a))
			m.info(fmt.Sprintf("artifact · %s (v%d) · /artifacts opens it", a.Title, a.Version))
		}
	case agent.EvDone:
		m.endThinking()
		// A plan-first turn ended: the last answer is the plan, and /go will
		// send it back. An error leaves nothing to approve.
		if m.turnPlanned && !ev.IsError {
			if plan := m.lastText(); strings.TrimSpace(plan) != "" {
				m.planPending, m.planText = true, plan
			}
		}
		m.outTokens += ev.Usage.OutputTokens
		if ev.SessionID != "" {
			m.sessFor[pid] = ev.SessionID
		}
		if ev.IsError {
			m.lastFailed = true
			m.errorf(ev.Text)
		}
		if !ev.IsError && !m.reviewing && !m.lastFailed && !m.turnSpoke && strings.TrimSpace(m.lastText()) == "" {
			// A provider that owns its own loop can still end a turn with words
			// in the tool rows and nothing else. Say it rather than leave a
			// finished-looking turn with no answer in it.
			m.info("the turn ended with no answer · the tool rows above are all there is (esc stops the turn; try again or ask for the answer)")
		}
		m.finishCompact()
	case agent.EvError:
		m.lastFailed = true
		m.errorf(ev.Err.Error())
	}
}

func (m *Model) endThinking() {
	if !m.thinking {
		return
	}
	m.thinking = false
	d := time.Since(m.thinkStart)
	m.thinkDur += d
	for i := len(m.entries) - 1; i >= 0; i-- {
		if m.entries[i].kind == kThinking && !m.entries[i].done {
			m.entries[i].done, m.entries[i].dur = true, d
			break
		}
	}
}

func (m *Model) trackChange(t agent.Tool) {
	path, _ := t.Input["file_path"].(string)
	if path == "" {
		return
	}
	var add, del int
	for _, d := range diffFor(t) {
		switch d.sign {
		case '+':
			add++
		case '-':
			del++
		}
	}
	if add == 0 && del == 0 && t.Name != "Patch" { // reads, or edits we can't size
		return
	}
	path = m.rel(path)
	if _, seen := m.changes[path]; !seen {
		m.changeOrder = append(m.changeOrder, path)
	}
	c := m.changes[path]
	m.changes[path] = change{c.add + add, c.del + del}
}

// ---------- approvals (manual / auto mode, native loop) ----------

func (m *Model) onApproval(a *agent.Approval) {
	scope := approvalScopeOf(a)
	// yolo skips mode questions, never the ones your own rules or a second
	// opinion raise.
	if (scope != "" && m.allow[scope]) || (m.mode == agent.ModeYolo && !a.Rule) {
		a.Reply <- true
		return
	}
	m.approval = a
	m.follow = true
}

func (m *Model) approvalKey(k tea.KeyPressMsg) tea.Cmd {
	a := m.approval
	switch strings.ToLower(k.String()) {
	case "y", "enter":
		a.Reply <- true
	case "a":
		if s := approvalScopeOf(a); s != "" {
			m.allow[s] = true
		}
		a.Reply <- true
	case "n", "esc":
		a.Reply <- false
	case "ctrl+c":
		m.stopAll()
		return nil
	default:
		return nil
	}
	m.approval = nil
	return nil
}

// ---------- model / provider helpers ----------

func (m *Model) info(s string) {
	m.entries = append(m.entries, entry{kind: kInfo, text: s})
	m.follow = true
}

func (m *Model) errorf(s string) {
	m.entries = append(m.entries, entry{kind: kError, text: s})
	m.follow = true
}

func (m *Model) currentModel() string {
	if m.model == "" {
		return m.prov.DefaultModel()
	}
	return m.model
}

func (m *Model) modelInfo() (agent.ModelInfo, bool) {
	cur := m.currentModel()
	for _, mi := range m.modelCache[m.prov.ID()] {
		if mi.ID == cur {
			return mi, true
		}
	}
	return agent.ModelInfo{}, false
}

func (m *Model) modelName() string {
	if m.realModel != "" {
		return prettyModel(m.realModel)
	}
	if mi, ok := m.modelInfo(); ok && mi.Name != "" && mi.Name != mi.ID {
		return mi.Name
	}
	return m.currentModel()
}

// contextMax is the active model's window: provider-reported first, then the
// model catalog, then a conservative default.
func (m *Model) contextMax() int {
	if m.ctxWindow > 0 {
		return m.ctxWindow
	}
	if mi, ok := m.modelInfo(); ok && mi.Context > 0 {
		return mi.Context
	}
	return 200_000
}

func (m *Model) setModel(id string) {
	m.model, m.realModel, m.ctxWindow = id, "", 0
	m.modelFor[m.prov.ID()] = id
}

func (m *Model) switchProvider(id string) tea.Cmd {
	p := m.provider(id)
	if p == nil || p == m.prov {
		return nil
	}
	if m.busy {
		m.errorf("wait for the current turn to finish (or esc) before switching provider")
		return nil
	}
	if !p.Available() {
		m.errorf(p.Name() + " is not set up · " + setupHint(p.ID()))
		return nil
	}
	m.modelFor[m.prov.ID()] = m.model
	m.batonFrom, m.batonStart = m.prov.ID(), m.frame
	m.prov = p
	m.model, m.realModel, m.ctxWindow, m.ctxTokens = m.modelFor[id], "", 0, 0
	m.limitWindow = ""
	m.info("provider → " + p.Name() + " · the conversation so far is handed over on your next message")
	if _, ok := m.modelCache[id]; !ok {
		return fetchModels(p)
	}
	return nil
}

// keyEnvOf maps API provider ids to their key variable (for hints).
var keyEnvOf = map[string]string{}

// SetupHint tells how to make provider id available (used by nox doctor).
func SetupHint(id string) string { return setupHint(id) }

func setupHint(string) string {
	return "nox-local pull qwen3.5-4b · or set NOX_LLAMA_SERVER to a llama-server build"
}

// ---------- persistence ----------

func (m *Model) save() {
	if m.store == nil || len(m.entries) == 0 {
		return
	}
	if m.saved == nil {
		m.saved = m.store.New(m.dir)
	}
	s := m.saved
	s.Provider, s.Model, s.Mode, s.Effort = m.prov.ID(), m.model, m.mode.String(), m.effort.String()
	s.ProviderSessions = m.sessFor
	s.Entries = s.Entries[:0]
	for _, e := range m.entries {
		s.Entries = append(s.Entries, session.Entry{
			Kind: int(e.kind), Text: e.text, Provider: e.prov, ToolID: e.tool.ID, ToolName: e.tool.Name,
			ToolInput: e.tool.Input, Result: e.result, IsError: e.isErr, DurationMS: e.dur.Milliseconds(),
		})
	}
	s.Turns, s.CostUSD, s.OutputTokens = m.turns, m.costUSD, m.outTokens
	m.lastSave = time.Now()
	if err := m.store.Save(s); err != nil {
		m.errorf("could not save session: " + err.Error())
	}
	session.MarkRunning(m.dir, s.ID)
}

func (m *Model) restore(s *session.Session) {
	m.saved = s
	if p := m.provider(s.Provider); p != nil && p.Available() {
		m.prov = p
		m.model = s.Model
	} else if s.Provider != "" {
		m.info("this session ran on " + s.Provider + ", which is not available here · continuing on " + m.prov.Name() + " with its own model")
	}
	if s.Dir != "" && s.Dir != m.dir {
		m.info("this session was saved in " + s.Dir + " · files it mentions may not exist here")
		s.Dir = m.dir // continue it here from now on
	}
	if md, ok := agent.ParseMode(s.Mode); ok {
		m.mode = md
	}
	if ef, ok := agent.ParseEffort(s.Effort); ok {
		m.effort = ef
	}
	for k, v := range s.ProviderSessions {
		m.sessFor[k] = v
	}
	m.entries = nil
	for _, e := range s.Entries {
		m.entries = append(m.entries, entry{
			kind: entryKind(e.Kind), text: e.Text, prov: e.Provider, result: e.Result, isErr: e.IsError, done: true,
			tool: agent.Tool{ID: e.ToolID, Name: e.ToolName, Input: e.ToolInput}, dur: time.Duration(e.DurationMS) * time.Millisecond,
		})
		if e.Kind == int(kTool) && !e.IsError {
			m.trackChange(agent.Tool{Name: e.ToolName, Input: e.ToolInput})
			m.noteTodos(agent.Tool{Name: e.ToolName, Input: e.ToolInput})
		}
	}
	m.turns, m.costUSD, m.outTokens = s.Turns, s.CostUSD, s.OutputTokens
	m.info("resumed session " + s.Title())
}

// clickAt toggles the tool row under the mouse between preview and full output.
func (m *Model) clickAt(x, y int) {
	if m.twin {
		return
	}
	line := m.vp.YOffset() + y - m.vpTop
	if y < m.vpTop || y >= m.vpTop+m.vp.Height() || x > m.vp.Width()+1 {
		return
	}
	hit := -1
	for i, r := range m.entryRows {
		if r >= 0 && r <= line {
			hit = i
		}
	}
	if hit >= 0 && (m.entries[hit].kind == kTool || m.entries[hit].kind == kCompact) && m.entries[hit].done {
		m.entries[hit].expanded = !m.entries[hit].expanded
		m.follow = false
	}
}

// turnTokens is the output of the running turn: exact counts the provider
// reported, plus an estimate (≈4 characters per token) for what streamed
// since — answer text, reasoning and tool input such as code and commands.
func (m *Model) turnTokens() int {
	return m.turnOutTok + max(m.outChars-m.estFrom, 0)/4
}

// approvalScopeOf is the key "a" (always) stores: a command prefix or file,
// "" when the action can't be pre-approved (compound shell commands).
func approvalScopeOf(a *agent.Approval) string {
	if a.Scope != "" {
		return a.Scope
	}
	if a.Tool.Name == "Bash" || a.Tool.Name == "Terminal" {
		cmd, _ := a.Tool.Input["command"].(string)
		if strings.ContainsAny(cmd, "|&;<>`$\n") { // compound: never pre-approve
			return ""
		}
		if f := strings.Fields(cmd); len(f) > 0 {
			return "Bash:" + f[0]
		}
		return ""
	}
	return a.Tool.Name
}

// scopeLabel describes an approval scope for people.
func scopeLabel(s string) string {
	switch {
	case strings.HasPrefix(s, "Bash:"):
		return "`" + strings.TrimPrefix(s, "Bash:") + " …` commands"
	case strings.HasPrefix(s, "Edit:"):
		return "edits to " + filepath.Base(strings.TrimPrefix(s, "Edit:"))
	}
	return s
}

func relAll(dir string, ps []string) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		if r, err := filepath.Rel(dir, p); err == nil {
			out[i] = r
		} else {
			out[i] = p
		}
	}
	return out
}

// abandonTurn stops the running turn and detaches from it: whatever it
// still sends is drained and ignored.
func (m *Model) abandonTurn() {
	m.queue = nil
	m.interrupt()
	m.finishTurn()
}

// resetConversation clears everything that belongs to one conversation.
// /clear, /resume, /search and the palette all go through it.
func (m *Model) resetConversation() {
	if m.busy {
		m.abandonTurn()
	}
	m.entries, m.queue, m.saved = nil, nil, nil
	m.sessFor = map[string]string{}
	m.ctxTokens, m.ctxWindow, m.realModel = 0, 0, ""
	m.thinkDur, m.turns, m.costUSD, m.outTokens = 0, 0, 0, 0
	m.changes, m.changeOrder = map[string]change{}, nil
	m.todos, m.artifactsMade, m.lspErr, m.costBy = nil, nil, nil, nil
	m.seed, m.lastSummary, m.shownAs = "", "", ""
	m.compacting, m.hinted = false, false
	m.billedUSD = 0
	m.handoffGoal, m.goal, m.redo = nil, nil, nil
	m.redoPoints, m.redoChanges, m.redoOrder = nil, nil, nil
	m.points = nil
	m.limitWindow, m.limitUsed = "", 0
	m.turnChanged, m.turnFiles = 0, nil
	m.approval = nil
	m.sessStart = time.Now()
}

// quit saves and stops everything this window started: the running turn
// and any work the agent left running in the background.
func (m *Model) quit() tea.Cmd {
	if m.busy {
		m.abandonTurn()
	}
	m.save()
	return tea.Quit
}

// checkModel replaces the model a provider would use when the account does
// not offer it any more.
func (m *Model) checkModel(pid string) {
	p := m.provider(pid)
	if p == nil {
		return
	}
	want := m.modelFor[pid]
	if p == m.prov {
		want = m.model
	}
	use, replaced := agent.PickModel(p, m.modelCache[pid], want)
	if !replaced {
		return
	}
	old := want
	if old == "" {
		old = p.DefaultModel()
	}
	m.modelFor[pid] = use
	if p == m.prov {
		m.model, m.realModel = use, ""
	}
	m.info(fmt.Sprintf("%s: %s is not available on your account · using %s (ctrl+o picks another)", p.Name(), old, use))
}
