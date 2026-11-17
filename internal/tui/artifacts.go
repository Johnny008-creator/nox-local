package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Johnny008-creator/nox-local/internal/agent"
	"github.com/Johnny008-creator/nox-local/internal/paths"
)

// Artifacts: what a conversation produced that is more than text — charts,
// graphics, interactive pages, diagrams and documents. The desktop draws them
// in a panel; a terminal cannot draw a PNG, so /artifacts lists them and opens
// the chosen one in the system viewer. That is the honest version of the same
// idea: you still get a list of what was made, and one keystroke to see it.

// artKinds are the fenced blocks treated as artifacts.
var artKinds = map[string]string{
	"chart":   "Chart",
	"svg":     "Graphic",
	"html":    "Interactive",
	"mermaid": "Diagram",
	"canvas":  "Canvas",
	"code":    "Code",
}

// artifact is one thing the conversation made.
type artifact struct {
	Kind  string // chart | svg | html | mermaid | doc
	Label string // human name for the list
	Title string
	Body  string   // the fenced block's content ("" for a document)
	Files []string // documents: absolute paths
	At    time.Time
}

// artifactOf turns an artifact the backend reported into a list row.
func artifactOf(a agent.Artifact) artifact {
	label := artKinds[a.Kind]
	switch {
	case label != "":
	case a.Kind == "markdown" || a.Kind == "document":
		label = "Document"
	default:
		label = "File"
	}
	row := artifact{Kind: a.Kind, Label: label, Title: a.Title, Body: a.Content}
	if a.Path != "" {
		row.Files = append(row.Files, a.Path)
	}
	row.Files = append(row.Files, a.Files...)
	return row
}

// fenceRe finds a fenced block of an artifact kind. A non-greedy body keeps a
// following code block from being swallowed.
var fenceRe = regexp.MustCompile("(?s)```(chart|svg|html|mermaid)[^\\n]*\\n(.*?)```")

// artifacts collects every artifact in the conversation, newest first. Text
// entries carry the fenced blocks; Document tool rows carry the files the
// tool wrote.
func (m *Model) artifacts() []artifact {
	out := append([]artifact(nil), m.artifactsMade...)
	for _, e := range m.entries {
		switch e.kind {
		case kText:
			for _, mm := range fenceRe.FindAllStringSubmatch(e.text, -1) {
				kind := mm[1]
				body := mm[2]
				out = append(out, artifact{Kind: kind, Label: artKinds[kind], Title: artifactTitle(kind, body), Body: body})
			}
		case kTool:
			// A model that writes a chart or a page as a file is making an
			// artifact too — often a better one than a fenced block. Anything
			// it wrote under outputs/ counts.
			if e.isErr || !e.done {
				continue
			}
			if e.tool.Name == "Write" || e.tool.Name == "Edit" {
				pth, _ := e.tool.Input["file_path"].(string)
				rel := filepath.ToSlash(pth)
				// Both "outputs/x.html" and "/abs/project/outputs/x.html" are
				// the project's outputs folder.
				if pth == "" || !(strings.HasPrefix(rel, "outputs/") || strings.Contains(rel, "/outputs/")) {
					continue
				}
				abs := pth
				if !filepath.IsAbs(abs) {
					abs = filepath.Join(m.dir, pth)
				}
				out = append(out, artifact{Kind: "file", Label: artifactLabel(abs), Title: strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs)), Files: []string{abs}})
				continue
			}
			if e.tool.Name != "Document" {
				continue
			}
			// The Document tool writes into outputs/; anything else on the
			// stdout lines (a preview image, a URL) is not the artifact.
			var files []string
			for _, l := range strings.Split(e.result, "\n") {
				l = strings.TrimSpace(l)
				if !filepath.IsAbs(l) || !strings.Contains(filepath.ToSlash(l), "/outputs/") {
					continue
				}
				if st, err := os.Stat(l); err == nil && !st.IsDir() {
					files = append(files, l)
				}
			}
			if len(files) == 0 {
				continue
			}
			title, _ := e.tool.Input["title"].(string)
			if title == "" {
				title = filepath.Base(files[0])
			}
			out = append(out, artifact{Kind: "doc", Label: "Document", Title: title, Files: files})
		}
	}
	// Newest first: the last entry is the most recent thing made.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	// Number them so the list reads like Claude Code's.
	for i := range out {
		if out[i].Title == "" {
			out[i].Title = fmt.Sprintf("%s %d", out[i].Label, i+1)
		}
	}
	return out
}

// artifactLabel names a file artifact by what it is, for the list.
func artifactLabel(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm":
		return "Interactive"
	case ".svg":
		return "Graphic"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return "Image"
	case ".pdf":
		return "PDF"
	case ".docx":
		return "Document"
	case ".pptx":
		return "Slides"
	case ".xlsx":
		return "Sheet"
	case ".mmd":
		return "Diagram"
	}
	return "File"
}

