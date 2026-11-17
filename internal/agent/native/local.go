package native

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/Johnny008-creator/nox-local/internal/agent"
	"github.com/Johnny008-creator/nox-local/internal/paths"
)

// Backend serves models on this machine (see internal/local). The loop is
// lean: a short prompt, few tools and tight output limits, so 4B–35B models
// keep their small context for the work.
type Backend interface {
	Ready() bool
	Installed() []agent.ModelInfo
	Ensure(ctx context.Context, model string, status func(string)) (window int, err error)
	// Sampling returns request fields for model at effort (they override the
	// defaults, e.g. a thinking-only model forces enable_thinking).
	Sampling(model string, effort agent.Effort) map[string]any
}

// Local is the provider for llama-server at base.
func Local(b Backend, base string) *Provider {
	return &Provider{base: base, backend: b}
}

// Lean-mode limits.
const (
	leanMaxSteps  = 30
	leanMaxOut    = 8_000 // chars of tool output the model sees
	leanReadLines = 300
	leanAnswer    = 4096  // tokens reserved for the reply (effort low: no thinking)
	leanThinking  = 8192  // … effort medium
	leanDeep      = 16384 // … effort high
	leanRepeat    = 3     // identical tool calls in a row before we intervene
	// Thinking budgets. max_tokens caps thinking and answer together, so a
	// model could think for 10k+ tokens (Qwen3.5-4B did: 16 minutes for one
	// reply). llama-server closes the think block once the budget is spent.
	thinkMedium = 2048
	thinkHigh   = 6144
)

// wrapUp is injected before the end-of-thinking tag when the budget runs out.
const wrapUp = "\n\nTime is up. I will stop thinking and act now.\n"

// thinkBudget is the per-request thinking_budget_tokens (-1: unlimited).
func thinkBudget(e agent.Effort) int {
	switch e {
	case agent.EffortLow:
		return 0
	case agent.EffortMedium:
		return thinkMedium
	}
	return thinkHigh
}

// leanTools are the core tools with short descriptions; small models follow
// a short, unambiguous menu much better than a long one.
var leanTools = []map[string]any{
	fn("Read", "Read a file (300 lines at a time; use offset for more).",
		props{"file_path": str("path"), "offset": num("first line, 1-based")}, "file_path"),
	fn("Write", "Create a new file or replace a whole file.",
		props{"file_path": str("path"), "content": str("full content")}, "file_path", "content"),
	fn("Edit", "Replace exact text in a file. old_string must appear exactly once.",
		props{"file_path": str("path"), "old_string": str("exact text to find"), "new_string": str("replacement")}, "file_path", "old_string", "new_string"),
	fn("Glob", "List files matching a pattern like **/*.py.",
		props{"pattern": str("glob pattern")}, "pattern"),
	fn("Grep", "Search file contents for a regex. Returns path:line: text.",
		props{"pattern": str("regex"), "glob": str("only files like *.go")}, "pattern"),
	fn("Bash", "Run a shell command (bash) in the project directory. For servers, dev servers and watchers set background: true: it starts, returns after 5 s with the URL and log, and keeps running.",
		props{"command": str("command"), "background": map[string]any{"type": "boolean", "description": "true for anything that keeps running (servers, watchers)"}}, "command"),
	fn("WebSearch", "Search the web. Returns titles, URLs and snippets.",
		props{"query": str("search query")}, "query"),
	fn("WebFetch", "Fetch a web page and return its text.",
		props{"url": str("http(s) URL")}, "url"),
	fn("Artifact", "Show the user a document, web page, code file, SVG or a live canvas. Use it for anything they will read, keep or reuse, not for short answers. The same id again updates it. kind=canvas draws live in the chat: content is JavaScript that defines draw(ctx, w, h, t) (t = seconds, called every frame, the canvas is cleared before each call) for animations, or draws once with ctx, W, H. Charts: nox.bar(ctx,{labels,values,title,t}), nox.line(ctx,{labels,series:[{name,values}],title,t}), nox.pie(ctx,{labels,values,title,t,donut:true}). Also nox.text(ctx,s,x,y,{size,color,align,bold}), nox.colors.palette, nox.hsl(h), nox.rand(seed), nox.ease(p). No libraries or network.",
		props{"id": str("short id, e.g. pricing-page"), "title": str("title shown to the user"),
			"kind":     map[string]any{"type": "string", "enum": []string{"markdown", "html", "code", "svg", "canvas"}},
			"language": str("for kind=code: go, python, js…"), "content": str("the full content"),
			"export": map[string]any{"type": "array", "description": "for kind=markdown: also save as docx and/or pdf in the working folder", "items": map[string]any{"type": "string", "enum": []string{"docx", "pdf"}}}},
		"title", "kind", "content"),
	fn("TodoWrite", "Keep a task list for multi-step work. Send the whole list every time; mark one item in_progress, finished ones completed.",
		props{"todos": map[string]any{"type": "array", "description": "the full list", "items": map[string]any{
			"type": "object", "required": []string{"content", "status"},
			"properties": props{"content": str("the task"), "status": map[string]any{"type": "string", "enum": []string{"pending", "in_progress", "completed"}}},
		}}}, "todos"),
}

