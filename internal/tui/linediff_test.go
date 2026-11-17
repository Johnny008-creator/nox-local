package tui

import (
	"strings"
	"testing"

	"github.com/Johnny008-creator/nox-local/internal/agent"
)

func signs(d []diffLine) string {
	var b strings.Builder
	for _, l := range d {
		b.WriteByte(byte(l.sign))
	}
	return b.String()
}

// TestEditDiffShowsOnlyChanges: an Edit whose old_string carries context
// (the live test: +10 −6 for a 4-line change) shows the real change.
func TestEditDiffShowsOnlyChanges(t *testing.T) {
	old := "def cena(kw):\n    return CENY[kw]\n\n\nif __name__ == \"__main__\":\n    print(cena(9))"
	new := "def cena(kw):\n    return CENY[kw]\n\n\ndef cena_po_dotaci(kw):\n    return max(0, cena(kw) - 90_000)\n\n\nif __name__ == \"__main__\":\n    print(cena_po_dotaci(9))"
	d := diffFor(agent.Tool{Name: "Edit", Input: map[string]any{"file_path": "app.py", "old_string": old, "new_string": new}})
	var add, del int
	for _, l := range d {
		switch l.sign {
		case '+':
			add++
		case '-':
			del++
		}
	}
	if add != 5 || del != 1 {
		t.Errorf("+%d −%d, want +5 −1: %q", add, del, signs(d))
	}
	// two context lines before the change (the blank lines), one between
	if got := signs(d); got != "  ++++ -+" {
		t.Errorf("signs = %q", got)
	}
}

func TestLineDiff(t *testing.T) {
	cases := []struct {
		a, b []string
		want string
	}{
		{nil, []string{"x", "y"}, "++"},
		{[]string{"x"}, nil, "-"},
		{[]string{"a", "b", "c"}, []string{"a", "B", "c"}, " -+ "},
		{[]string{"a", "b"}, []string{"a", "b"}, "  "},
		{[]string{"a", "x", "b", "y", "c"}, []string{"a", "b", "c"}, " - - "},
	}
	for _, c := range cases {
		if got := signs(lineDiff(c.a, c.b)); got != c.want {
			t.Errorf("lineDiff(%q, %q) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
	if d := trimContext(lineDiff([]string{"a"}, []string{"a"}), 2); d != nil {
		t.Error("no change should give no lines")
	}
}

// TestSplitNumbers: in the split /diff view an unchanged line after an
// insertion keeps its old number on the left (the live test showed 12, not 8).
func TestSplitNumbers(t *testing.T) {
	ctx := &dline{sign: ' ', oldNo: 8, newNo: 12}
	if sideNo(ctx, true) != 8 || sideNo(ctx, false) != 12 {
		t.Errorf("context numbers: %d / %d", sideNo(ctx, true), sideNo(ctx, false))
	}
	if sideNo(&dline{sign: '-', oldNo: 9}, false) != 9 || sideNo(&dline{sign: '+', newNo: 13}, true) != 13 {
		t.Error("changed lines keep their own number")
	}
}
