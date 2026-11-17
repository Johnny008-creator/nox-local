package native

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Johnny008-creator/nox-local/internal/agent"
	"github.com/Johnny008-creator/nox-local/internal/docexport"
	"github.com/Johnny008-creator/nox-local/internal/mcpclient"
	"github.com/Johnny008-creator/nox-local/internal/paths"
)

type props map[string]any

func fn(name, desc string, p props, required ...string) map[string]any {
	return map[string]any{"type": "function", "function": map[string]any{
		"name": name, "description": desc,
		"parameters": map[string]any{"type": "object", "properties": p, "required": required},
	}}
}
func str(d string) map[string]any     { return map[string]any{"type": "string", "description": d} }
func num(d string) map[string]any     { return map[string]any{"type": "integer", "description": d} }
func boolean(d string) map[string]any { return map[string]any{"type": "boolean", "description": d} }

// ---------- permissions ----------

var (
	readOnly  = map[string]bool{"Read": true, "Glob": true, "Grep": true, "WebFetch": true, "WebSearch": true, "TodoWrite": true, "Artifact": true}
	riskyCmd  = regexp.MustCompile(`(?i)(\brm\s+-[a-z]*r|\brmdir\b|remove-item\b.*-recurse|\bdel\s+/[sq]|\bformat\b\s+[a-z]:|git\s+push\b.*(--force|-f\b)|git\s+reset\s+--hard|git\s+clean\s+-[a-z]*f|\bshutdown\b|\breboot\b|mkfs|dd\s+if=|curl[^|]*\|\s*(ba)?sh|iwr[^|]*\|\s*iex|drop\s+(table|database)|chmod\s+-r\s+777|:\(\)\s*\{)`)
	protected = regexp.MustCompile(`(?i)(^|[\\/])(\.env(\..*)?|\.git[\\/].*|id_(rsa|ed25519)(\.pub)?|.*\.(pem|key|pfx|p12))$`)
)

// policy decides what happens to a tool call in a mode:
// allow silently, ask the user (reason says why), or deny outright.
func policy(mode agent.Mode, name string, in map[string]any) (ask bool, reason string, deny string) {
	if mode == agent.ModeYolo {
		return false, "", ""
	}
	path, _ := in["file_path"].(string)
	cmd, _ := in["command"].(string)
	switch {
	case readOnly[name]:
		return false, "", ""
	case mode == agent.ModePlan && (name == "Write" || name == "Edit"):
		return false, "", "plan mode is read-only: describe the change in your plan instead of making it"
	case mode == agent.ModePlan, mode == agent.ModeManual:
		return true, mode.String() + " mode asks before every change", ""
	case path != "" && protected.MatchString(path):
		return true, "protected file", ""
	case name == "Bash" && riskyCmd.MatchString(cmd):
		return true, "potentially destructive command", ""
	}
	return false, "", ""
}

// mcpPolicy uses the server's own hints: read-only tools run freely,
// destructive ones are confirmed even in auto mode.
func mcpPolicy(mode agent.Mode, t mcpclient.Tool) (ask bool, reason string, deny string) {
	switch {
	case mode == agent.ModeYolo, t.ReadOnly:
		return false, "", ""
	case mode == agent.ModePlan:
		return false, "", "plan mode is read-only and " + t.Server + "." + t.Name + " may change things"
	case mode == agent.ModeManual:
		return true, "MCP tool from " + t.Server, ""
	case t.Destructive:
		return true, t.Server + " marks this tool as destructive", ""
	}
	return false, "", ""
}

// ---------- execution ----------

type runner struct {
	dir     string
	art     string   // folder for this session's artifacts
	without []string // built-in tools switched off for this turn
}

func (r runner) abs(p string) string {
	if p == "" {
		return r.dir
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(r.dir, p)
	}
	return filepath.Clean(p)
}

const maxOut = 30_000

func clip(s string) string {
	if len(s) > maxOut {
		return s[:maxOut/2] + fmt.Sprintf("\n… [%d bytes omitted] …\n", len(s)-maxOut) + s[len(s)-maxOut/2:]
	}
	return s
}

