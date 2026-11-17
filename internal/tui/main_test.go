package tui

import (
	"os"
	"testing"
)

// TestMain points HOME at a temporary directory so tests never write into
// the real ~/.nox (sessions, stats, audit log, caches).
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "nox-test-home")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
