// Package proc starts provider CLIs. On Windows npm installs .cmd shims;
// running those through cmd.exe mangles quotes, newlines and non-ASCII
// arguments, so we read the shim and run its real target directly.
package proc

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var shimTarget = regexp.MustCompile(`"%dp0%\\([^"]+\.(?:js|mjs|cjs|exe))"`)

// Resolve returns the program and leading arguments needed to run bin.
func Resolve(bin string) (prog string, lead []string, err error) {
	p, err := exec.LookPath(bin)
	if err != nil {
		return "", nil, err
	}
	if runtime.GOOS != "windows" {
		return p, nil, nil
	}
	ext := strings.ToLower(filepath.Ext(p))
	if ext != ".cmd" && ext != ".bat" {
		return p, nil, nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", nil, err
	}
	// The shim also mentions "%dp0%\node.exe" (the optional bundled node);
	// the real target is the one inside node_modules.
	var rel string
	for _, m := range shimTarget.FindAllSubmatch(b, -1) {
		if strings.Contains(strings.ToLower(string(m[1])), "node_modules") {
			rel = string(m[1])
		}
	}
	if rel == "" {
		return "", nil, errors.New(bin + ": unrecognised npm shim " + p)
	}
	target := filepath.Join(filepath.Dir(p), rel)
	if strings.EqualFold(filepath.Ext(target), ".exe") {
		return target, nil, nil
	}
	node := filepath.Join(filepath.Dir(p), "node.exe")
	if _, err := os.Stat(node); err != nil {
		if node, err = exec.LookPath("node"); err != nil {
			return "", nil, errors.New("node not found for " + bin)
		}
	}
	return node, []string{target}, nil
}

// Command builds an *exec.Cmd for bin with args, resolving shims.
func Command(ctx context.Context, bin string, args ...string) (*exec.Cmd, error) {
	prog, lead, err := Resolve(bin)
	if err != nil {
		return nil, err
	}
	return exec.CommandContext(ctx, prog, append(lead, args...)...), nil
}

// Available reports whether bin is on PATH and looks runnable.
func Available(bin string) bool {
	p, err := exec.LookPath(bin)
	if err != nil {
		return false
	}
	return usable(p)
}

// usable reports whether the file LookPath found can plausibly run.
// A failed install can leave a 0-byte file with the exec bit set,
// which LookPath happily accepts. Called on every picker render, so
// it must be cheap — no spawning the binary here.
func usable(path string) bool {
	// os.Stat follows symlinks, so a dangling link fails here too.
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir() && fi.Size() > 0
}