func (r runner) run(ctx context.Context, name string, in map[string]any) (out string, err error) {
	// A bug in one tool must end as an error result the model can react to,
	// not as a panic that leaves the turn hanging on "running…".
	defer func() {
		if p := recover(); p != nil {
			out, err = "", fmt.Errorf("%s crashed: %v", name, p)
		}
	}()
	s := func(k string) string { v, _ := in[k].(string); return v }
	n := func(k string) int {
		switch v := in[k].(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
		return 0
	}
	switch name {
	case "Read":
		return r.read(s("file_path"), n("offset"), n("limit"))
	case "Write":
		p := r.abs(s("file_path"))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(p, []byte(s("content")), 0o644); err != nil {
			return "", err
		}
		return fmt.Sprintf("wrote %d bytes to %s", len(s("content")), p), nil
	case "Edit":
		return r.edit(s("file_path"), s("old_string"), s("new_string"), in["replace_all"] == true)
	case "Glob":
		return r.glob(s("pattern"), s("path"))
	case "Grep":
		return r.grep(ctx, s("pattern"), s("path"), s("glob"))
	case "Bash":
		if bg, _ := in["background"].(bool); bg {
			return r.background(ctx, s("command"))
		}
		return r.bash(ctx, s("command"), n("timeout"))
	case "WebFetch":
		return webFetch(ctx, s("url"))
	case "WebSearch":
		return webSearch(ctx, searchURL, s("query"))
	}
	return "", fmt.Errorf("unknown tool %q", name)
}

func (r runner) read(path string, offset, limit int) (string, error) {
	b, err := os.ReadFile(r.abs(path))
	if err != nil {
		return "", err
	}
	if limit <= 0 {
		limit = 2000
	}
	if offset <= 0 {
		offset = 1
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	var out strings.Builder
	for i := offset - 1; i < len(lines) && i < offset-1+limit; i++ {
		fmt.Fprintf(&out, "%6d\t%s\n", i+1, lines[i])
	}
	if offset-1+limit < len(lines) {
		fmt.Fprintf(&out, "… %d more lines (use offset)\n", len(lines)-(offset-1+limit))
	}
	return clip(out.String()), nil
}

func (r runner) edit(path, old, new string, all bool) (string, error) {
	p := r.abs(path)
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	src := string(b)
	crlf := strings.Contains(src, "\r\n")
	if crlf { // models send \n; keep the file's line endings
		src = strings.ReplaceAll(src, "\r\n", "\n")
	}
	if old == "" {
		return "", errors.New("old_string is empty; use Write to create files")
	}
	c := strings.Count(src, old)
	switch {
	case c == 0:
		return "", errors.New("old_string not found in file (read it again and copy the exact text)")
	case c > 1 && !all:
		return "", fmt.Errorf("old_string matches %d places; add more context or set replace_all", c)
	}
	if all {
		src = strings.ReplaceAll(src, old, new)
	} else {
		src = strings.Replace(src, old, new, 1)
	}
	if crlf {
		src = strings.ReplaceAll(src, "\n", "\r\n")
	}
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		return "", err
	}
	n := 1
	if all {
		n = c
	}
	return fmt.Sprintf("edited %s (%d replacement(s))", p, n), nil
}

var skipDirs = map[string]bool{".git": true, "node_modules": true, "vendor": true, "dist": true, "build": true, ".venv": true, "__pycache__": true, "target": true}

// globRe converts a ** glob to a regexp over slash paths.
func globRe(pat string) (*regexp.Regexp, error) {
	pat = filepath.ToSlash(pat)
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pat); i++ {
		c := pat[i]
		switch {
		case c == '*' && i+1 < len(pat) && pat[i+1] == '*':
			i++
			if i+1 < len(pat) && pat[i+1] == '/' {
				i++
				b.WriteString("(?:.*/)?")
			} else {
				b.WriteString(".*")
			}
		case c == '*':
			b.WriteString("[^/]*")
		case c == '?':
			b.WriteString("[^/]")
		case c == '{':
			b.WriteString("(?:")
		case c == '}':
			b.WriteString(")")
		case c == ',':
			b.WriteString("|")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	b.WriteString("$")
	return regexp.Compile("(?i)" + b.String())
}

func (r runner) walk(root string, fnc func(rel string, d fs.DirEntry) error) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		return fnc(filepath.ToSlash(rel), d)
	})
}

var errEnough = errors.New("enough")