// artifactTitle pulls a name out of the block where one is easy to find: a
// chart's JSON title, an html <title>, an svg <title>.
func artifactTitle(kind, body string) string {
	switch kind {
	case "chart":
		if m := regexp.MustCompile(`"title"\s*:\s*"([^"]{1,70})"`).FindStringSubmatch(body); m != nil {
			return m[1]
		}
	case "html":
		if m := regexp.MustCompile(`(?i)<title[^>]*>([^<]{1,70})</title>`).FindStringSubmatch(body); m != nil {
			return strings.TrimSpace(m[1])
		}
		if m := regexp.MustCompile(`(?i)<h1[^>]*>([^<]{1,70})</h1>`).FindStringSubmatch(body); m != nil {
			return strings.TrimSpace(m[1])
		}
	case "svg":
		if m := regexp.MustCompile(`(?i)<title[^>]*>([^<]{1,70})</title>`).FindStringSubmatch(body); m != nil {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

// artFile writes an artifact to a file the system viewer can open, and
// returns its path. Files live under ~/.nox/cache/artifacts and are rewritten
// on each open, so an edited answer shows its newest version.
func artFile(a artifact) (string, error) {
	// A document and a file the model wrote are already on disk in the
	// project; open them where they are, so an edit made later is visible.
	if (a.Kind == "doc" || a.Kind == "file") && len(a.Files) > 0 {
		return a.Files[0], nil
	}
	dir := paths.Sub("cache", "artifacts")
	var ext string
	var body string
	switch a.Kind {
	case "chart":
		// A chart is JSON drawn by the desktop's viz.js. The terminal has no
		// renderer, so it is shown as a small standalone HTML page that plots
		// the values with inline SVG — enough to read the numbers.
		ext, body = "html", chartHTML(a.Title, a.Body)
	case "svg":
		ext, body = "svg", a.Body
	case "html":
		ext, body = "html", a.Body
	case "mermaid":
		ext, body = "mmd", a.Body
	case "canvas":
		// Live canvas is drawn by the desktop. In a terminal you get the
		// JavaScript itself, which is the honest version of the same file.
		ext, body = "js", a.Body
	default:
		ext, body = "txt", a.Body
	}
	name := strings.Join(strings.Fields(regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(a.Title, " ")), "-")
	if name == "" {
		name = a.Kind
	}
	if len(name) > 60 {
		name = name[:60]
	}
	p := filepath.Join(dir, name+"."+ext)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// chartHTML is a minimal renderer for the chart JSON the desktop understands:
// {title, type: bar|line|pie, data: [{label, value}]}. It is a fallback for
// the terminal, not a replacement for the desktop panel.
func chartHTML(title, body string) string {
	esc := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return `<!doctype html><meta charset="utf-8"><title>` + esc.Replace(title) + `</title>
<style>body{font:15px/1.5 system-ui,sans-serif;margin:2rem;background:#12151c;color:#e7e9f0}
h1{font-size:1.2rem;font-weight:600}.row{display:flex;gap:.5rem;align-items:center;margin:.35rem 0}
.lbl{width:12rem;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:#8d93a6}
.bar{height:1.1rem;background:linear-gradient(90deg,#2dd4bf,#a3e635);border-radius:3px}
.val{color:#8d93a6;font-variant-numeric:tabular-nums}</style>
<h1>` + esc.Replace(title) + `</h1><div id="c"></div>
<script>
const raw = ` + "`" + strings.ReplaceAll(body, "`", "\\`") + "`" + `;
let c; try { c = JSON.parse(raw); } catch (e) { document.getElementById("c").textContent = raw; throw e; }
const data = c.data || c.values || [];
const max = Math.max(1, ...data.map(d => Math.abs(d.value ?? d.y ?? 0)));
const box = document.getElementById("c");
for (const d of data) {
  const v = d.value ?? d.y ?? 0;
  const row = document.createElement("div"); row.className = "row";
  const l = document.createElement("div"); l.className = "lbl"; l.textContent = d.label ?? d.x ?? "";
  const b = document.createElement("div"); b.className = "bar"; b.style.width = (60 * Math.abs(v) / max) + "%";
  const n = document.createElement("div"); n.className = "val"; n.textContent = v;
  row.append(l, b, n); box.appendChild(row);
}
</script>`
}

// cmdArtifacts opens the list; picking one writes it out and opens it.
func cmdArtifacts(m *Model, arg string) tea.Cmd {
	arts := m.artifacts()
	if len(arts) == 0 {
		m.info("no artifacts in this conversation yet · charts, graphics, interactive pages, diagrams and documents show up here")
		return nil
	}
	if n, err := parseIndex(arg); err == nil && n > 0 && n <= len(arts) {
		return m.openArtifact(arts[n-1])
	}
	items := make([]pickItem, len(arts))
	for i, a := range arts {
		desc := a.Label
		if a.Kind == "doc" || a.Kind == "file" {
			desc += " · " + filepath.Base(a.Files[0])
		}
		items[i] = pickItem{value: fmt.Sprint(i + 1), label: fmt.Sprintf("%d. %s", i+1, truncRight(a.Title, 60)), desc: desc}
	}
	m.picker = newPicker(fmt.Sprintf("Artifacts · %s from this conversation", plural(len(arts), "item")), items, true,
		func(m *Model, it pickItem) tea.Cmd {
			var n int
			fmt.Sscan(it.value, &n)
			if n < 1 || n > len(arts) {
				return nil
			}
			return m.openArtifact(arts[n-1])
		})
	return nil
}

// openArtifact writes the artifact out and hands it to the system viewer.
func (m *Model) openArtifact(a artifact) tea.Cmd {
	p, err := artFile(a)
	if err != nil {
		m.errorf("artifact: " + err.Error())
		return nil
	}
	opener := "xdg-open"
	switch runtime.GOOS {
	case "darwin":
		opener = "open"
	case "windows":
		opener = "explorer"
	}
	c := exec.Command(opener, p)
	if err := c.Start(); err != nil {
		m.errorf("open: " + err.Error() + " · the file is at " + p)
		return nil
	}
	go c.Wait()
	m.info(a.Label + " · " + p)
	return nil
}

// parseIndex reads a 1-based number from an argument.
func parseIndex(s string) (int, error) {
	var n int
	_, err := fmt.Sscan(strings.TrimSpace(s), &n)
	return n, err
}
