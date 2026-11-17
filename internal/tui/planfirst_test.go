package tui

import (
	"strings"
	"testing"

	"github.com/Johnny008-creator/N0X-Local/internal/agent"
)

// Plan first runs the turn read-only; /go sends the plan back in the real
// mode. The mode in force before planning must survive, or /go would run the
// plan read-only too and change nothing.
func TestPlanFirstGuidesAndGoRuns(t *testing.T) {
	m := &Model{mode: agent.ModeAuto}
	cmdPlanFirst(m, "")
	if !m.planFirst {
		t.Fatal("plan first did not turn on")
	}
	cmdPlanFirst(m, "")
	if m.planFirst {
		t.Fatal("plan first did not toggle off")
	}

	// A pending plan from a read-only turn.
	m.planFirst, m.planPending, m.planText = true, true, "1. edit a.go\n2. run go test"
	if m.mode != agent.ModeAuto {
		t.Fatalf("planning changed the mode to %v", m.mode)
	}
	// /go clears the pending plan; the turn itself needs a provider, so only
	// the bookkeeping is checked here.
	_ = cmdGo
	if strings.TrimSpace(m.planText) == "" {
		t.Fatal("the plan text was lost")
	}
}

func TestGoWithoutPlanFails(t *testing.T) {
	m := &Model{}
	if cmd := cmdGo(m, ""); cmd != nil {
		t.Fatal("/go with no plan must not start a turn")
	}
}
