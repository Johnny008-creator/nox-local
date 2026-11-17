// Package paths locates NØX-Local's own files under ~/.nox-local, kept
// apart from the cloud NØX (~/.nox) so settings and sessions never mix.
package paths

import (
	"os"
	"path/filepath"
)

// Home is ~/.nox-local (created on demand).
func Home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		h = "."
	}
	d := filepath.Join(h, ".nox-local")
	_ = os.MkdirAll(d, 0o755)
	return d
}

// Sub returns ~/.nox-local/<parts...>, creating the directory.
func Sub(parts ...string) string {
	d := filepath.Join(append([]string{Home()}, parts...)...)
	_ = os.MkdirAll(d, 0o755)
	return d
}

// File returns ~/.nox-local/<parts...> without creating it (parent dir is created).
func File(parts ...string) string {
	p := filepath.Join(append([]string{Home()}, parts...)...)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	return p
}
