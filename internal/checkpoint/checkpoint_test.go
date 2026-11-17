package checkpoint

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotRestore(t *testing.T) {
	dir := t.TempDir()
	if _, err := git(context.Background(), dir, nil, "init", "-q"); err != nil {
		t.Skip("git not available:", err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return "<missing>"
		}
		return string(b)
	}
	write("keep.txt", "v1")
	write("gone.txt", "will be deleted")

	r := Open(dir)
	if r == nil {
		t.Fatal("Open returned nil inside a repo")
	}
	p, err := r.Snapshot("before turn")
	if err != nil {
		t.Fatal(err)
	}

	write("keep.txt", "v2")
	write("new.txt", "created by agent")
	_ = os.Remove(filepath.Join(dir, "gone.txt"))

	if n, _ := r.Changed(p); n != 3 {
		t.Fatalf("Changed = %d, want 3", n)
	}
	if _, err := r.Restore(p); err != nil {
		t.Fatal(err)
	}
	if got := read("keep.txt"); got != "v1" {
		t.Errorf("keep.txt = %q", got)
	}
	if got := read("gone.txt"); got != "will be deleted" {
		t.Errorf("gone.txt = %q", got)
	}
	if got := read("new.txt"); got != "<missing>" {
		t.Errorf("new.txt should be removed, got %q", got)
	}
	// the user's own index must be untouched
	if out, _ := git(context.Background(), dir, nil, "diff", "--cached", "--name-only"); out != "" {
		t.Errorf("user index modified: %q", out)
	}
}

func TestCommit(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	if _, err := git(ctx, dir, nil, "init", "-q"); err != nil {
		t.Skip("git not available")
	}
	_, _ = git(ctx, dir, nil, "config", "user.email", "nox@example.invalid")
	_, _ = git(ctx, dir, nil, "config", "user.name", "nox test")
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("1"), 0o644)
	r := Open(dir)
	h, err := r.Commit("NØX: first")
	if err != nil || h == "" {
		t.Fatalf("commit: %q %v", h, err)
	}
	if h2, err := r.Commit("NØX: nothing"); err != nil || h2 != "" {
		t.Fatalf("empty commit should be skipped: %q %v", h2, err)
	}
}
