package native

import (
	"strings"
	"testing"
)

func TestTextToolCallsJSON(t *testing.T) {
	calls, rest := textToolCalls("Let me look.\n<tool_call>\n{\"name\": \"Read\", \"arguments\": {\"file_path\": \"main.go\"}}\n</tool_call>")
	if len(calls) != 1 || calls[0].Function.Name != "Read" || !strings.Contains(calls[0].Function.Arguments, `"main.go"`) {
		t.Fatalf("calls: %+v", calls)
	}
	if rest != "Let me look." {
		t.Fatalf("rest %q", rest)
	}
}

func TestTextToolCallsXML(t *testing.T) {
	in := "<tool_call>\n<function=Bash>\n<parameter=command>\ngo test ./...\n</parameter>\n</function>\n</tool_call>"
	calls, rest := textToolCalls(in)
	if len(calls) != 1 || calls[0].Function.Name != "Bash" || calls[0].Function.Arguments != `{"command":"go test ./..."}` {
		t.Fatalf("calls: %+v", calls)
	}
	if rest != "" {
		t.Fatalf("rest %q", rest)
	}
}

func TestTextToolCallsPlainText(t *testing.T) {
	if calls, rest := textToolCalls("Just an answer, no tools."); len(calls) != 0 || rest != "Just an answer, no tools." {
		t.Fatalf("got %+v %q", calls, rest)
	}
}

func TestRepeatsCountsIdenticalCalls(t *testing.T) {
	var r repeats
	a := toolCall{}
	a.Function.Name, a.Function.Arguments = "Read", `{"file_path":"x"}`
	b := a
	b.Function.Arguments = `{"file_path":"y"}`
	if r.see(a) != 1 || r.see(a) != 2 || r.see(a) != 3 || r.see(b) != 1 || r.see(a) != 1 {
		t.Fatal("wrong repeat counting")
	}
}

func TestWithReadLimit(t *testing.T) {
	tc := toolCall{}
	tc.Function.Name, tc.Function.Arguments = "Read", `{"file_path":"a.go"}`
	if got := withReadLimit(tc).Function.Arguments; !strings.Contains(got, `"limit":300`) {
		t.Fatalf("no default limit: %s", got)
	}
	tc.Function.Arguments = `{"file_path":"a.go","limit":50}`
	if got := withReadLimit(tc).Function.Arguments; got != tc.Function.Arguments {
		t.Fatalf("explicit limit changed: %s", got)
	}
}

func TestClipTo(t *testing.T) {
	s := strings.Repeat("a", 20_000)
	out := clipTo(s, leanMaxOut)
	if len(out) > leanMaxOut+200 || !strings.Contains(out, "chars cut") {
		t.Fatalf("clip len %d", len(out))
	}
}

func TestResearchGate(t *testing.T) {
	call := func(name, args string) toolCall {
		tc := toolCall{}
		tc.Function.Name, tc.Function.Arguments = name, args
		return tc
	}
	art := call("Artifact", `{"title":"R","kind":"markdown","content":"# R\n\ntext"}`)
	g := researchGate{on: true}
	if msg := g.check(art); !strings.Contains(msg, "read 0 source") {
		t.Fatalf("first: %q", msg)
	}
	for i := 0; i < 3; i++ {
		g.seen(call("WebFetch", `{}`), "page")
	}
	g.seen(call("WebFetch", `{}`), "ERROR: 404")
	if g.reads != 3 {
		t.Fatalf("reads = %d", g.reads)
	}
	if msg := g.check(art); !strings.Contains(msg, "Sources") {
		t.Fatalf("second: %q", msg)
	}
	if msg := g.check(art); msg != "" {
		t.Fatalf("each check fires once, got %q", msg)
	}
	ok := researchGate{on: true, reads: 4}
	if msg := ok.check(call("Artifact", `{"content":"# R\n\n## 5. Sources\n1. https://a"}`)); msg != "" {
		t.Fatalf("complete report blocked: %q", msg)
	}
	if msg := (&researchGate{}).check(art); msg != "" {
		t.Fatal("gate must be off outside research")
	}
}
