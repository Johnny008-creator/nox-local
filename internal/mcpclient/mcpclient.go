// Package mcpclient connects the agent loop to MCP servers. Servers come
// from the project's .mcp.json (Claude Code format) and from [mcp.<name>]
// tables in the config; each one is connected only once enabled with /mcp.
package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Johnny008-creator/N0X-Local/internal/proc"
)

type ServerConfig struct {
	Command string            `toml:"command" json:"command"`
	Args    []string          `toml:"args" json:"args"`
	Env     map[string]string `toml:"env" json:"env"`
	URL     string            `toml:"url" json:"url"`
}

// LoadProject reads <dir>/.mcp.json and merges extra (config.toml) on top.
func LoadProject(dir string, extra map[string]ServerConfig) map[string]ServerConfig {
	out := map[string]ServerConfig{}
	if b, err := os.ReadFile(filepath.Join(dir, ".mcp.json")); err == nil {
		var f struct {
			Servers map[string]ServerConfig `json:"mcpServers"`
		}
		if json.Unmarshal(b, &f) == nil {
			for k, v := range f.Servers {
				out[k] = v
			}
		}
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// Tool is one MCP tool exposed to the model as mcp__<server>__<tool>.
type Tool struct {
	Server, Name, Desc string
	Schema             any
	ReadOnly           bool
	Destructive        bool
}

var unsafe = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// FuncName is the function name the model sees (max 64 chars, safe chars).
func (t Tool) FuncName() string {
	n := "mcp__" + unsafe.ReplaceAllString(t.Server, "_") + "__" + unsafe.ReplaceAllString(t.Name, "_")
	if len(n) > 64 {
		n = n[:64]
	}
	return n
}

// Server is one configured MCP server and its state in this session.
type Server struct {
	Name      string
	Enabled   bool // picked with /mcp; only enabled servers are connected
	Connected bool
	Tools     int
	Err       error
}

// Manager connects only the servers you enable. Small local models cope with
// a handful of tools, not with every tool of every configured server, so
// nothing is connected until it is switched on with /mcp. The choice is
// remembered in statePath.
type Manager struct {
	mu        sync.RWMutex
	cfgs      map[string]ServerConfig
	enabled   map[string]bool
	sessions  map[string]*mcp.ClientSession
	tools     map[string][]Tool
	errs      map[string]error
	statePath string
	startup   sync.WaitGroup // connections of servers enabled last time
}

// New prepares the configured servers and connects the ones that were
// enabled last time, in the background.
func New(cfgs map[string]ServerConfig, statePath string) *Manager {
	m := &Manager{cfgs: cfgs, enabled: map[string]bool{}, sessions: map[string]*mcp.ClientSession{},
		tools: map[string][]Tool{}, errs: map[string]error{}, statePath: statePath}
	if b, err := os.ReadFile(statePath); err == nil {
		var names []string
		_ = json.Unmarshal(b, &names)
		for _, n := range names {
			if _, ok := cfgs[n]; ok {
				m.enabled[n] = true
			}
		}
	}
	for n := range m.enabled {
		m.startup.Add(1)
		go func() {
			defer m.startup.Done()
			_ = m.connect(n)
		}()
	}
	return m
}

// WaitStartup blocks until the servers enabled last time have connected (or
// failed), or ctx ends, so the first request already offers their tools.
func (m *Manager) WaitStartup(ctx context.Context) {
	if m == nil {
		return
	}
	done := make(chan struct{})
	go func() { m.startup.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// Servers lists every configured server, sorted by name.
func (m *Manager) Servers() []Server {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Server
	for n := range m.cfgs {
		_, conn := m.sessions[n]
		out = append(out, Server{Name: n, Enabled: m.enabled[n], Connected: conn, Tools: len(m.tools[n]), Err: m.errs[n]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// SetEnabled switches a server on (connecting it, which may take a while) or
// off (closing it), and remembers the choice.
func (m *Manager) SetEnabled(name string, on bool) error {
	m.mu.Lock()
	if _, ok := m.cfgs[name]; !ok {
		m.mu.Unlock()
		return fmt.Errorf("no MCP server %q", name)
	}
	m.enabled[name] = on
	m.saveLocked()
	sess := m.sessions[name]
	if !on {
		delete(m.sessions, name)
		delete(m.tools, name)
		delete(m.errs, name)
	}
	m.mu.Unlock()
	if !on {
		if sess != nil {
			_ = sess.Close()
		}
		return nil
	}
	if sess != nil {
		return nil
	}
	return m.connect(name)
}

// Add registers a server at runtime (off until enabled); an existing one
// with the same name is replaced.
func (m *Manager) Add(name string, c ServerConfig) {
	m.mu.Lock()
	m.cfgs[name] = c
	m.mu.Unlock()
}

// Remove switches a server off and forgets it.
func (m *Manager) Remove(name string) {
	_ = m.SetEnabled(name, false)
	m.mu.Lock()
	delete(m.cfgs, name)
	delete(m.enabled, name)
	m.saveLocked()
	m.mu.Unlock()
}

func (m *Manager) saveLocked() {
	var names []string
	for n, on := range m.enabled {
		if on {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	if b, err := json.Marshal(names); err == nil {
		_ = os.WriteFile(m.statePath, b, 0o644)
	}
}

func (m *Manager) connect(name string) error {
	m.mu.RLock()
	c := m.cfgs[name]
	m.mu.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fail := func(err error) error {
		m.mu.Lock()
		m.errs[name] = err
		m.mu.Unlock()
		return err
	}
	var t mcp.Transport
	switch {
	case c.URL != "":
		t = &mcp.StreamableClientTransport{Endpoint: c.URL}
	case c.Command != "":
		prog, lead, err := proc.Resolve(c.Command)
		if err != nil {
			return fail(err)
		}
		cmd := exec.Command(prog, append(lead, c.Args...)...)
		cmd.Env = os.Environ()
		for k, v := range c.Env {
			cmd.Env = append(cmd.Env, k+"="+os.ExpandEnv(v))
		}
		t = &mcp.CommandTransport{Command: cmd}
	default:
		return fail(fmt.Errorf("needs command or url"))
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "nox-local", Version: "0.2"}, nil)
	sess, err := client.Connect(ctx, t, nil)
	if err != nil {
		return fail(err)
	}
	res, err := sess.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		_ = sess.Close()
		return fail(err)
	}
	var tools []Tool
	for _, tl := range res.Tools {
		t := Tool{Server: name, Name: tl.Name, Desc: tl.Description, Schema: tl.InputSchema}
		if a := tl.Annotations; a != nil {
			t.ReadOnly = a.ReadOnlyHint
			t.Destructive = a.DestructiveHint != nil && *a.DestructiveHint
		}
		tools = append(tools, t)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.enabled[name] { // switched off while connecting
		_ = sess.Close()
		return nil
	}
	m.sessions[name] = sess
	m.tools[name] = tools
	delete(m.errs, name)
	return nil
}

// Tools lists the tools of the enabled, connected servers.
func (m *Manager) Tools() []Tool {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Tool
	for _, ts := range m.tools {
		out = append(out, ts...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FuncName() < out[j].FuncName() })
	return out
}

// Lookup finds a tool by its function name.
func (m *Manager) Lookup(fn string) (Tool, bool) {
	if m == nil || !strings.HasPrefix(fn, "mcp__") {
		return Tool{}, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ts := range m.tools {
		for _, t := range ts {
			if t.FuncName() == fn {
				return t, true
			}
		}
	}
	return Tool{}, false
}

// Call runs a tool and flattens its text content.
func (m *Manager) Call(ctx context.Context, t Tool, args map[string]any) (string, bool, error) {
	m.mu.RLock()
	sess := m.sessions[t.Server]
	m.mu.RUnlock()
	if sess == nil {
		return "", true, fmt.Errorf("MCP server %s is not connected", t.Server)
	}
	res, err := sess.CallTool(ctx, &mcp.CallToolParams{Name: t.Name, Arguments: args})
	if err != nil {
		return "", true, err
	}
	var b strings.Builder
	for _, c := range res.Content {
		switch v := c.(type) {
		case *mcp.TextContent:
			b.WriteString(v.Text)
			b.WriteByte('\n')
		default:
			raw, _ := json.Marshal(v)
			b.Write(raw)
			b.WriteByte('\n')
		}
	}
	if b.Len() == 0 && res.StructuredContent != nil {
		raw, _ := json.Marshal(res.StructuredContent)
		b.Write(raw)
	}
	return strings.TrimSpace(b.String()), res.IsError, nil
}

func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		_ = s.Close()
	}
}