func (r runner) glob(pattern, path string) (string, error) {
	re, err := globRe(pattern)
	if err != nil {
		return "", err
	}
	if !strings.Contains(pattern, "/") { // "*.go" means anywhere
		re, _ = globRe("**/" + pattern)
	}
	root := r.abs(path)
	type hit struct {
		p string
		t time.Time
	}
	var hits []hit
	_ = r.walk(root, func(rel string, d fs.DirEntry) error {
		if re.MatchString(rel) {
			info, _ := d.Info()
			var t time.Time
			if info != nil {
				t = info.ModTime()
			}
			hits = append(hits, hit{rel, t})
			if len(hits) >= 1000 {
				return errEnough
			}
		}
		return nil
	})
	sort.Slice(hits, func(i, j int) bool { return hits[i].t.After(hits[j].t) })
	var out []string
	for i, h := range hits {
		if i == 300 {
			out = append(out, fmt.Sprintf("… %d more", len(hits)-300))
			break
		}
		out = append(out, h.p)
	}
	if len(out) == 0 {
		return "no files found", nil
	}
	return strings.Join(out, "\n"), nil
}

func (r runner) grep(ctx context.Context, pattern, path, glob string) (string, error) {
	if rg, err := exec.LookPath("rg"); err == nil {
		args := []string{"-n", "--no-heading", "--color", "never", "-M", "300", "-m", "50", pattern}
		if glob != "" {
			args = append(args, "-g", glob)
		}
		args = append(args, r.abs(path))
		out, _ := exec.CommandContext(ctx, rg, args...).Output()
		s := strings.ReplaceAll(string(out), filepath.ToSlash(r.dir)+"/", "")
		s = strings.ReplaceAll(s, r.dir+string(filepath.Separator), "")
		if strings.TrimSpace(s) == "" {
			return "no matches", nil
		}
		return clip(s), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	var gre *regexp.Regexp
	if glob != "" {
		gre, _ = globRe("**/" + glob)
	}
	root := r.abs(path)
	if st, err := os.Stat(root); err == nil && !st.IsDir() {
		return grepFile(re, root, filepath.Base(root), nil), nil
	}
	var out []string
	_ = r.walk(root, func(rel string, d fs.DirEntry) error {
		if gre != nil && !gre.MatchString(rel) {
			return nil
		}
		if info, _ := d.Info(); info != nil && info.Size() > 2<<20 {
			return nil
		}
		if s := grepFile(re, filepath.Join(root, rel), rel, &out); s != "" && len(out) > 300 {
			return errEnough
		}
		return nil
	})
	if len(out) == 0 {
		return "no matches", nil
	}
	return clip(strings.Join(out, "\n")), nil
}

func grepFile(re *regexp.Regexp, full, rel string, acc *[]string) string {
	f, err := os.Open(full)
	if err != nil {
		return ""
	}
	defer f.Close()
	var local []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for i := 1; sc.Scan(); i++ {
		line := sc.Text()
		if strings.ContainsRune(line, 0) {
			return "" // binary
		}
		if re.MatchString(line) {
			if len(line) > 300 {
				line = line[:300] + "…"
			}
			local = append(local, fmt.Sprintf("%s:%d: %s", rel, i, line))
		}
	}
	if acc != nil {
		*acc = append(*acc, local...)
		return strings.Join(local, "\n")
	}
	return strings.Join(local, "\n")
}

func (r runner) bash(ctx context.Context, command string, timeout int) (string, error) {
	if timeout <= 0 {
		timeout = 120
	}
	timeout = min(timeout, 600)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
			"[Console]::OutputEncoding=[Text.Encoding]::UTF8; "+command)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-lc", command)
	}
	cmd.Dir = r.dir
	ownGroup(cmd)
	// If something still holds the output open after the kill, stop waiting.
	cmd.WaitDelay = 2 * time.Second
	out, err := cmd.CombinedOutput()
	s := strings.ReplaceAll(string(out), "\r\n", "\n")
	if ctx.Err() == context.DeadlineExceeded {
		return clip(s), fmt.Errorf("timed out after %ds · servers and watchers need background: true", timeout)
	}
	if err != nil {
		if s == "" {
			s = err.Error()
		}
		return clip(s + "\n[" + err.Error() + "]"), errors.New("command failed")
	}
	if strings.TrimSpace(s) == "" {
		s = "(no output)"
	}
	return clip(s), nil
}

var (
	tagRe   = regexp.MustCompile(`(?is)<(script|style|noscript)[^>]*>.*?</(script|style|noscript)>|<[^>]+>`)
	spaceRe = regexp.MustCompile(`\n\s*\n\s*\n+`)
)

