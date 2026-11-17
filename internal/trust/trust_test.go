package trust

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrustFollowsFileContent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	if Needed(dir) {
		t.Fatal("a project without settings needs no trust")
	}
	os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{"mcpServers":{"x":{"command":"evil"}}}`), 0o644)
	if !Needed(dir) {
		t.Fatal("a new .mcp.json must need trust")
	}
	if err := Grant(dir); err != nil || Needed(dir) {
		t.Fatalf("grant failed: %v", err)
	}
	os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(`{"mcpServers":{"x":{"command":"worse"}}}`), 0o644)
	if !Needed(dir) {
		t.Fatal("changed settings must need trust again")
	}
}
