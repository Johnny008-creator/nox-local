package agent

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Johnny008-creator/N0X-Local/internal/paths"
)

// Cross-session memory: short facts NØX should keep in mind next time, per
// project (~/.nox-local/memory/<project>.md) or global (…/memory/global.md).
// Added with /remember or by starting a message with "#".

func memoryDir() string { return paths.Sub("memory") }

// MemoryFile is the facts file for dir, or the global one when dir is "".
func MemoryFile(dir string) string {
	if dir == "" {
		return filepath.Join(memoryDir(), "global.md")
	}
	sum := sha1.Sum([]byte(strings.ToLower(filepath.Clean(dir))))
	return filepath.Join(memoryDir(), filepath.Base(dir)+"-"+hex.EncodeToString(sum[:4])+".md")
}

// Remember appends a fact.
func Remember(dir, fact string) error {
	fact = strings.Join(strings.Fields(fact), " ")
	if fact == "" {
		return fmt.Errorf("nothing to remember")
	}
	f, err := os.OpenFile(MemoryFile(dir), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "- %s  <!-- %s -->\n", fact, time.Now().Format("2006-01-02"))
	return err
}

// Facts lists remembered facts (without the date comment).
func Facts(dir string) []string {
	b, err := os.ReadFile(MemoryFile(dir))
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "- ") {
			continue
		}
		if i := strings.Index(l, "  <!--"); i > 0 {
			l = l[:i]
		}
		out = append(out, strings.TrimPrefix(l, "- "))
	}
	return out
}

// Forget removes the n-th fact (0-based).
func Forget(dir string, n int) error {
	p := MemoryFile(dir)
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	var keep []string
	i := 0
	for _, l := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "- ") {
			if i == n {
				i++
				continue
			}
			i++
		}
		keep = append(keep, l)
	}
	return os.WriteFile(p, []byte(strings.Join(keep, "\n")+"\n"), 0o644)
}

func factsSection(dir string) string {
	g, p := Facts(""), Facts(dir)
	if len(g)+len(p) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n# Remembered from earlier sessions\n")
	for _, f := range g {
		b.WriteString("- " + f + "\n")
	}
	for _, f := range p {
		b.WriteString("- " + f + " (this project)\n")
	}
	return b.String()
}
