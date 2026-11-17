package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/Johnny008-creator/nox-local/internal/agent"
)

func TestArtifactsFindsFencedBlocks(t *testing.T) {
	m := &Model{}
	m.entries = []entry{
		{kind: kText, text: "Here is a chart:\n\n```chart\n{\"title\":\"Revenue\",\"data\":[{\"label\":\"A\",\"value\":3}]}\n```\n"},
		{kind: kText, text: "And a page:\n\n```html\n<html><head><title>Report</title></head><body></body></html>\n```\n"},
	}
	got := m.artifacts()
	if len(got) != 2 {
		t.Fatalf("got %d artifacts, want 2: %+v", len(got), got)
	}
	// Newest first: the html block was in the later entry.
	if got[0].Kind != "html" || got[0].Title != "Report" {
		t.Errorf("first artifact = %+v, want the html page titled Report", got[0])
	}
	if got[1].Kind != "chart" || got[1].Title != "Revenue" {
		t.Errorf("second artifact = %+v, want the chart titled Revenue", got[1])
	}
}

// A code fence that is not an artifact kind must not be collected, and a
// second block must not be swallowed by the first.
func TestArtifactsIgnoresCodeAndKeepsBlocksSeparate(t *testing.T) {
	m := &Model{}
	m.entries = []entry{{kind: kText, text: "```go\nfunc main() {}\n```\n\n```svg\n<svg><title>Logo</title></svg>\n```\n"}}
	got := m.artifacts()
	if len(got) != 1 || got[0].Kind != "svg" || got[0].Title != "Logo" {
		t.Fatalf("got %+v, want one svg titled Logo", got)
	}
}

func TestArtifactsFromDocumentTool(t *testing.T) {
	p := t.TempDir() + "/outputs/report.pdf"
	if err := os.MkdirAll(p[:strings.LastIndex(p, "/")], 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("pdf"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &Model{}
	m.entries = []entry{{
		kind:   kTool,
		tool:   agent.Tool{Name: "Document"},
		result: "/tmp/a.png\n" + p + "\nhttps://example.com/x",
		done:   true,
	}}
	got := m.artifacts()
	if len(got) != 1 || got[0].Kind != "doc" {
		t.Fatalf("got %+v, want one document", got)
	}
	if len(got[0].Files) != 1 || got[0].Files[0] != p {
		t.Errorf("files = %v, want only the absolute non-URL path", got[0].Files)
	}
}

func TestArtifactsSkipsFailedDocument(t *testing.T) {
	m := &Model{}
	m.entries = []entry{{kind: kTool, tool: agent.Tool{Name: "Document"}, result: "/tmp/x.pdf", done: true, isErr: true}}
	if got := m.artifacts(); len(got) != 0 {
		t.Errorf("a failed Document call produced %+v", got)
	}
}

func TestArtFileWritesOpenableFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p, err := artFile(artifact{Kind: "svg", Title: "My Logo", Body: "<svg/>"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "<svg/>" {
		t.Errorf("write lost the body: %q", b)
	}
	if !strings.HasSuffix(p, ".svg") {
		t.Errorf("path %q should end in .svg so the viewer picks the right app", p)
	}
	if strings.Contains(p, "My Logo") {
		t.Errorf("path %q should be slugged, not contain spaces", p)
	}
}

func TestArtFileChartBecomesHTML(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p, err := artFile(artifact{Kind: "chart", Title: "Sales", Body: `{"data":[{"label":"a","value":1}]}`})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if !strings.HasSuffix(p, ".html") {
		t.Errorf("a chart should open as html, got %q", p)
	}
	if !strings.Contains(string(b), "Sales") || !strings.Contains(string(b), "\"a\"") {
		t.Errorf("chart html lacks the title or the data:\n%s", b)
	}
}

func TestArtifactTitleFallbacks(t *testing.T) {
	if got := artifactTitle("mermaid", "graph TD; A-->B"); got != "" {
		t.Errorf("mermaid has no extractable title, got %q", got)
	}
	if got := artifactTitle("chart", "not json"); got != "" {
		t.Errorf("a broken chart body should have no title, got %q", got)
	}
}

// A model that saves a chart as a file under outputs/ made an artifact, even
// though no fenced block appears in its answer.
func TestArtifactsFromWrittenFile(t *testing.T) {
	dir := t.TempDir()
	m := &Model{dir: dir}
	m.entries = []entry{{
		kind: kTool,
		tool: agent.Tool{Name: "Write", Input: map[string]any{"file_path": "outputs/quarterly-sales.html"}},
		done: true,
	}}
	got := m.artifacts()
	if len(got) != 1 {
		t.Fatalf("got %+v, want one artifact", got)
	}
	if got[0].Label != "Interactive" || got[0].Title != "quarterly-sales" {
		t.Errorf("artifact = %+v", got[0])
	}
	if !strings.HasSuffix(got[0].Files[0], "outputs/quarterly-sales.html") || !strings.HasPrefix(got[0].Files[0], "/") {
		t.Errorf("path %q should be absolute and inside outputs/", got[0].Files[0])
	}
}

func TestArtifactsIgnoresSourceFiles(t *testing.T) {
	m := &Model{dir: t.TempDir()}
	m.entries = []entry{{
		kind: kTool,
		tool: agent.Tool{Name: "Write", Input: map[string]any{"file_path": "internal/parser.go"}},
		done: true,
	}}
	if got := m.artifacts(); len(got) != 0 {
		t.Errorf("a source file is not an artifact: %+v", got)
	}
}
