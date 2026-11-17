// Package native is NØX-Local's agent loop over llama-server's
// OpenAI-compatible chat API. It owns the tools, the permission policy and
// the conversation history.
package native

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Johnny008-creator/N0X-Local/internal/agent"
	"github.com/Johnny008-creator/N0X-Local/internal/mcpclient"
	"github.com/Johnny008-creator/N0X-Local/internal/paths"
)

// Provider is the local agent loop. backend serves the models; mcp holds the
// MCP servers enabled with /mcp.
type Provider struct {
	base    string
	backend Backend
	mcp     *mcpclient.Manager
}

// SetMCP gives the loop access to the tools of enabled MCP servers.
func (p *Provider) SetMCP(m *mcpclient.Manager) { p.mcp = m }

// tools is the lean built-in set (minus the ones switched off for this turn)
// plus the tools of enabled MCP servers.
func (p *Provider) tools(without []string) []map[string]any {
	var out []map[string]any
	for _, t := range leanTools {
		name, _ := t["function"].(map[string]any)["name"].(string)
		if !slices.Contains(without, name) {
			out = append(out, t)
		}
	}
	for _, t := range p.mcp.Tools() {
		schema := t.Schema
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		desc := strings.Join(strings.Fields(t.Desc), " ")
		if len(desc) > 300 { // small models: keep the menu short
			desc = desc[:300]
		}
		out = append(out, map[string]any{"type": "function", "function": map[string]any{
			"name": t.FuncName(), "description": "[" + t.Server + "] " + desc, "parameters": schema,
		}})
	}
	return out
}

func (p *Provider) ID() string      { return "local" }
func (p *Provider) Name() string    { return "Local" }
func (p *Provider) Available() bool { return p.backend.Ready() }

func (p *Provider) DefaultModel() string {
	if ms := p.backend.Installed(); len(ms) > 0 {
		return ms[0].ID
	}
	return ""
}

// Models lists the downloaded models.
func (p *Provider) Models(context.Context) ([]agent.ModelInfo, error) {
	return p.backend.Installed(), nil
}