func leanGuide(dir string, mode agent.Mode) string {
	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "PowerShell"
	}
	g := fmt.Sprintf(`

# Environment
Directory: %s · OS: %s · shell: %s · today: %s

# Rules
1. Look before you change: Read a file before you Edit it.
2. One step at a time. Call a tool, read the result, then decide.
3. Edit for small changes, Write for new files.
4. After changing code, run the build or tests with Bash if the project has them.
5. If a tool fails, read the error and fix the cause. Do not repeat the same call.
6. For work with 3+ steps, keep a list with TodoWrite and update it as you go.
7. Use WebSearch / WebFetch for docs or facts you are unsure about.
8. Put documents, pages and files the user will keep into an Artifact, then answer briefly. For an animation, chart, diagram or "show me" visual, use an Artifact with kind canvas.
9. To run a server or anything that keeps running, call Bash with background: true. Then tell the user what to open (the URL) and how to try it.
10. When the task is done, stop calling tools and answer briefly: what you made and how to try it.`, dir, runtime.GOOS, shell, time.Now().Format("2006-01-02"))
	switch mode {
	case agent.ModePlan:
		g += "\nPLAN MODE: only read and search, then give a numbered plan."
	case agent.ModeManual:
		g += "\nThe user approves each change. If a call is denied, ask what they want."
	}
	return g
}

// codeGuide adds a repository workflow to the same native coding tools.
const codeGuide = `

# CODE WORKSPACE
You are working in a dedicated repository checkout on a task branch.
Inspect git status, the repository instructions (AGENTS.md/CONTRIBUTING.md) and relevant files first.
Implement the requested change using Read, Edit, Write and Bash; run the relevant tests/build.
Keep unrelated changes intact. Use TodoWrite for substantial tasks.
The desktop shows your tool output, working-tree diff and commits beside the conversation.
When the user requests a commit or push, perform it with git on the current task branch (git push -u origin HEAD for its first push).
Otherwise leave changes for review with the desktop's Commit selected and Push branch buttons.
Do not claim a commit, push, test or pull request succeeded unless the command succeeded.
Finish with the change, verification results and any remaining limitations.
`

// researchMaxSteps gives deep research room for several rounds.
const researchMaxSteps = 60