// ObscuraPath finds the Obscura headless browser: $NOX_OBSCURA, PATH, or
// ~/.nox-local/bin. "" when it is not installed.
func ObscuraPath() string {
	if p := os.Getenv("NOX_OBSCURA"); p != "" {
		return p
	}
	if p, err := exec.LookPath("obscura"); err == nil {
		return p
	}
	if p := filepath.Join(paths.Home(), "bin", "obscura"); fileExists(p) {
		return p
	}
	return ""
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// obscuraFetch renders the page (JavaScript included) and returns it as
// markdown. Obscura refuses private and loopback addresses by default.
func obscuraFetch(ctx context.Context, bin, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "fetch", url, "--dump", "markdown", "--quiet", "--timeout", "20").Output()
	if err != nil {
		return "", err
	}
	s := tidyMarkdown(string(out))
	if s == "" {
		return "", errors.New("empty page")
	}
	return s, nil
}

// tidyMarkdown drops indentation-only lines and runs of blank lines, which
// rendered pages produce plenty of and small context windows can't afford.
func tidyMarkdown(s string) string {
	var b strings.Builder
	blank := 0
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimRight(l, " \t\r")
		if strings.TrimSpace(l) == "" {
			blank++
			if blank > 1 {
				continue
			}
			l = ""
		} else {
			blank = 0
			l = strings.TrimLeft(l, " \t") // layout indentation; list nesting is noise here
		}
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

func webFetch(ctx context.Context, url string) (string, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "", errors.New("url must start with http:// or https://")
	}
	if bin := ObscuraPath(); bin != "" {
		if md, err := obscuraFetch(ctx, bin, url); err == nil {
			return clip(md), nil
		}
		// fall back to a plain request (binary files, Obscura errors)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "nox")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	s := string(b)
	if strings.Contains(resp.Header.Get("Content-Type"), "html") {
		s = tagRe.ReplaceAllString(s, "")
		s = strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'").Replace(s)
		s = spaceRe.ReplaceAllString(s, "\n\n")
	}
	if resp.StatusCode >= 400 {
		return clip(s), fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return clip(strings.TrimSpace(s)), nil
}

// searchURL is DuckDuckGo's HTML endpoint: no API key, results in plain
// markup. A var so tests can point it at a local server.
var searchURL = "https://html.duckduckgo.com/html/"

var (
	resultRe  = regexp.MustCompile(`(?s)class="result__a" href="([^"]+)"[^>]*>(.*?)</a>`)
	snippetRe = regexp.MustCompile(`(?s)class="result__snippet"[^>]*>(.*?)</a>`)
)

// webSearch returns up to 8 results as "title\nurl\nsnippet" blocks.
func webSearch(ctx context.Context, endpoint, query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", errors.New("query is empty")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(url.Values{"q": {query}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) nox-local")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("search HTTP %d", resp.StatusCode)
	}
	page := string(b)
	links := resultRe.FindAllStringSubmatch(page, 8)
	snips := snippetRe.FindAllStringSubmatch(page, 8)
	if len(links) == 0 {
		return "no results (or the search engine refused the request)", nil
	}
	plain := func(h string) string {
		h = tagRe.ReplaceAllString(h, "")
		h = strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'", "&#x27;", "'").Replace(h)
		return strings.Join(strings.Fields(h), " ")
	}
	var out strings.Builder
	for i, l := range links {
		link := l[1]
		if u, err := url.Parse(link); err == nil && u.Query().Get("uddg") != "" { // DDG redirect
			link = u.Query().Get("uddg")
		}
		fmt.Fprintf(&out, "%d. %s\n   %s\n", i+1, plain(l[2]), link)
		if i < len(snips) {
			fmt.Fprintf(&out, "   %s\n", plain(snips[i][1]))
		}
	}
	return strings.TrimSpace(out.String()), nil
}

// artifactKinds are what the viewers can show.
var artifactKinds = map[string]string{"markdown": "md", "html": "html", "code": "txt", "svg": "svg", "canvas": "html"}

var safeID = regexp.MustCompile(`[^a-z0-9-]+`)

var langExt = map[string]string{"python": "py", "javascript": "js", "typescript": "ts", "golang": "go",
	"bash": "sh", "shell": "sh", "rust": "rs", "ruby": "rb", "markdown": "md", "yaml": "yml", "c++": "cpp", "csharp": "cs", "lua": "lua"}

