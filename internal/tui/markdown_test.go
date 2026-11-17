package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestMarkdown(t *testing.T) {
	out, ok := renderMarkdown("## Plan\n\n1. Read `config.go`\n2. Add **NVIDIA**\n\n```go\nfunc main() {}\n```", 60)
	if !ok {
		t.Fatal("render failed")
	}
	plain := ansi.Strip(out)
	for _, want := range []string{"Plan", "config.go", "NVIDIA", "func main()"} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing %q in:\n%s", want, plain)
		}
	}
	for _, l := range strings.Split(out, "\n") {
		if ansi.StringWidth(l) > 60 {
			t.Errorf("line wider than 60: %q", ansi.Strip(l))
		}
	}
}
