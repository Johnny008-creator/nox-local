// Package trust decides whether a project's own NØX settings may run.
//
// A repository can ship files that make NØX execute commands as soon as it
// starts: .mcp.json (MCP servers), .nox/config.toml (hooks, MCP, providers,
// allow rules, verify commands) and .nox/agents. A freshly cloned repo must
// not get that for free, so these files are only used after the user ran
// /trust (or `nox trust`) in the project. The approval is tied to a hash
// of the files: when they change, NØX asks again.
package trust

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Johnny008-creator/nox-local/internal/paths"
)

var mu sync.Mutex

// Root is the project root: the git top level, else dir itself.
func Root(dir string) string {
	if out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output(); err == nil {
		if r := strings.TrimSpace(string(out)); r != "" {
			return r
		}
	}
	return dir
}

// Files lists the project files that can make NØX run things.
func Files(dir string) []string {
	var out []string
	for _, root := range uniq(dir, Root(dir)) {
		for _, f := range []string{".mcp.json", filepath.Join(".nox", "config.toml")} {
			if st, err := os.Stat(filepath.Join(root, f)); err == nil && !st.IsDir() {
				out = append(out, filepath.Join(root, f))
			}
		}
		agents, _ := filepath.Glob(filepath.Join(root, ".nox", "agents", "*.md"))
		out = append(out, agents...)
	}
	sort.Strings(out)
	return out
}

func uniq(a, b string) []string {
	if filepath.Clean(a) == filepath.Clean(b) {
		return []string{a}
	}
	return []string{a, b}
}

func hash(files []string) string {
	h := sha256.New()
	for _, f := range files {
		b, _ := os.ReadFile(f)
		h.Write([]byte(f + "\x00"))
		h.Write(b)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func load() map[string]string {
	m := map[string]string{}
	if b, err := os.ReadFile(paths.File("trust.json")); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

// Needed reports whether dir has project settings that are not trusted
// yet (false when there is nothing that could run).
func Needed(dir string) bool {
	files := Files(dir)
	if len(files) == 0 {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	return load()[Root(dir)] != hash(files)
}

// Grant trusts the project's current settings.
func Grant(dir string) error {
	mu.Lock()
	defer mu.Unlock()
	m := load()
	m[Root(dir)] = hash(Files(dir))
	b, _ := json.MarshalIndent(m, "", " ")
	return os.WriteFile(paths.File("trust.json"), b, 0o600)
}

// Revoke forgets the approval.
func Revoke(dir string) error {
	mu.Lock()
	defer mu.Unlock()
	m := load()
	delete(m, Root(dir))
	b, _ := json.MarshalIndent(m, "", " ")
	return os.WriteFile(paths.File("trust.json"), b, 0o600)
}
