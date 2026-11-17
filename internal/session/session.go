// Package session persists NØX conversations under ~/.nox/sessions so they
// can be resumed (/resume, nox -c) and exported.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Johnny008-creator/N0X-Local/internal/paths"
)

// Entry kinds mirror the TUI's (user, text, thinking, tool, info, error).
const (
	KindUser = iota
	KindText
	KindThinking
	KindTool
	KindInfo
	KindError
)

type Entry struct {
	Kind       int            `json:"kind"`
	Text       string         `json:"text,omitempty"`
	Provider   string         `json:"provider,omitempty"`
	ToolID     string         `json:"tool_id,omitempty"`
	ToolName   string         `json:"tool_name,omitempty"`
	ToolInput  map[string]any `json:"tool_input,omitempty"`
	Result     string         `json:"result,omitempty"`
	IsError    bool           `json:"is_error,omitempty"`
	DurationMS int64          `json:"duration_ms,omitempty"`
}

type Session struct {
	ID               string            `json:"id"`
	Dir              string            `json:"dir"`
	Created          time.Time         `json:"created"`
	Updated          time.Time         `json:"updated"`
	Provider         string            `json:"provider"`
	Model            string            `json:"model,omitempty"`
	Mode             string            `json:"mode"`
	Effort           string            `json:"effort"`
	ProviderSessions map[string]string `json:"provider_sessions,omitempty"`
	Entries          []Entry           `json:"entries"`
	Turns            int               `json:"turns"`
	CostUSD          float64           `json:"cost_usd"`
	OutputTokens     int               `json:"output_tokens"`
}

// Title is the first prompt, shortened.
func (s *Session) Title() string {
	for _, e := range s.Entries {
		if e.Kind == KindUser {
			t := strings.Join(strings.Fields(e.Text), " ")
			if r := []rune(t); len(r) > 60 {
				t = string(r[:59]) + "…"
			}
			return t
		}
	}
	return s.ID
}

type Store struct{ dir string }

func Open() *Store { return &Store{dir: paths.Sub("sessions")} }

func newID() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(b)
}

func (st *Store) New(dir string) *Session {
	now := time.Now()
	return &Session{ID: newID(), Dir: dir, Created: now, Updated: now}
}

const maxResult = 4000

func (st *Store) Save(s *Session) error {
	s.Updated = time.Now()
	cp := *s
	cp.Entries = make([]Entry, len(s.Entries)) // shorten a copy, never the live entries
	for i, e := range s.Entries {
		if len(e.Result) > maxResult {
			cut := maxResult
			for cut > 0 && !utf8.RuneStart(e.Result[cut]) {
				cut--
			}
			e.Result = e.Result[:cut] + "\n…"
		}
		cp.Entries[i] = e
	}
	b, err := json.MarshalIndent(&cp, "", " ")
	if err != nil {
		return err
	}
	p := filepath.Join(st.dir, s.ID+".json")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (st *Store) Load(id string) (*Session, error) {
	if strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") || id == "" {
		return nil, fmt.Errorf("invalid session id %q", id)
	}
	b, err := os.ReadFile(filepath.Join(st.dir, id+".json"))
	if err != nil {
		return nil, err
	}
	var s Session
	return &s, json.Unmarshal(b, &s)
}

// List returns sessions newest first; dir filters to one project ("" = all).
func (st *Store) List(dir string) ([]*Session, error) {
	files, err := filepath.Glob(filepath.Join(st.dir, "*.json"))
	if err != nil {
		return nil, err
	}
	var out []*Session
	for _, f := range files {
		s, err := st.Load(strings.TrimSuffix(filepath.Base(f), ".json"))
		if err != nil || (dir != "" && !strings.EqualFold(s.Dir, dir)) {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Updated.After(out[j].Updated) })
	return out, nil
}

// Latest is the most recently updated session in dir.
func (st *Store) Latest(dir string) (*Session, error) {
	l, err := st.List(dir)
	if err != nil {
		return nil, err
	}
	if len(l) == 0 {
		return nil, fmt.Errorf("no saved sessions for %s", dir)
	}
	return l[0], nil
}

// Markdown renders the conversation for /export.
func (s *Session) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# NØX session %s\n\n- dir: `%s`\n- started: %s\n- provider: %s %s\n\n", s.ID, s.Dir, s.Created.Format(time.RFC1123), s.Provider, s.Model)
	for _, e := range s.Entries {
		switch e.Kind {
		case KindUser:
			fmt.Fprintf(&b, "## ❯ You\n\n%s\n\n", e.Text)
		case KindText:
			fmt.Fprintf(&b, "**NØX** _(%s)_\n\n%s\n\n", e.Provider, e.Text)
		case KindTool:
			status := "ok"
			if e.IsError {
				status = "error"
			}
			target := ""
			for _, k := range []string{"file_path", "command", "pattern", "url", "query"} {
				if v, ok := e.ToolInput[k].(string); ok {
					target = v
					break
				}
			}
			fmt.Fprintf(&b, "- ◆ `%s` %s _(%s)_\n", e.ToolName, target, status)
		case KindError:
			fmt.Fprintf(&b, "> ✗ %s\n\n", e.Text)
		}
	}
	return b.String()
}
