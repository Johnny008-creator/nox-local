// Package headless runs one prompt without the TUI (nox -p).
package headless

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/Johnny008-creator/nox-local/internal/agent"
)

// Run streams the answer to stdout (tool activity to stderr), or every
// event as JSON lines with --output json.
func Run(p agent.Provider, req agent.Request, asJSON bool) int {
	if req.Prompt == "-" {
		b, _ := io.ReadAll(os.Stdin)
		req.Prompt = string(b)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ch, err := p.Run(ctx, req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nox:", err)
		return 1
	}
	code := 0
	for ev := range ch {
		if ev.Kind == agent.EvToolInput { // argument deltas: counted in the TUI, too chatty here
			continue
		}
		if asJSON {
			fmt.Println(agent.EventJSON(ev))
		}
		if ev.Kind == agent.EvApproval { // headless never blocks on a human
			allow := req.Mode == agent.ModeYolo || req.Mode == agent.ModeAuto
			if !asJSON {
				verdict := "denied"
				if allow {
					verdict = "allowed"
				}
				fmt.Fprintf(os.Stderr, "◇ %s %s (headless %s mode)\n", ev.Approval.Tool.Name, verdict, req.Mode)
			}
			ev.Approval.Reply <- allow
			continue
		}
		switch ev.Kind {
		case agent.EvText:
			if !asJSON {
				fmt.Print(ev.Text)
			}
		case agent.EvToolStart:
			if !asJSON {
				fmt.Fprintf(os.Stderr, "◆ %s\n", ev.Tool.Name)
			}
		case agent.EvDone:
			if ev.IsError {
				fmt.Fprintln(os.Stderr, "nox:", ev.Text)
				code = 1
			}
		case agent.EvError:
			fmt.Fprintln(os.Stderr, "nox:", ev.Err)
			code = 1
		}
	}
	if !asJSON {
		fmt.Println()
	}
	return code
}
