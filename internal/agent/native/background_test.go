//go:build !windows

package native

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Stop (context cancel) must end a command whose child keeps the output
// open, like a dev server started in the foreground.
func TestBashCancelKillsChildren(t *testing.T) {
	r := runner{dir: t.TempDir()}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(300 * time.Millisecond); cancel() }()
	start := time.Now()
	// The child is a copy of sleep under a unique name, so pgrep finds it
	// and nothing else.
	_, _ = r.bash(ctx, "cp /bin/sleep ./noxchild_zz && ./noxchild_zz 30 & sleep 0.1; ./noxchild_zz 30; wait", 60)
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("bash did not return after cancel: %v", d)
	}
	time.Sleep(100 * time.Millisecond)
	if out, _ := exec.Command("pgrep", "-x", "noxchild_zz").Output(); strings.TrimSpace(string(out)) != "" {
		t.Fatal("child process survived the cancel")
	}
}

func TestBackgroundServerReportsURL(t *testing.T) {
	r := runner{dir: t.TempDir()}
	out, err := r.background(context.Background(), "cp /bin/sleep ./noxbg_zz; echo 'Serving on http://0.0.0.0:8765/ now'; ./noxbg_zz 30")
	defer StopBackground()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Running in the background") || !strings.Contains(out, "URL: http://localhost:8765/") {
		t.Fatalf("output = %s", out)
	}
	StopBackground()
	time.Sleep(200 * time.Millisecond)
	if o, _ := exec.Command("pgrep", "-x", "noxbg_zz").Output(); len(strings.TrimSpace(string(o))) > 0 {
		t.Fatal("background command survived StopBackground")
	}
}

func TestBackgroundQuickCommandIsReportedLikeARun(t *testing.T) {
	r := runner{dir: t.TempDir()}
	out, err := r.background(context.Background(), "echo hi")
	if err != nil || !strings.Contains(out, "hi") || !strings.Contains(out, "finished right away") {
		t.Fatalf("out=%q err=%v", out, err)
	}
}
