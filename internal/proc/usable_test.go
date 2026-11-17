package proc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUsableRejectsEmptyAndDangling(t *testing.T) {
	d := t.TempDir()
	empty := filepath.Join(d, "empty")
	real := filepath.Join(d, "real")
	_ = os.WriteFile(empty, nil, 0o755)
	_ = os.WriteFile(real, []byte("#!/bin/sh\n"), 0o755)
	if usable(empty) {
		t.Error("empty file counted as usable")
	}
	if !usable(real) {
		t.Error("real file rejected")
	}
	link := filepath.Join(d, "link")
	if os.Symlink(filepath.Join(d, "missing"), link) == nil && usable(link) {
		t.Error("dangling symlink counted as usable")
	}
}
