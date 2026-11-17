package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Johnny008-creator/N0X-Local/internal/agent"
)

// /goal keeps NØX working until a condition holds. "$ cmd" goals are checked
// by running the command (exit 0 = met); plain-language goals are judged by
// a reviewer in a read-only turn. Every round and the total time are capped.

type goalT struct {
	cond     string
	cmd      string // shell check; "" = judged by a model
	rounds   int
	max      int
	started  time.Time
	judging  bool // the current turn is the judge
	lastNote string
}

const goalMaxDur = 45 * time.Minute

type goalCheckMsg struct {
	ok  bool
	out string
}

func cmdGoal(m *Model, arg string) tea.Cmd {
	switch strings.TrimSpace(arg) {
	case "":
		if m.goal == nil {
			m.info("no goal · /goal <condition> or /goal $ <command that must succeed>")
		} else {
			m.info(fmt.Sprintf("goal: %s · round %d/%d", m.goal.cond, m.goal.rounds, m.goal.max))
		}
		return nil
	case "off", "stop", "clear":
		m.goal = nil
		m.info("goal cleared")
		return nil
	}
	if m.busy {
		m.errorf("wait for the current turn to finish first")
		return nil
	}
	g := &goalT{cond: arg, max: 8, started: time.Now()}
	if c, ok := strings.CutPrefix(arg, "$"); ok {
		g.cmd = strings.TrimSpace(c)
		g.cond = "`" + g.cmd + "` succeeds"
	}
	m.goal = g
	m.info("goal set: " + g.cond + " · NØX keeps going until it holds (max 8 rounds, /goal off stops)")
	return m.startTurn("Your goal: " + g.cond + ". Work until it is achieved, verifying as you go. When you believe it is met, say so and stop.")
}

// goalAfterTurn decides what happens after a turn while a goal is active.
func (m *Model) goalAfterTurn() tea.Cmd {
	g := m.goal
	if g == nil {
		return nil
	}
	if g.judging {
		g.judging = false
		verdict := m.lastText()
		if strings.Contains(strings.ToUpper(verdict), "GOAL MET") && !strings.Contains(strings.ToUpper(verdict), "NOT MET") {
			return m.goalDone()
		}
		return m.goalContinue(verdict)
	}
	if m.lastFailed {
		m.info("goal paused: the last turn failed · /goal off or send a message to continue")
		m.goal = nil
		return nil
	}
	if g.cmd != "" {
		dir, cmdline := m.dir, g.cmd
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			var c *exec.Cmd
			if runtime.GOOS == "windows" {
				c = exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-Command", cmdline)
			} else {
				c = exec.CommandContext(ctx, "bash", "-lc", cmdline)
			}
			c.WaitDelay = 2 * time.Second
			c.Dir = dir
			out, err := c.CombinedOutput()
			return goalCheckMsg{err == nil, string(out)}
		}
	}
	r := m.reviewer()
	if r == nil {
		r = m.prov
	}
	g.judging = true
	return m.startTurnOn(r, "Judge, read-only: has this goal been fully achieved in the current project state?\nGOAL: "+g.cond+
		"\nInspect the files and run read-only checks as needed. End with exactly one line: GOAL MET or GOAL NOT MET: <what is missing>.", agent.ModePlan, true)
}

func (m *Model) onGoalCheck(msg goalCheckMsg) tea.Cmd {
	if m.goal == nil || m.busy {
		return nil // a turn you started meanwhile checks the goal when it ends
	}
	if msg.ok {
		return m.goalDone()
	}
	out := strings.TrimSpace(msg.out)
	if len(out) > 3000 {
		out = "…" + out[len(out)-3000:]
	}
	return m.goalContinue("`" + m.goal.cmd + "` still fails:\n" + out)
}

func (m *Model) goalDone() tea.Cmd {
	g := m.goal
	m.goal = nil
	m.info(fmt.Sprintf("✓ goal met: %s · %s · %s", g.cond, plural(g.rounds+1, "round"), fmtDur(time.Since(g.started).Truncate(time.Second))))
	m.toast("goal met: " + g.cond)
	return nil
}

func (m *Model) goalContinue(why string) tea.Cmd {
	g := m.goal
	if m.busy { // you sent something meanwhile: check again after that turn
		return nil
	}
	g.rounds++
	if g.rounds >= g.max || time.Since(g.started) > goalMaxDur {
		m.goal = nil
		m.errorf("goal not met after " + plural(g.rounds, "round") + " · stopping: " + truncRight(strings.Join(strings.Fields(why), " "), 200))
		return nil
	}
	m.info(fmt.Sprintf("goal not met yet · round %d/%d", g.rounds+1, g.max))
	return m.startTurnOn(m.prov, "The goal is not met yet.\n"+why+"\n\nKeep working toward the goal: "+g.cond, m.mode, false)
}

// lastText is the latest answer text of the current turn.
func (m *Model) lastText() string {
	for i := len(m.entries) - 1; i >= 0 && m.entries[i].kind != kUser; i-- {
		if m.entries[i].kind == kText {
			return m.entries[i].text
		}
	}
	return ""
}