// ---------- conversation history ----------

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type message struct {
	Role       string     `json:"role"`
	Content    *string    `json:"content"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

func text(s string) *string { return &s }

func historyPath(id string) string { return paths.File("native", id+".json") }

func loadHistory(id string) []message {
	b, err := os.ReadFile(historyPath(id))
	if err != nil {
		return nil
	}
	var h []message
	_ = json.Unmarshal(b, &h)
	return h
}

func saveHistory(id string, h []message) {
	if b, err := json.Marshal(h); err == nil {
		_ = os.WriteFile(historyPath(id), b, 0o644)
	}
}

func newSessionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "nx_" + hex.EncodeToString(b)
}

// ---------- tool calls ----------

func (p *Provider) callTool(ctx context.Context, r runner, mode agent.Mode, tc toolCall, ch chan<- agent.Event) string {
	in, err := decodeArgs(tc.Function.Arguments)
	tool := agent.Tool{ID: tc.ID, Name: tc.Function.Name, Input: in}
	ch <- agent.Event{Kind: agent.EvToolStart, Tool: tool}
	result := func(s string, isErr bool) string {
		ch <- agent.Event{Kind: agent.EvToolResult, Tool: agent.Tool{ID: tc.ID}, Result: s, IsError: isErr}
		if isErr {
			return "ERROR: " + s
		}
		return s
	}
	if err != nil {
		return result("invalid JSON arguments: "+err.Error(), true)
	}
	mt, isMCP := p.mcp.Lookup(tool.Name)
	var ask bool
	var reason, deny string
	if isMCP {
		ask, reason, deny = mcpPolicy(mode, mt)
	} else {
		ask, reason, deny = policy(mode, tool.Name, in)
	}
	if deny != "" {
		return result(deny, true)
	}
	if ask {
		reply := make(chan bool, 1)
		ch <- agent.Event{Kind: agent.EvApproval, Approval: &agent.Approval{Tool: tool, Reason: reason, Reply: reply}}
		select {
		case ok := <-reply:
			if !ok {
				return result("the user denied this tool call", true)
			}
		case <-ctx.Done():
			return result("interrupted", true)
		}
	}
	if isMCP {
		out, isErr, err := p.mcp.Call(ctx, mt, in)
		if err != nil {
			return result(err.Error(), true)
		}
		return result(clip(out), isErr)
	}
	if slices.Contains(r.without, tool.Name) {
		return result(tool.Name+" is switched off for this conversation", true)
	}
	if tool.Name == "Artifact" {
		a, err := saveArtifact(r.art, in)
		if err != nil {
			return result(err.Error(), true)
		}
		msg := fmt.Sprintf("artifact %q saved (version %d, %d chars); the user sees it next to the chat", a.ID, a.Version, len(a.Content))
		if formats := exportFormats(in); len(formats) > 0 {
			files, err := exportArtifact(a, r.dir, formats)
			a.Files = files
			if err != nil {
				msg += "; export failed: " + err.Error()
			} else {
				msg += "; exported " + strings.Join(files, ", ")
			}
		}
		ch <- agent.Event{Kind: agent.EvArtifact, Artifact: a}
		return result(msg, false)
	}
	if tool.Name == "TodoWrite" {
		todos, out, err := parseTodos(in)
		if err != nil {
			return result(err.Error(), true)
		}
		ch <- agent.Event{Kind: agent.EvTodos, Todos: todos}
		return result(out, false)
	}
	out, err := r.run(ctx, tool.Name, in)
	if err != nil {
		if out != "" {
			return result(out+"\n"+err.Error(), true)
		}
		return result(err.Error(), true)
	}
	return result(out, false)
}

// trim keeps the history inside the context window by blanking the oldest
// tool outputs first (they are re-readable), then dropping old turns.
func trim(h []message, window int) []message {
	if window <= 0 {
		window = 32_000
	}
	budget := window * 3 // ~chars per token, leaving room for the answer
	size := func() int {
		n := 0
		for _, m := range h {
			if m.Content != nil {
				n += len(*m.Content)
			}
			for _, t := range m.ToolCalls {
				n += len(t.Function.Arguments)
			}
		}
		return n
	}
	for i := 1; i < len(h)-6 && size() > budget; i++ {
		if h[i].Role == "tool" && h[i].Content != nil && len(*h[i].Content) > 200 {
			h[i].Content = text("[output trimmed to save context; run the tool again if needed]")
		}
	}
	for size() > budget && len(h) > 8 {
		// drop the oldest non-system message, keeping tool results paired with their call
		cut := 2
		for cut < len(h)-6 && h[cut].Role == "tool" {
			cut++
		}
		h = append(h[:1], h[cut:]...)
	}
	return h
}

// ---------- streaming completion ----------

type chunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			Reasoning        string `json:"reasoning"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		Prompt     int `json:"prompt_tokens"`
		Completion int `json:"completion_tokens"`
		Details    *struct {
			Cached int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *Provider) complete(ctx context.Context, model string, effort agent.Effort, without []string, hist []message, ch chan<- agent.Event) (message, agent.Usage, error) {
	th := think(effort)
	body := map[string]any{
		"model":                model,
		"messages":             hist,
		"tools":                p.tools(without),
		"stream":               true,
		"stream_options":       map[string]any{"include_usage": true},
		"max_tokens":           replyBudget(effort),
		"chat_template_kwargs": map[string]any{"enable_thinking": th},
	}
	if th {
		body["thinking_budget_tokens"] = thinkBudget(effort)
		body["reasoning_budget_message"] = wrapUp
	}
	for k, v := range p.backend.Sampling(model, effort) {
		body[k] = v
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, p.base+"/chat/completions", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 0}).Do(req)
	if err != nil {
		return message{}, agent.Usage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4000))
		return message{}, agent.Usage{}, serverError(resp.StatusCode, raw)
	}

	var (
		content strings.Builder
		calls   []toolCall
		usage   agent.Usage
	)
	rd := bufio.NewReaderSize(resp.Body, 1<<16)
	for {
		line, err := rd.ReadString('\n')
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				break
			}
			var c chunk
			if json.Unmarshal([]byte(data), &c) != nil {
				continue
			}
			if c.Error != nil {
				return message{}, usage, errors.New(c.Error.Message)
			}
			if u := c.Usage; u != nil {
				usage.InputTokens, usage.OutputTokens = u.Prompt, u.Completion
				if u.Details != nil {
					usage.CacheRead = u.Details.Cached
					usage.InputTokens -= u.Details.Cached
				}
			}
			for _, ch0 := range c.Choices {
				d := ch0.Delta
				if r := d.Reasoning + d.ReasoningContent; r != "" {
					ch <- agent.Event{Kind: agent.EvThinking, Text: r}
				}
				if d.Content != "" {
					content.WriteString(d.Content)
					ch <- agent.Event{Kind: agent.EvText, Text: d.Content}
				}
				for _, t := range d.ToolCalls {
					for len(calls) <= t.Index {
						calls = append(calls, toolCall{Type: "function"})
					}
					c := &calls[t.Index]
					if t.ID != "" {
						c.ID = t.ID
					}
					c.Function.Name += t.Function.Name
					c.Function.Arguments += t.Function.Arguments
					if d := t.Function.Name + t.Function.Arguments; d != "" {
						ch <- agent.Event{Kind: agent.EvToolInput, Text: d}
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return message{}, usage, err
		}
	}
	msg := message{Role: "assistant"}
	if content.Len() > 0 || len(calls) == 0 {
		msg.Content = text(content.String())
	}
	for i := range calls {
		if calls[i].ID == "" {
			calls[i].ID = "call_" + strconv.Itoa(i) + "_" + newSessionID()[3:9]
		}
		if calls[i].Function.Name != "" {
			msg.ToolCalls = append(msg.ToolCalls, calls[i])
		}
	}
	return msg, usage, nil
}

// serverError turns a llama-server error reply into a readable message.
func serverError(status int, raw []byte) error {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &e)
	msg := e.Error.Message
	if msg == "" {
		msg = strings.TrimSpace(string(raw))
	}
	hint := ""
	if strings.Contains(msg, "context") || strings.Contains(msg, "exceed") {
		hint = " · the conversation no longer fits the model's window; run /compact or /clear"
	}
	return fmt.Errorf("llama-server HTTP %d: %s%s", status, msg, hint)
}
