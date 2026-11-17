// Command nox-local is NØX for models running on this machine: it manages
// llama.cpp's llama-server and drives small models with a lean agent loop.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"text/tabwriter"

	tea "charm.land/bubbletea/v2"

	"github.com/Johnny008-creator/N0X-Local/internal/agent"
	"github.com/Johnny008-creator/N0X-Local/internal/agent/native"
	"github.com/Johnny008-creator/N0X-Local/internal/config"
	"github.com/Johnny008-creator/N0X-Local/internal/headless"
	"github.com/Johnny008-creator/N0X-Local/internal/local"
	"github.com/Johnny008-creator/N0X-Local/internal/mcpclient"
	"github.com/Johnny008-creator/N0X-Local/internal/paths"
	"github.com/Johnny008-creator/N0X-Local/internal/session"
	"github.com/Johnny008-creator/N0X-Local/internal/tui"
)

const version = "1.0.0-rc1"

var commit = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "models", "ls":
			listModels()
			return
		case "pull":
			pull(os.Args[2:])
			return
		case "help", "-h", "--help":
			usage()
			return
		}
	}
	cfg := config.Load()
	var (
		prompt  = flag.String("p", "", "run one prompt headless and print the answer (- reads stdin)")
		output  = flag.String("output", "text", "headless output: text | json")
		model   = flag.String("m", cfg.Model, "model id (nox-local models)")
		mode    = flag.String("mode", cfg.Mode, "manual | plan | auto | yolo")
		effort  = flag.String("effort", cfg.Effort, "low = answer directly (fast) · medium / high = think first")
		cont    = flag.Bool("c", false, "continue the most recent session in this directory")
		showVer = flag.Bool("version", false, "print version")
	)
	flag.Usage = usage
	flag.Parse()
	if *showVer {
		fmt.Println("NØX-Local", version, commit)
		return
	}
	md, ok := agent.ParseMode(*mode)
	if !ok {
		fail("unknown mode %q", *mode)
	}
	ef, ok := agent.ParseEffort(*effort)
	if !ok {
		fail("unknown effort %q", *effort)
	}
	if local.Binary() == "" {
		fail("llama-server not found · build llama.cpp with Vulkan (see docs/LOCAL.md) or set NOX_LLAMA_SERVER")
	}
	srv := local.NewServer()
	defer srv.Close()
	defer native.StopBackground() // servers the agent started with background: true
	prov := native.Local(srv, local.Base)
	dir, _ := os.Getwd()
	mcp := mcpclient.New(mcpclient.LoadProject(dir, cfg.MCP), paths.File("state", "mcp.json"))
	defer mcp.Close()
	prov.SetMCP(mcp)
	if !prov.Available() {
		fail("no model downloaded yet · try: nox-local pull qwen3.5-4b")
	}
	if *model == "" {
		*model = pickDefault()
	}
	if p, ok := local.Find(*model); !ok || !p.Installed() {
		fail("model %q is not downloaded · nox-local models lists what you have", *model)
	}
	if *prompt != "" {
		code := headless.Run(prov, agent.Request{Prompt: *prompt, Model: *model, Dir: dir, Mode: md, Effort: ef}, *output == "json")
		mcp.Close()
		srv.Close()
		native.StopBackground() // os.Exit skips the defers above
		srv.Close()
		os.Exit(code)
	}

	store := session.Open()
	var resume *session.Session
	if *cont {
		s, err := store.Latest(dir)
		if err != nil {
			fail("%v", err)
		}
		resume = s
	}
	m := tui.New(tui.Options{Providers: []agent.Provider{prov}, Active: prov.ID(), Model: *model,
		Mode: md, Effort: ef, MCP: mcp, Version: version, Dir: dir, Resume: resume,
		Store: store, Hooks: cfg.Hooks, AutoCommit: cfg.AutoCommit})
	if _, err := tea.NewProgram(m).Run(); err != nil {
		srv.Close()
		fail("%v", err)
	}
}

// pickDefault prefers FAST (Qwen 4B) or MEDIUM (Qwen 9B), then any downloaded model.
func pickDefault() string {
	for _, id := range []string{"qwen3.5-4b", "qwen3.5-9b", "qwen2.5-coder-14b", "qwen3.8-27b"} {
		if p, _ := local.Find(id); p.Installed() {
			return p.ID
		}
	}
	best := ""
	for _, p := range local.Catalog {
		if p.Installed() && (p.Tested || best == "") {
			best = p.ID
		}
	}
	return best
}

func listModels() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSIZE\tCONTEXT\tSTATUS\tNOTE")
	for _, p := range local.Catalog {
		st := "—"
		if p.Installed() {
			st = "downloaded"
		}
		if !p.Tested {
			p.Note += " (untested)"
		}
		fmt.Fprintf(w, "%s\t%.1f GB\t%dK\t%s\t%s\n", p.ID, p.SizeGB, p.Ctx/1024, st, p.Note)
	}
	w.Flush()
	fmt.Printf("\nmodels live in %s (set NOX_MODELS to change) · llama-server: %s\n", local.ModelDir(), orNone(local.Binary()))
}

func pull(args []string) {
	if len(args) != 1 {
		fail("usage: nox-local pull <model id> · nox-local models lists them")
	}
	p, ok := local.Find(args[0])
	if !ok {
		fail("unknown model %q · nox-local models lists them", args[0])
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := local.Pull(ctx, p, os.Stderr); err != nil {
		fail("%v", err)
	}
	fmt.Fprintf(os.Stderr, "ready · start it with: nox-local -m %s\n", p.ID)
}

func usage() {
	fmt.Fprint(os.Stderr, `nox-local — NØX for models on this machine (llama.cpp)

usage:
  nox-local                 start the TUI with the default model
  nox-local -m <id>         start with a specific model
  nox-local -p "prompt"     run one prompt and print the answer
  nox-local models          list models, sizes and what is downloaded
  nox-local pull <id>       download a model (resumes if interrupted)

flags:
`)
	flag.PrintDefaults()
}

func orNone(s string) string {
	if s == "" {
		return "not found"
	}
	return s
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "nox-local: "+format+"\n", a...)
	os.Exit(1)
}
