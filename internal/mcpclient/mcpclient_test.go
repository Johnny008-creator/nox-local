package mcpclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type echoIn struct {
	Text string `json:"text"`
}

// TestMain doubles as a tiny MCP server when re-executed by the test.
func TestMain(m *testing.M) {
	if os.Getenv("NOX_MCP_HELPER") == "1" {
		s := mcp.NewServer(&mcp.Implementation{Name: "helper", Version: "1"}, nil)
		mcp.AddTool(s, &mcp.Tool{Name: "echo", Description: "echo text back", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
			func(_ context.Context, _ *mcp.CallToolRequest, in echoIn) (*mcp.CallToolResult, any, error) {
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "echo: " + in.Text}}}, nil, nil
			})
		_ = s.Run(context.Background(), &mcp.StdioTransport{})
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestOptInServer(t *testing.T) {
	exe, _ := os.Executable()
	state := filepath.Join(t.TempDir(), "mcp.json")
	cfgs := map[string]ServerConfig{"helper": {Command: exe, Env: map[string]string{"NOX_MCP_HELPER": "1"}}}
	mgr := New(cfgs, state)
	defer mgr.Close()

	if n := len(mgr.Tools()); n != 0 {
		t.Fatalf("a server must not connect before it is enabled, got %d tools", n)
	}
	if err := mgr.SetEnabled("helper", true); err != nil {
		t.Fatal(err)
	}
	tools := mgr.Tools()
	if len(tools) != 1 || tools[0].FuncName() != "mcp__helper__echo" || !tools[0].ReadOnly {
		t.Fatalf("tools = %+v, servers = %+v", tools, mgr.Servers())
	}
	got, ok := mgr.Lookup("mcp__helper__echo")
	if !ok {
		t.Fatal("lookup failed")
	}
	out, isErr, err := mgr.Call(context.Background(), got, map[string]any{"text": "hi"})
	if err != nil || isErr || out != "echo: hi" {
		t.Fatalf("call = %q %v %v", out, isErr, err)
	}

	// the choice survives a restart, and the server reconnects by itself
	again := New(cfgs, state)
	defer again.Close()
	if s := again.Servers(); len(s) != 1 || !s[0].Enabled {
		t.Fatalf("enabled state not remembered: %+v", s)
	}
	again.WaitStartup(context.Background())
	if n := len(again.Tools()); n != 1 {
		t.Fatalf("after restart: %d tools", n)
	}

	if err := mgr.SetEnabled("helper", false); err != nil {
		t.Fatal(err)
	}
	if n := len(mgr.Tools()); n != 0 {
		t.Fatalf("disabled server still offers %d tools", n)
	}
}

func TestBadServerReportsError(t *testing.T) {
	mgr := New(map[string]ServerConfig{"nope": {Command: "definitely-not-a-real-binary-nox"}}, filepath.Join(t.TempDir(), "s.json"))
	if err := mgr.SetEnabled("nope", true); err == nil {
		t.Fatal("expected an error")
	}
	if s := mgr.Servers(); len(s) != 1 || s[0].Err == nil || s[0].Connected {
		t.Fatalf("servers = %+v", s)
	}
}
