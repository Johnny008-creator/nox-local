package tui

import (
	"strings"
	"testing"

	"github.com/Johnny008-creator/nox-local/internal/agent"
)

// A side answer is shown, but it must not become the turn's answer: lastText
// feeds the end-of-turn report and the Discord mirror, and a btw reply there
// would replace the real one.
func TestBtwIsNotTheTurnsAnswer(t *testing.T) {
	m := &Model{}
	m.entries = []entry{
		{kind: kUser, text: "refactor the parser"},
		{kind: kText, text: "The parser is refactored."},
		{kind: kBtw, text: "Because the old code split on commas."},
	}
	if got := m.lastText(); got != "The parser is refactored." {
		t.Errorf("lastText = %q, want the turn's answer", got)
	}
	if len(m.artifacts()) != 0 {
		t.Error("a btw answer is not an artifact source")
	}
}

func TestBtwTranscriptTakesTheTail(t *testing.T) {
	m := &Model{}
	for i := 0; i < 30; i++ {
		m.entries = append(m.entries, entry{kind: kUser, text: "question"})
		m.entries = append(m.entries, entry{kind: kText, text: "answer"})
	}
	tr := m.btwTranscript()
	if tr == "" {
		t.Fatal("no transcript")
	}
	if len(tr) > 8000 {
		t.Errorf("transcript is %d bytes; it is sent with every side question", len(tr))
	}
	if n := strings.Count(tr, "USER: "); n > btwTurns+1 {
		t.Errorf("transcript has %d user turns, want at most %d", n, btwTurns+1)
	}
}

// A side question must see the work, not just the words: without the tool
// rows it cannot answer "what are you doing right now?".
func TestBtwTranscriptCarriesToolRows(t *testing.T) {
	m := &Model{}
	m.entries = []entry{
		{kind: kUser, text: "read big.txt"},
		{kind: kTool, tool: agent.Tool{ID: "1", Name: "Read", Input: map[string]any{"file_path": "big.txt"}}, result: "201 lines", done: true},
		{kind: kTool, tool: agent.Tool{ID: "2", Name: "Bash", Input: map[string]any{"command": "go test ./..."}}, result: "FAIL", done: true, isErr: true},
	}
	tr := m.btwTranscript()
	for _, want := range []string{"USER: read big.txt", "TOOL Read big.txt", "201 lines", "TOOL Run go test ./...", "failed"} {
		if !strings.Contains(tr, want) {
			t.Errorf("transcript lacks %q:\n%s", want, tr)
		}
	}
}

func TestBtwTranscriptEmptyEarly(t *testing.T) {
	if got := (&Model{}).btwTranscript(); got != "" {
		t.Errorf("an empty conversation has no transcript, got %q", got)
	}
}

// A side question must never edit files: the main turn may be mid-change.
// Plan mode is the guarantee here — the local request has no separate
// plain-chat switch.
func TestBtwRunsReadOnly(t *testing.T) {
	req := btwRequest("why?", "qwen3.5-9b", "/tmp/x")
	if req.Mode != agent.ModePlan {
		t.Errorf("btw mode = %v, want plan (read-only)", req.Mode)
	}
	if req.Effort != agent.EffortLow {
		t.Errorf("btw effort = %v, want low", req.Effort)
	}
}