// researchGuide turns a turn into deep research. Small models follow an
// explicit procedure much better than "research this thoroughly".
const researchGuide = `

# DEEP RESEARCH MODE
Work in rounds until the question is answered well:
1. Plan: split the question into 3-6 sub-questions and put them in TodoWrite.
2. For each sub-question: WebSearch with 2-3 different queries, then WebFetch the 2-4 best sources. Prefer official sites, documentation and papers over blogs.
3. After each round ask yourself what is still unknown or contradictory, and search again for that. Mark finished sub-questions in TodoWrite.
4. Note dates. When sources disagree, say so; never invent numbers.
5. Finish with ONE markdown Artifact (kind markdown): a short summary first, then a section per sub-question, then a numbered "Sources" list with URLs. Cite inline as [1], [2].
6. If the user asked for DOCX or PDF, set export on that Artifact.
Then answer in 3-5 sentences with the main findings.`

// researchGate holds deep research to a minimum: the report waits until a
// few sources were read and must list them. Each check fires once, so a
// model that can't comply still finishes.
type researchGate struct {
	on                  bool
	reads               int
	askedRead, askedSrc bool
}

const minSources = 3

var sourcesHead = regexp.MustCompile(`(?im)^#{1,4}\s*(\d+[.)]?\s*)?(sources|references|zdroje)\b`)

func (g *researchGate) seen(tc toolCall, out string) {
	if g.on && tc.Function.Name == "WebFetch" && !strings.HasPrefix(out, "ERROR") {
		g.reads++
	}
}

func (g *researchGate) check(tc toolCall) string {
	if !g.on || tc.Function.Name != "Artifact" {
		return ""
	}
	if g.reads < minSources && !g.askedRead {
		g.askedRead = true
		return fmt.Sprintf("research is not done: you read %d source(s). Read at least %d different sources with WebFetch first (search again if needed), then write the report.", g.reads, minSources)
	}
	in, _ := decodeArgs(tc.Function.Arguments)
	if c, _ := in["content"].(string); !sourcesHead.MatchString(c) && !g.askedSrc {
		g.askedSrc = true
		return `the report needs a "## Sources" section: a numbered list of the URLs you used, cited inline as [1], [2]. Add it and call Artifact again.`
	}
	return ""
}

// answerNow asks a model that ended its turn without visible text for the
// answer it worked out while thinking.
const answerNow = "(Your reply to me was empty. Write your final answer now as plain text, briefly. Do not call tools.)"

func blank(s *string) bool { return s == nil || strings.TrimSpace(*s) == "" }

// think: effort low answers directly; higher efforts let the model reason.
func think(e agent.Effort) bool { return e != agent.EffortLow }

// replyBudget is max_tokens for a request: the answer plus room to think.
func replyBudget(e agent.Effort) int {
	switch e {
	case agent.EffortLow:
		return leanAnswer
	case agent.EffortMedium:
		return leanThinking
	}
	return leanDeep
}

func clipTo(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n*2/3] + fmt.Sprintf("\n… [%d chars cut to save context; narrow the command or use offset] …\n", len(s)-n) + s[len(s)-n/3:]
}

// Small models sometimes print the call instead of emitting it. Two shapes
// are common: Hermes/Qwen JSON and the Qwen3-Coder XML form.
var (
	jsonCall = regexp.MustCompile(`(?s)<tool_call>\s*(\{.*?\})\s*</tool_call>`)
	xmlCall  = regexp.MustCompile(`(?s)<function=([A-Za-z_]+)>(.*?)</function>`)
	xmlParam = regexp.MustCompile(`(?s)<parameter=([A-Za-z_]+)>\n?(.*?)\n?</parameter>`)
)

