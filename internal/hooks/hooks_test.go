package hooks

import "testing"

func TestSelectAndRun(t *testing.T) {
	all := []Hook{
		{Event: "post_tool", Match: "Edit|Write", Command: "echo edited $env:NOX_FILE"},
		{Event: "post_tool", Match: "Bash", Command: "echo bash"},
		{Event: "stop", Command: "exit 3"},
	}
	if n := len(Select(all, "post_tool", "Write")); n != 1 {
		t.Fatalf("post_tool/Write selected %d", n)
	}
	if n := len(Select(all, "prompt", "")); n != 0 {
		t.Fatalf("prompt selected %d", n)
	}
	res := Run(Select(all, "stop", ""), Vars{Event: "stop", Dir: t.TempDir()})
	if len(res) != 1 || res[0].Err == nil {
		t.Fatalf("failing stop hook should report an error: %+v", res)
	}
	ok := Run([]Hook{{Event: "stop", Command: "echo nox-hook"}}, Vars{Dir: t.TempDir()})
	if ok[0].Err != nil || ok[0].Output != "nox-hook" {
		t.Fatalf("got %+v", ok[0])
	}
}
