// Package hooks runs user commands on NØX events, for every provider.
//
//	[[hooks]]
//	event   = "stop"          # prompt | post_tool | stop
//	match   = "Edit|Write"    # optional regexp on the tool name (post_tool)
//	command = "go build ./..."
//
// A failing "prompt" hook blocks the prompt; other hooks only report.
// Commands get NOX_EVENT, NOX_TOOL, NOX_FILE, NOX_PROMPT, NOX_DIR in the env.
package hooks

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type Hook struct {
	Event   string `toml:"event"`
	Match   string `toml:"match"`
	Command string `toml:"command"`
	Timeout int    `toml:"timeout"` // seconds, default 60
}

type Result struct {
	Hook   Hook
	Output string
	Err    error
}

// Vars are passed to the command as NOX_* environment variables.
type Vars struct {
	Event, Tool, File, Prompt, Dir string
}

// Select returns the hooks for an event (and tool, for post_tool).
func Select(all []Hook, event, tool string) []Hook {
	var out []Hook
	for _, h := range all {
		if !strings.EqualFold(h.Event, event) || strings.TrimSpace(h.Command) == "" {
			continue
		}
		if h.Match != "" {
			re, err := regexp.Compile(h.Match)
			if err != nil || !re.MatchString(tool) {
				continue
			}
		}
		out = append(out, h)
	}
	return out
}

// Run executes hooks in order and stops at the first failure.
func Run(hs []Hook, v Vars) []Result {
	var res []Result
	for _, h := range hs {
		r := run(h, v)
		res = append(res, r)
		if r.Err != nil {
			break
		}
	}
	return res
}

func run(h Hook, v Vars) Result {
	t := h.Timeout
	if t <= 0 {
		t = 60
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(t)*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", h.Command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", h.Command)
	}
	cmd.Dir = v.Dir
	cmd.Env = append(os.Environ(), "NOX_EVENT="+v.Event, "NOX_TOOL="+v.Tool, "NOX_FILE="+v.File, "NOX_PROMPT="+v.Prompt, "NOX_DIR="+v.Dir)
	out, err := cmd.CombinedOutput()
	s := strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n"))
	if len(s) > 4000 {
		s = s[:4000] + "\n…"
	}
	return Result{Hook: h, Output: s, Err: err}
}