// textToolCalls recovers tool calls written into the reply text. It returns
// the calls and the text with them removed.
func textToolCalls(s string) ([]toolCall, string) {
	var calls []toolCall
	add := func(name string, args map[string]any) {
		b, _ := json.Marshal(args)
		tc := toolCall{Type: "function", ID: fmt.Sprintf("call_txt%d_%s", len(calls), newSessionID()[3:9])}
		tc.Function.Name, tc.Function.Arguments = name, string(b)
		calls = append(calls, tc)
	}
	rest := jsonCall.ReplaceAllStringFunc(s, func(m string) string {
		var v struct {
			Name string         `json:"name"`
			Args map[string]any `json:"arguments"`
		}
		if json.Unmarshal([]byte(jsonCall.FindStringSubmatch(m)[1]), &v) == nil && v.Name != "" {
			add(v.Name, v.Args)
			return ""
		}
		return m
	})
	rest = xmlCall.ReplaceAllStringFunc(rest, func(m string) string {
		sm := xmlCall.FindStringSubmatch(m)
		args := map[string]any{}
		for _, p := range xmlParam.FindAllStringSubmatch(sm[2], -1) {
			args[p[1]] = p[2]
		}
		add(sm[1], args)
		return ""
	})
	rest = strings.ReplaceAll(rest, "<tool_call>", "")
	rest = strings.ReplaceAll(rest, "</tool_call>", "")
	return calls, strings.TrimSpace(rest)
}

// repeats counts how many times in a row the same call has been made.
type repeats struct {
	last string
	n    int
}

func (r *repeats) see(tc toolCall) int {
	sig := tc.Function.Name + "\x00" + tc.Function.Arguments
	if sig == r.last {
		r.n++
	} else {
		r.last, r.n = sig, 1
	}
	return r.n
}

