package native

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Johnny008-creator/N0X-Local/internal/agent"
)

func TestPolicy(t *testing.T) {
	cases := []struct {
		mode      agent.Mode
		tool, cmd string
		path      string
		ask, deny bool
	}{
		{agent.ModeAuto, "Read", "", "a.go", false, false},
		{agent.ModeAuto, "Edit", "", "a.go", false, false},
		{agent.ModeAuto, "Edit", "", ".env", true, false},
		{agent.ModeAuto, "Bash", "go test ./...", "", false, false},
		{agent.ModeAuto, "Bash", "rm -rf build", "", true, false},
		{agent.ModeAuto, "Bash", "git push --force origin main", "", true, false},
		{agent.ModeManual, "Bash", "ls", "", true, false},
		{agent.ModePlan, "Edit", "", "a.go", false, true},
		{agent.ModePlan, "Grep", "", "", false, false},
		{agent.ModeYolo, "Bash", "rm -rf /", "", false, false},
	}
	for _, c := range cases {
		in := map[string]any{"command": c.cmd, "file_path": c.path}
		ask, _, deny := policy(c.mode, c.tool, in)
		if ask != c.ask || (deny != "") != c.deny {
			t.Errorf("%s %s %q %q: ask=%v deny=%q", c.mode, c.tool, c.cmd, c.path, ask, deny)
		}
	}
}

