// Package agent defines what the TUI and the agent loop share: modes, effort
// levels, requests and the event stream a turn emits.
package agent

import (
	"context"
	"strings"
)

// Mode controls what the agent may do without asking. Shift+Tab cycles it.
type Mode int

const (
	ModeManual Mode = iota
	ModePlan
	ModeAuto
	ModeYolo
)

var Modes = []Mode{ModeManual, ModePlan, ModeAuto, ModeYolo}

func (m Mode) String() string {
	switch m {
	case ModeManual:
		return "manual"
	case ModePlan:
		return "plan"
	case ModeAuto:
		return "auto"
	case ModeYolo:
		return "yolo"
	}
	return "unknown"
}

// Label is the status-line text, e.g. "auto mode on".
func (m Mode) Label() string {
	if m == ModeManual {
		return "manual mode"
	}
	return m.String() + " mode on"
}

// Glyph is the status-line icon for the mode.
func (m Mode) Glyph() string {
	switch m {
	case ModePlan:
		return "⏸"
	case ModeAuto:
		return "⏵⏵"
	case ModeYolo:
		return "⏵⏵⏵"
	}
	return "◇"
}

func (m Mode) Next() Mode { return Modes[(int(m)+1)%len(Modes)] }

func ParseMode(s string) (Mode, bool) {
	for _, m := range Modes {
		if strings.EqualFold(s, m.String()) {
			return m, true
		}
	}
	return 0, false
}

// Effort is how much a local model may think: low answers directly,
// medium and high let it reason first (high with a bigger thinking budget).
type Effort int

const (
	EffortLow Effort = iota
	EffortMedium
	EffortHigh
)

var Efforts = []Effort{EffortLow, EffortMedium, EffortHigh}

func (e Effort) String() string {
	return [...]string{"low", "medium", "high"}[e]
}

func (e Effort) Next() Effort { return Efforts[(int(e)+1)%len(Efforts)] }

func ParseEffort(s string) (Effort, bool) {
	for _, e := range Efforts {
		if strings.EqualFold(s, e.String()) {
			return e, true
		}
	}
	return 0, false
}

// Request is one user turn handed to a provider.
type Request struct {
	Prompt    string
	Model     string // provider-specific model id or alias; "" = provider default
	SessionID string // continue this provider session; "" = new
	Dir       string
	Mode      Mode
	Effort    Effort
	// Without lists built-in tools to leave out of this turn, e.g. the web
	// tools when the user switched web access off.
	Without []string
	// Code runs a repository task with an implementation and verification workflow.
	Code bool
	// Research runs planned rounds of searching and reading, ending in a sourced report.
	Research bool
}

// Usage is token accounting for a turn (or a running total).
type Usage struct {
	InputTokens  int
	OutputTokens int
	CacheRead    int
	CacheWrite   int
}

// ContextTokens is how much of the context window the last request used.
func (u Usage) ContextTokens() int { return u.InputTokens + u.CacheRead + u.CacheWrite }

type EventKind int

const (
	EvSession    EventKind = iota // SessionID, Model known
	EvText                        // Text delta of the visible answer
	EvThinking                    // Text delta of model reasoning
	EvToolStart                   // Tool is being called
	EvToolResult                  // Tool finished: Tool.ID, Result, IsError
	EvContext                     // Usage.ContextTokens() = current context size
	EvTodos                       // Todos: the agent's task list changed
	EvToolInput                   // Text delta of tool-call arguments being written (not shown, counted)
	EvArtifact                    // Artifact: the model created or updated an artifact
	EvDone                        // Turn finished: Usage, SessionID, IsError+Text on failure
	EvError                       // Transport/process error: Err
)

// Tool describes a tool call made by the agent.
type Tool struct {
	ID     string
	Name   string
	Input  map[string]any
	Parent string // set for steps a sub-agent takes inside a parent tool call
}

type Event struct {
	Kind          EventKind
	Text          string
	Tool          Tool
	Result        string
	IsError       bool
	Usage         Usage
	SessionID     string
	Model         string
	ContextWindow int       // model's context size when the provider reports it
	Todos         []Todo    // EvTodos only
	Artifact      *Artifact // EvArtifact only
	Approval      *Approval // EvApproval only
	Err           error
}

// EvApproval asks the user to allow a tool call; answer on Approval.Reply.
const EvApproval EventKind = 100

type Approval struct {
	Tool   Tool
	Reason string
	Reply  chan<- bool
	// Rule is set when a user rule asks: the question is asked even in yolo
	// mode.
	Rule bool
	// Scope is what "always allow" covers, e.g. "Bash:git push" or
	// "Edit:/path/file.go" — never the whole tool for commands.
	Scope string
}

// Todo is one item of the agent's task list (the TodoWrite tool).
type Todo struct {
	Content string `json:"content"`
	Status  string `json:"status"` // pending | in_progress | completed
}

// Artifact is a standalone piece of content the model makes for the user to
// view or keep: a document, a web page, a code file, an image as SVG. The
// same ID again means an update of that artifact.
type Artifact struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Kind    string   `json:"kind"` // markdown | html | code | svg | canvas
	Lang    string   `json:"lang,omitempty"`
	Content string   `json:"content"`
	Path    string   `json:"path,omitempty"` // where it was saved
	Version int      `json:"version"`
	Files   []string `json:"files,omitempty"` // exported copies (DOCX, PDF) in the working folder
}

// ModelInfo describes a model the user can pick right now.
type ModelInfo struct {
	ID      string
	Name    string // display name; "" = ID
	Context int    // tokens; 0 = unknown
	Note    string // short extra info shown in the picker
}

func (m ModelInfo) Label() string {
	if m.Name != "" {
		return m.Name
	}
	return m.ID
}

// Provider runs agent turns. NØX-Local has one: the lean loop over
// llama-server.
type Provider interface {
	ID() string      // short id, e.g. "local"
	Name() string    // display name
	Available() bool // llama-server and at least one model are present
	// Models lists the downloaded models.
	Models(ctx context.Context) ([]ModelInfo, error)
	DefaultModel() string // what "" resolves to, for display
	// Run starts a turn. The channel is closed when the turn ends; the last
	// event is EvDone or EvError. Cancelling ctx interrupts the turn.
	Run(ctx context.Context, req Request) (<-chan Event, error)
}

// Identity is appended to every provider's system prompt so the assistant
// always presents itself as NØX.
const Identity = `You are NØX, a terminal coding agent. Always refer to yourself as NØX.
Do not introduce yourself as Claude, ChatGPT, Codex, Gemini or any other product name.
If the user explicitly asks which model powers you, answer truthfully,
for example: "I'm NØX, running <model> locally on this machine."
The user writes in English or Czech; reply in the language of their message.`

// PickModel checks the model a provider would use (want, or its default
// when want is empty) against the models it can really run. It returns the
// model to use and whether it had to be replaced.
func PickModel(p Provider, models []ModelInfo, want string) (string, bool) {
	eff := want
	if eff == "" {
		eff = p.DefaultModel()
	}
	if eff == "" || eff == "default" || len(models) < 2 {
		return want, false
	}
	for _, m := range models {
		if m.ID == eff {
			return want, false
		}
	}
	return models[0].ID, true
}
