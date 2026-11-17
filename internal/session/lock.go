package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Johnny008-creator/N0X-Local/internal/paths"
)

// A running NØX writes ~/.nox/state/run-<pid>.json naming its directory and
// session. A clean exit removes it; a file whose process is gone means the
// last run crashed or was killed, and its session can be offered again.
type runFile struct {
	PID     int       `json:"pid"`
	Dir     string    `json:"dir"`
	Session string    `json:"session,omitempty"`
	State   string    `json:"state,omitempty"` // idle | working | needs approval
	Title   string    `json:"title,omitempty"`
	Model   string    `json:"model,omitempty"`
	Updated time.Time `json:"updated"`
}

// Running is a NØX process seen through its run record (nox agents).
type Running = runFile

// Status updates this process's record: state, current title and model.
func Status(dir, sessionID, state, title, model string) {
	b, _ := json.Marshal(runFile{os.Getpid(), dir, sessionID, state, title, model, time.Now()})
	_ = os.WriteFile(runPath(os.Getpid()), b, 0o644)
}

// List returns every live NØX process.
func List() []Running {
	files, _ := filepath.Glob(filepath.Join(paths.Sub("state"), "run-*.json"))
	var out []Running
	for _, f := range files {
		var r runFile
		if b, err := os.ReadFile(f); err == nil && json.Unmarshal(b, &r) == nil && alive(r.PID) {
			out = append(out, r)
		}
	}
	return out
}

// Send drops a message into another NØX process's inbox.
func Send(pid int, from, text string) error {
	if !alive(pid) {
		return fmt.Errorf("no running NØX with pid %d", pid)
	}
	d := paths.Sub("state", "inbox-"+strconv.Itoa(pid))
	name := filepath.Join(d, time.Now().Format("20060102-150405.000000")+".json")
	b, _ := json.Marshal(map[string]string{"from": from, "text": text})
	return os.WriteFile(name, b, 0o644)
}

// Inbox returns and removes messages sent to this process.
func Inbox() []map[string]string {
	d := filepath.Join(paths.Home(), "state", "inbox-"+strconv.Itoa(os.Getpid()))
	files, _ := filepath.Glob(filepath.Join(d, "*.json"))
	var out []map[string]string
	for _, f := range files {
		var m map[string]string
		if b, err := os.ReadFile(f); err == nil && json.Unmarshal(b, &m) == nil {
			out = append(out, m)
		}
		_ = os.Remove(f)
	}
	return out
}

func runPath(pid int) string { return paths.File("state", "run-"+strconv.Itoa(pid)+".json") }

// MarkRunning records this process; call again when the session id is known.
func MarkRunning(dir, sessionID string) {
	prev := runFile{}
	if b, err := os.ReadFile(runPath(os.Getpid())); err == nil {
		_ = json.Unmarshal(b, &prev)
	}
	Status(dir, sessionID, prev.State, prev.Title, prev.Model)
}

// MarkStopped removes the record on a clean exit.
func MarkStopped() {
	_ = os.Remove(runPath(os.Getpid()))
	_ = os.RemoveAll(filepath.Join(paths.Home(), "state", "inbox-"+strconv.Itoa(os.Getpid())))
}

// Interrupted returns the session id of an earlier run in dir that did not
// exit cleanly ("" if none) and clears those stale records.
func Interrupted(dir string) string {
	files, _ := filepath.Glob(filepath.Join(paths.Sub("state"), "run-*.json"))
	found := ""
	for _, f := range files {
		var r runFile
		b, err := os.ReadFile(f)
		if err != nil || json.Unmarshal(b, &r) != nil {
			continue
		}
		if r.PID == os.Getpid() || alive(r.PID) {
			continue
		}
		_ = os.RemoveAll(filepath.Join(paths.Home(), "state", "inbox-"+strconv.Itoa(r.PID)))
		if !strings.EqualFold(r.Dir, dir) {
			continue // another project's crash: reported when NØX starts there
		}
		if r.Session != "" {
			found = r.Session
		}
		_ = os.Remove(f)
	}
	return found
}

func alive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}
