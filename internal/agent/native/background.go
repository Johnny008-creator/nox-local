package native

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Background commands (servers, watchers) keep running after the tool call
// returns. They are killed by StopBackground when the app exits.
var (
	bgMu   sync.Mutex
	bgPIDs = map[int]*Job{} // pid → job
)

// Job is a background command the agent started (dev server, watcher).
type Job struct {
	PID     int
	Command string
	Log     string
	URL     string
	Started time.Time
}

// Jobs lists the background commands still running, oldest first.
func Jobs() []Job {
	bgMu.Lock()
	defer bgMu.Unlock()
	out := make([]Job, 0, len(bgPIDs))
	for _, j := range bgPIDs {
		out = append(out, *j)
	}
	sort.Slice(out, func(i, k int) bool { return out[i].Started.Before(out[k].Started) })
	return out
}

// StopJob kills one background command (its whole process group).
func StopJob(pid int) error {
	bgMu.Lock()
	_, ok := bgPIDs[pid]
	delete(bgPIDs, pid)
	bgMu.Unlock()
	if !ok {
		return fmt.Errorf("no background job with pid %d", pid)
	}
	return killGroup(pid)
}

// JobTail returns the last n lines of a job's log.
func JobTail(pid, n int) (string, error) {
	bgMu.Lock()
	j, ok := bgPIDs[pid]
	bgMu.Unlock()
	if !ok {
		return "", fmt.Errorf("no background job with pid %d", pid)
	}
	b, err := os.ReadFile(j.Log)
	if err != nil {
		return "", err
	}
	return tail(string(b), n), nil
}

// StopBackground kills every background command started this session.
func StopBackground() {
	bgMu.Lock()
	defer bgMu.Unlock()
	for pid := range bgPIDs {
		_ = killGroup(pid)
		delete(bgPIDs, pid)
	}
}

var localURL = regexp.MustCompile(`https?://(?:localhost|127\.0\.0\.1|0\.0\.0\.0|\[::\])(?::\d+)?[^\s"'<>]*`)

// bgWait is how long a background command gets to print its first lines
// (or fail) before the tool call returns.
const bgWait = 5 * time.Second

func (r runner) background(ctx context.Context, command string) (string, error) {
	logDir := filepath.Join(os.TempDir(), "nox-bg")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}
	logPath := filepath.Join(logDir, time.Now().Format("20060102-150405.000")+".log")
	logf, err := os.Create(logPath)
	if err != nil {
		return "", err
	}
	defer logf.Close()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.Command("bash", "-lc", command)
	}
	cmd.Dir = r.dir
	cmd.Stdout, cmd.Stderr = logf, logf
	ownGroup(cmd)
	cmd.Cancel = nil // not tied to the turn: it outlives it
	if err := cmd.Start(); err != nil {
		return "", err
	}
	pid := cmd.Process.Pid
	exited := make(chan error, 1)
	go func() {
		exited <- cmd.Wait()
		bgMu.Lock()
		delete(bgPIDs, pid)
		bgMu.Unlock()
	}()
	bgMu.Lock()
	bgPIDs[pid] = &Job{PID: pid, Command: command, Log: logPath, Started: time.Now()}
	bgMu.Unlock()

	select {
	case err := <-exited: // it finished (or failed) right away: report like a normal run
		out, _ := os.ReadFile(logPath)
		s := strings.TrimSpace(string(out))
		if err != nil {
			return clip(s + "\n[" + err.Error() + "]"), fmt.Errorf("the command exited at once (%v)", err)
		}
		if s == "" {
			s = "(no output)"
		}
		return clip(s + "\n[it finished right away, so it was not left running]"), nil
	case <-time.After(bgWait):
	case <-ctx.Done():
		_ = killGroup(pid)
		return "", ctx.Err()
	}
	out, _ := os.ReadFile(logPath)
	s := strings.TrimSpace(string(out))
	var b strings.Builder
	fmt.Fprintf(&b, "Running in the background (pid %d). It keeps running after this reply and stops when NØX closes.\n", pid)
	if u := localURL.FindString(s); u != "" {
		u = strings.Replace(u, "0.0.0.0", "localhost", 1)
		bgMu.Lock()
		if j := bgPIDs[pid]; j != nil {
			j.URL = u
		}
		bgMu.Unlock()
		fmt.Fprintf(&b, "URL: %s  ← tell the user to open this\n", u)
	}
	fmt.Fprintf(&b, "Log: %s (read it with: tail -n 50 %s)\nStop it with: kill -- -%d\n", logPath, logPath, pid)
	if s != "" {
		b.WriteString("First output:\n" + clip(s))
	}
	return b.String(), nil
}

// tail keeps the last n lines of a log.
func tail(s string, lines int) string {
	ls := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(ls) > lines {
		ls = append([]string{"…"}, ls[len(ls)-lines:]...)
	}
	return strings.Join(ls, "\n")
}