// Run is one agent turn: lean limits, recovery of tool calls written as
// text, and a stop for loops.
func (p *Provider) Run(ctx context.Context, req agent.Request) (<-chan agent.Event, error) {
	model := req.Model
	if model == "" || model == "default" {
		model = p.DefaultModel()
	}
	if model == "" {
		return nil, errors.New("no local model downloaded · run: nox-local pull qwen3.5-4b")
	}
	sid := req.SessionID
	var hist []message
	if sid != "" {
		hist = loadHistory(sid)
	} else {
		sid = newSessionID()
	}
	sys := agent.SystemPromptLean(req.Dir) + leanGuide(req.Dir, req.Mode)
	steps := leanMaxSteps
	if req.Code {
		sys += codeGuide
		steps = 60
	}
	if req.Research {
		sys += researchGuide
		steps = researchMaxSteps
		req.Without = slices.DeleteFunc(slices.Clone(req.Without), func(t string) bool { return t == "WebSearch" || t == "WebFetch" })
	}
	if len(hist) == 0 || hist[0].Role != "system" {
		hist = append([]message{{Role: "system", Content: text(sys)}}, hist...)
	} else {
		hist[0].Content = text(sys)
	}
	hist = append(hist, message{Role: "user", Content: text(req.Prompt)})

	ch := make(chan agent.Event, 64)
	go func() {
		defer close(ch)
		defer func() { saveHistory(sid, hist) }()
		window, err := p.backend.Ensure(ctx, model, func(s string) {
			ch <- agent.Event{Kind: agent.EvText, Text: "_" + s + "_\n\n"}
		})
		if err != nil {
			if ctx.Err() != nil {
				ch <- agent.Event{Kind: agent.EvError, Err: errors.New("interrupted")}
			} else {
				ch <- agent.Event{Kind: agent.EvDone, SessionID: sid, IsError: true, Text: err.Error()}
			}
			return
		}
		ch <- agent.Event{Kind: agent.EvSession, SessionID: sid, Model: model, ContextWindow: window}
		wctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		p.mcp.WaitStartup(wctx) // tools of servers enabled last time
		cancel()

		var total agent.Usage
		var rep repeats
		nudge := -1 // >= 0 once the model was asked for its answer this turn
		rg := researchGate{on: req.Research}
		r := runner{dir: req.Dir, art: paths.Sub("artifacts", sid), without: req.Without}
		// what the history may use: the window minus the reply and the tool menu
		budget := max(window-replyBudget(req.Effort)-1000, window/3)
		for step := 0; step < steps; step++ {
			hist = trim(hist, budget)
			msg, usage, err := p.complete(ctx, model, req.Effort, req.Without, hist, ch)
			total.OutputTokens += usage.OutputTokens
			total.InputTokens, total.CacheRead = usage.InputTokens, usage.CacheRead
			// after every request: exact token counts, including tool-call
			// arguments (fully cached prompts report 0 new input tokens)
			if usage.InputTokens+usage.CacheRead+usage.OutputTokens > 0 {
				ch <- agent.Event{Kind: agent.EvContext, Usage: usage}
			}
			if err != nil {
				if ctx.Err() != nil {
					ch <- agent.Event{Kind: agent.EvError, Err: errors.New("interrupted")}
				} else {
					ch <- agent.Event{Kind: agent.EvDone, SessionID: sid, Usage: total, IsError: true, Text: err.Error()}
				}
				return
			}
			if len(msg.ToolCalls) == 0 && msg.Content != nil {
				if calls, rest := textToolCalls(*msg.Content); len(calls) > 0 {
					msg.ToolCalls, msg.Content = calls, text(rest)
				}
			}
			if len(msg.ToolCalls) == 0 && blank(msg.Content) && nudge < 0 {
				// Thinking-only models (Mellum) sometimes put the whole answer
				// in their reasoning and leave the reply empty. Ask once for
				// the answer; the request is removed from history afterwards.
				nudge = len(hist)
				hist = append(hist, message{Role: "user", Content: text(answerNow)})
				continue
			}
			if nudge >= 0 {
				// trim may have shifted the history, so find the request by content
				for i := len(hist) - 1; i > 0; i-- {
					if hist[i].Role == "user" && hist[i].Content != nil && *hist[i].Content == answerNow {
						hist = append(hist[:i], hist[i+1:]...)
						break
					}
				}
			}
			hist = append(hist, msg)
			if len(msg.ToolCalls) == 0 {
				ch <- agent.Event{Kind: agent.EvDone, SessionID: sid, Usage: total}
				return
			}
			for _, tc := range msg.ToolCalls {
				var out string
				if gate := rg.check(tc); gate != "" {
					out = "ERROR: " + gate
					ch <- agent.Event{Kind: agent.EvToolStart, Tool: agent.Tool{ID: tc.ID, Name: tc.Function.Name}}
					ch <- agent.Event{Kind: agent.EvToolResult, Tool: agent.Tool{ID: tc.ID}, Result: "not yet: " + gate, IsError: true}
				} else if rep.see(tc) >= leanRepeat {
					out = "ERROR: you already made this exact call several times and the result will not change. Try a different approach, or answer the user with what you know."
					ch <- agent.Event{Kind: agent.EvToolStart, Tool: agent.Tool{ID: tc.ID, Name: tc.Function.Name}}
					ch <- agent.Event{Kind: agent.EvToolResult, Tool: agent.Tool{ID: tc.ID}, Result: "repeated call blocked", IsError: true}
				} else {
					out = clipTo(p.callTool(ctx, r, req.Mode, withReadLimit(tc), ch), leanMaxOut)
					rg.seen(tc, out)
				}
				hist = append(hist, message{Role: "tool", ToolCallID: tc.ID, Content: text(out)})
				if ctx.Err() != nil {
					ch <- agent.Event{Kind: agent.EvError, Err: errors.New("interrupted")}
					return
				}
			}
		}
		ch <- agent.Event{Kind: agent.EvDone, SessionID: sid, Usage: total, IsError: true, Text: fmt.Sprintf("stopped after %d steps", steps)}
	}()
	return ch, nil
}

// withReadLimit pages Read in small chunks unless the model asked otherwise.
func withReadLimit(tc toolCall) toolCall {
	if tc.Function.Name != "Read" {
		return tc
	}
	in, err := decodeArgs(tc.Function.Arguments)
	if err != nil || in["limit"] != nil {
		return tc
	}
	in["limit"] = leanReadLines
	b, _ := json.Marshal(in)
	tc.Function.Arguments = string(b)
	return tc
}
