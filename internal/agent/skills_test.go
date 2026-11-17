package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillsIndex(t *testing.T) {
	dir := t.TempDir()
	sd := filepath.Join(dir, ".nox", "skills", "deploy")
	if err := os.MkdirAll(sd, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(sd, "SKILL.md"), []byte("---\nname: deploy\ndescription: \"How to ship this app\"\nmetadata:\n  version: 1\n---\n# Deploy\n"), 0o644)
	var found *Skill
	for _, s := range Skills(dir) {
		if s.Name == "deploy" {
			s := s
			found = &s
		}
	}
	if found == nil || found.Desc != "How to ship this app" || found.Source != "project" {
		t.Fatalf("skill = %+v", found)
	}
	p := SystemPromptLean(dir)
	if !strings.Contains(p, "# Skills") || !strings.Contains(p, "deploy: How to ship this app") {
		t.Fatalf("prompt lacks skills index")
	}
	if !strings.Contains(p, "You are NØX") {
		t.Fatal("identity missing")
	}
}