func TestTools(t *testing.T) {
	dir := t.TempDir()
	r := runner{dir: dir}
	ctx := context.Background()
	if _, err := r.run(ctx, "Write", map[string]any{"file_path": "sub/a.go", "content": "package a\n\nfunc A() int { return 1 }\n"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.run(ctx, "Edit", map[string]any{"file_path": "sub/a.go", "old_string": "return 1", "new_string": "return 2"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "sub", "a.go"))
	if !strings.Contains(string(b), "return 2") {
		t.Fatalf("edit not applied: %s", b)
	}
	if _, err := r.run(ctx, "Edit", map[string]any{"file_path": "sub/a.go", "old_string": "missing", "new_string": "x"}); err == nil {
		t.Fatal("edit of missing text should fail")
	}
	out, _ := r.run(ctx, "Read", map[string]any{"file_path": "sub/a.go"})
	if !strings.Contains(out, "3\tfunc A()") {
		t.Fatalf("read output: %q", out)
	}
	out, _ = r.run(ctx, "Glob", map[string]any{"pattern": "**/*.go"})
	if strings.TrimSpace(out) != "sub/a.go" {
		t.Fatalf("glob: %q", out)
	}
	out, _ = r.run(ctx, "Grep", map[string]any{"pattern": "func A"})
	if !strings.Contains(out, "a.go:3") {
		t.Fatalf("grep: %q", out)
	}
	out, err := r.run(ctx, "Bash", map[string]any{"command": "echo nox-ok"})
	if err != nil || !strings.Contains(out, "nox-ok") {
		t.Fatalf("bash: %q %v", out, err)
	}
}

func TestTrimKeepsSystemAndRecent(t *testing.T) {
	big := strings.Repeat("x", 5000)
	h := []message{{Role: "system", Content: text("sys")}}
	for i := 0; i < 20; i++ {
		h = append(h, message{Role: "user", Content: text("q")}, message{Role: "tool", ToolCallID: "t", Content: text(big)})
	}
	out := trim(h, 10_000)
	if out[0].Role != "system" {
		t.Fatal("system prompt dropped")
	}
	if *out[len(out)-1].Content != big {
		t.Fatal("most recent tool output was trimmed")
	}
}

func TestWebSearchParsesResults(t *testing.T) {
	page := `<a rel="nofollow" class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fgo.dev%2Fdoc&amp;rut=x">The <b>Go</b> docs</a>
<a class="result__snippet" href="x">Official <b>Go</b> documentation &amp; tutorials</a>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("q") != "golang docs" {
			t.Errorf("query = %q", r.Form.Get("q"))
		}
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()
	out, err := webSearch(context.Background(), srv.URL, "golang docs")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"1. The Go docs", "https://go.dev/doc", "Official Go documentation & tutorials"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestParseTodos(t *testing.T) {
	in := map[string]any{"todos": []any{
		map[string]any{"content": "read code", "status": "completed"},
		map[string]any{"content": "fix bug", "status": "in_progress"},
		map[string]any{"content": "test", "status": "weird"},
	}}
	todos, msg, err := parseTodos(in)
	if err != nil || len(todos) != 3 || todos[2].Status != "pending" || msg != "task list saved (1/3 done)" {
		t.Fatalf("todos=%+v msg=%q err=%v", todos, msg, err)
	}
	if _, _, err := parseTodos(map[string]any{"todos": "nope"}); err == nil {
		t.Fatal("bad input accepted")
	}
}

func TestSaveArtifactVersions(t *testing.T) {
	dir := t.TempDir()
	a, err := saveArtifact(dir, map[string]any{"title": "Pricing Page", "kind": "html", "content": "<h1>v1</h1>"})
	if err != nil || a.ID != "pricing-page" || a.Version != 1 || !strings.HasSuffix(a.Path, "pricing-page.html") {
		t.Fatalf("first save: %+v %v", a, err)
	}
	a, err = saveArtifact(dir, map[string]any{"id": "pricing-page", "title": "Pricing Page", "kind": "html", "content": "<h1>v2</h1>"})
	if err != nil || a.Version != 2 {
		t.Fatalf("update: %+v %v", a, err)
	}
	if b, _ := os.ReadFile(a.Path); string(b) != "<h1>v2</h1>" {
		t.Fatalf("file holds %q", b)
	}
	c, err := saveArtifact(dir, map[string]any{"title": "calc", "kind": "code", "language": "Python", "content": "print(1)"})
	if err != nil || !strings.HasSuffix(c.Path, "calc.py") {
		t.Fatalf("code artifact: %+v %v", c, err)
	}
	if _, err := saveArtifact(dir, map[string]any{"title": "x", "kind": "pdf", "content": "y"}); err == nil {
		t.Fatal("unknown kind accepted")
	}
}

func TestTidyMarkdown(t *testing.T) {
	in := "# Title\n      \n\n\n        [Skip](#main)\n   \n          - item one\n\n\n\ntext   \n"
	want := "# Title\n\n[Skip](#main)\n\n- item one\n\ntext"
	if got := tidyMarkdown(in); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestExportArtifact(t *testing.T) {
	dir := t.TempDir()
	a := &agent.Artifact{ID: "report", Title: "Report", Kind: "markdown", Content: "# Hi\n\nText **bold**."}
	files, err := exportArtifact(a, dir, exportFormats(map[string]any{"export": []any{"docx", "pdf"}}))
	if err != nil {
		if strings.Contains(err.Error(), "fonts") {
			t.Skip(err)
		}
		t.Fatal(err)
	}
	if len(files) != 2 || !strings.HasSuffix(files[0], "report.docx") || !strings.HasSuffix(files[1], "report.pdf") {
		t.Fatalf("files = %v", files)
	}
	if got := exportFormats(map[string]any{"export": "docx, PDF"}); len(got) != 2 || got[1] != "pdf" {
		t.Fatalf("string export list: %v", got)
	}
	if _, err := exportArtifact(&agent.Artifact{Kind: "html"}, dir, []string{"pdf"}); err == nil {
		t.Fatal("html export accepted")
	}
}

func TestSaveCanvasArtifactWritesRunnablePage(t *testing.T) {
	dir := t.TempDir()
	a, err := saveArtifact(dir, map[string]any{"title": "Orbit", "kind": "canvas", "content": "function draw(ctx,w,h,t){ctx.fillRect(0,0,10,10)}"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Content != "function draw(ctx,w,h,t){ctx.fillRect(0,0,10,10)}" || filepath.Ext(a.Path) != ".html" {
		t.Fatalf("artifact = %+v", a)
	}
	b, _ := os.ReadFile(a.Path)
	if !strings.Contains(string(b), "<canvas") || !strings.Contains(string(b), "ctx.fillRect(0,0,10,10)") {
		t.Error("saved file is not the wrapped canvas page")
	}
}