// saveArtifact validates an Artifact call, writes it to dir (one file per
// artifact plus a .json record with its version) and returns it.
func saveArtifact(dir string, in map[string]any) (*agent.Artifact, error) {
	s := func(k string) string { v, _ := in[k].(string); return strings.TrimSpace(v) }
	a := &agent.Artifact{ID: s("id"), Title: s("title"), Kind: strings.ToLower(s("kind")), Lang: s("language")}
	a.Content, _ = in["content"].(string)
	ext, ok := artifactKinds[a.Kind]
	switch {
	case !ok:
		return nil, errors.New("kind must be markdown, html, code, svg or canvas")
	case strings.TrimSpace(a.Content) == "":
		return nil, errors.New("content is empty")
	case a.Title == "":
		return nil, errors.New("title is required")
	}
	if a.ID == "" {
		a.ID = a.Title
	}
	a.ID = strings.Trim(safeID.ReplaceAllString(strings.ToLower(a.ID), "-"), "-")
	if a.ID == "" {
		a.ID = "artifact"
	}
	if len(a.ID) > 48 {
		a.ID = a.ID[:48]
	}
	if a.Kind == "code" && a.Lang != "" {
		l := strings.ToLower(a.Lang)
		if e, ok := langExt[l]; ok {
			ext = e
		} else if e := strings.Trim(safeID.ReplaceAllString(l, ""), "-"); e != "" && len(e) <= 8 {
			ext = e
		}
	}
	meta := filepath.Join(dir, a.ID+".json")
	var prev agent.Artifact
	if b, err := os.ReadFile(meta); err == nil && json.Unmarshal(b, &prev) == nil {
		a.Version = prev.Version
	}
	a.Version++
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	a.Path = filepath.Join(dir, a.ID+"."+ext)
	file := a.Content
	if a.Kind == "canvas" { // saved as a page that runs on its own in a browser
		file = agent.CanvasPage(a.Content, agent.CanvasTheme{})
	}
	if err := os.WriteFile(a.Path, []byte(file), 0o644); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(a)
	_ = os.WriteFile(meta, b, 0o644)
	return a, nil
}

// exportFormats reads the Artifact tool's export list.
func exportFormats(in map[string]any) []string {
	var out []string
	switch v := in["export"].(type) {
	case []any:
		for _, x := range v {
			if f, ok := x.(string); ok {
				out = append(out, strings.ToLower(strings.TrimSpace(f)))
			}
		}
	case string: // small models sometimes send "docx,pdf"
		for _, f := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' }) {
			out = append(out, strings.ToLower(f))
		}
	}
	return out
}

// exportArtifact writes a markdown artifact as DOCX/PDF into dir and
// returns the paths written.
func exportArtifact(a *agent.Artifact, dir string, formats []string) ([]string, error) {
	if a.Kind != "markdown" {
		return nil, errors.New("only markdown artifacts can be exported")
	}
	var files []string
	for _, f := range formats {
		var write func(io.Writer, string, string) error
		switch f {
		case "docx":
			write = docexport.DOCX
		case "pdf":
			write = docexport.PDF
		default:
			return files, fmt.Errorf("unknown export format %q (docx or pdf)", f)
		}
		p := filepath.Join(dir, a.ID+"."+f)
		out, err := os.Create(p)
		if err != nil {
			return files, err
		}
		err = write(out, a.Title, a.Content)
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return files, err
		}
		files = append(files, p)
	}
	return files, nil
}

// parseTodos validates a TodoWrite call and returns the list plus a short
// confirmation for the model.
func parseTodos(in map[string]any) ([]agent.Todo, string, error) {
	raw, _ := json.Marshal(in["todos"])
	var todos []agent.Todo
	if err := json.Unmarshal(raw, &todos); err != nil || len(todos) == 0 {
		return nil, "", errors.New(`todos must be a non-empty list of {"content", "status"}`)
	}
	done := 0
	for i, t := range todos {
		switch t.Status {
		case "pending", "in_progress":
		case "completed":
			done++
		default:
			todos[i].Status = "pending"
		}
	}
	return todos, fmt.Sprintf("task list saved (%d/%d done)", done, len(todos)), nil
}

// decodeArgs parses tool-call arguments, tolerating empty strings.
func decodeArgs(s string) (map[string]any, error) {
	in := map[string]any{}
	if strings.TrimSpace(s) == "" {
		return in, nil
	}
	err := json.Unmarshal([]byte(s), &in)
	return in, err
}
