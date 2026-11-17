package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Johnny008-creator/nox-local/internal/agent"
	"github.com/Johnny008-creator/nox-local/internal/paths"
)

// Port is where nox-local runs llama-server (off 8080 to avoid clashes).
const Port = 8089

// Base is the OpenAI-compatible endpoint of the managed server.
var Base = "http://127.0.0.1:" + strconv.Itoa(Port) + "/v1"

// Server starts llama-server for one model at a time and stops it on Close.
type Server struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	exit  chan error
	model string
	ctx   int
}

func NewServer() *Server { return &Server{} }

// Binary finds llama-server: $NOX_LLAMA_SERVER, PATH, ~/.nox-local/llama.cpp,
// or the build the cloud NØX set up in ~/.nox/llama.cpp.
func Binary() string {
	if b := os.Getenv("NOX_LLAMA_SERVER"); b != "" {
		return b
	}
	if b, err := exec.LookPath("llama-server"); err == nil {
		return b
	}
	h, _ := os.UserHomeDir()
	for _, root := range []string{paths.Home(), filepath.Join(h, ".nox")} {
		b := filepath.Join(root, "llama.cpp", "build", "bin", "llama-server")
		if _, err := os.Stat(b); err == nil {
			return b
		}
	}
	return ""
}

// Ready reports whether a server binary and at least one model exist.
func (s *Server) Ready() bool {
	if Binary() == "" {
		return false
	}
	for _, p := range Catalog {
		if p.Installed() {
			return true
		}
	}
	return false
}

// Installed lists downloaded models for the picker.
func (s *Server) Installed() []agent.ModelInfo {
	var out []agent.ModelInfo
	for _, p := range Catalog {
		if p.Installed() {
			note := fmt.Sprintf("%.1f GB · %s", p.SizeGB, p.Note)
			if !p.Tested {
				note += " · untested"
			}
			out = append(out, agent.ModelInfo{ID: p.ID, Name: p.Name, Context: p.Ctx, Note: note})
		}
	}
	return out
}

// Sampling is the profile's recommended sampling for a request. Effort high
// also raises gpt-oss's own reasoning_effort.
func (s *Server) Sampling(model string, effort agent.Effort) map[string]any {
	p, ok := Find(model)
	if !ok {
		return nil
	}
	return p.Sampling(effort != agent.EffortLow)
}

// Ensure makes sure llama-server is serving model and returns its context
// size. Switching models restarts the server; loading can take a while, so
// progress goes to status.
func (s *Server) Ensure(ctx context.Context, model string, status func(string)) (int, error) {
	p, ok := Find(model)
	if !ok {
		return 0, fmt.Errorf("unknown local model %q · nox-local models lists them", model)
	}
	if !p.Installed() {
		return 0, fmt.Errorf("%s is not downloaded · run: nox-local pull %s", p.ID, p.ID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil && s.model == p.ID && healthy() {
		return s.ctx, nil
	}
	if s.cmd == nil && healthy() { // someone else's server on our port
		if n := serverCtx(); n > 0 {
			return n, nil
		}
	}
	s.stopLocked()
	bin := Binary()
	if bin == "" {
		return 0, errors.New("llama-server not found · build llama.cpp with Vulkan or set NOX_LLAMA_SERVER")
	}
	args := []string{"-m", p.Path(), "--host", "127.0.0.1", "--port", strconv.Itoa(Port),
		"--jinja", "-c", strconv.Itoa(p.Ctx), "-fa", "on", "-ctk", "q8_0", "-ctv", "q8_0",
		"--no-webui", "-np", "1",
		// Templates that support it (Qwen3.8) would otherwise keep every
		// past turn's thinking in the history: the prompt grows with each
		// step, and on a partly offloaded model prompt processing dominates.
		"--no-reasoning-preserve"}
	if p.MoE {
		// Experts that --fit leaves in system RAM must not be mmapped: the
		// page cache evicts them under memory pressure and generation fell
		// from 18 to 3 tok/s (gpt-oss-20b). Loading them outright fixes it.
		args = append(args, "--load-mode", "none")
	}
	if !slices.Contains(p.Args, "--spec-type") {
		// Draftless n-gram speculation: free (no draft model, no VRAM) and
		// it reuses text already in the context. Agents rewrite code they
		// just read, so edits ran 2–4x faster (gpt-oss-20b 18 → 38 tok/s,
		// Mellum 49 → 186); plain generation didn't get slower.
		args = append(args, "--spec-type", "ngram-mod")
	}
	args = append(args, p.Args...)
	logPath := paths.File("local", "llama-server.log")
	logf, err := os.Create(logPath)
	if err != nil {
		return 0, err
	}
	cmd := exec.Command(bin, args...)
	cmd.Env = serverEnv(os.Environ())
	cmd.Stdout, cmd.Stderr = logf, logf
	detach(cmd)
	status(fmt.Sprintf("loading %s (%.1f GB)…", p.Name, p.SizeGB))
	if err := cmd.Start(); err != nil {
		logf.Close()
		return 0, err
	}
	exit := make(chan error, 1)
	go func() { exit <- cmd.Wait(); logf.Close() }()
	s.cmd, s.exit, s.model = cmd, exit, p.ID

	tick := time.NewTicker(300 * time.Millisecond)
	defer tick.Stop()
	deadline := time.After(5 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			s.stopLocked()
			return 0, ctx.Err()
		case err := <-exit:
			s.cmd = nil
			return 0, fmt.Errorf("llama-server exited (%v) · see %s", err, logPath)
		case <-deadline:
			s.stopLocked()
			return 0, fmt.Errorf("llama-server did not become ready in 5 minutes · see %s", logPath)
		case <-tick.C:
			if healthy() {
				s.ctx = serverCtx()
				if s.ctx == 0 {
					s.ctx = p.Ctx
				}
				return s.ctx, nil
			}
		}
	}
}

// Close stops the server if nox-local started it.
func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *Server) stopLocked() {
	if s.cmd == nil {
		return
	}
	_ = s.cmd.Process.Signal(os.Interrupt)
	select {
	case <-s.exit:
	case <-time.After(5 * time.Second):
		_ = s.cmd.Process.Kill()
		<-s.exit
	}
	s.cmd, s.model = nil, ""
}

func healthy() bool {
	c := http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get("http://127.0.0.1:" + strconv.Itoa(Port) + "/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

// serverCtx is the context size the running server was launched with.
func serverCtx() int {
	c := http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get("http://127.0.0.1:" + strconv.Itoa(Port) + "/props")
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	var v struct {
		Gen struct {
			NCtx int `json:"n_ctx"`
		} `json:"default_generation_settings"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&v)
	return v.Gen.NCtx
}

// serverEnv keeps Mesa's RADV driver from spilling GPU buffers into system
// RAM (GTT). On a PCIe x4 card that spill made a 4B model 3x slower at
// generation and 8x slower at prompt processing; without it the buffers stay
// in VRAM. Flags the user already set in RADV_PERFTEST are kept.
func serverEnv(env []string) []string {
	const key = "RADV_PERFTEST="
	for i, kv := range env {
		if strings.HasPrefix(kv, key) {
			if !slices.Contains(strings.Split(kv[len(key):], ","), "nogttspill") {
				env[i] = kv + ",nogttspill"
			}
			return env
		}
	}
	return append(env, key+"nogttspill")
}
