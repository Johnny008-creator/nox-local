package tui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Johnny008-creator/N0X-Local/internal/agent"
)

// Structured review: the reviewer never edits, and must back every finding
// with evidence, so one model cannot simply hallucinate bugs in another's
// code. The JSON answer is rendered as a findings list.

const reviewContract = `
Reply with ONLY a JSON object, no prose before or after:
{"approved": true|false,
 "summary": "one sentence verdict",
 "findings": [{"severity": "high|medium|low", "file": "path", "line": 0,
   "issue": "what is wrong", "evidence": "the code or output that proves it", "confidence": 0.0-1.0}]}
Only report issues you can prove from the code, a diff or a command's output. No style nitpicks unless they break conventions the project clearly follows.`

type finding struct {
	Severity   string  `json:"severity"`
	File       string  `json:"file"`
	Line       int     `json:"line"`
	Issue      string  `json:"issue"`
	Evidence   string  `json:"evidence"`
	Confidence float64 `json:"confidence"`
}

type reviewResult struct {
	Approved bool      `json:"approved"`
	Summary  string    `json:"summary"`
	Findings []finding `json:"findings"`
}

func parseReview(s string) (reviewResult, bool) {
	i, j := strings.Index(s, "{"), strings.LastIndex(s, "}")
	if i < 0 || j <= i {
		return reviewResult{}, false
	}
	var r reviewResult
	if json.Unmarshal([]byte(s[i:j+1]), &r) != nil {
		return reviewResult{}, false
	}
	return r, r.Summary != "" || r.Findings != nil || r.Approved
}

func formatReview(r reviewResult, by string) string {
	verdict := "✓ LGTM"
	if !r.Approved {
		verdict = "✗ NEEDS CHANGES"
	}
	rows := []string{fmt.Sprintf("review by %s · %s · %s", by, verdict, plural(len(r.Findings), "finding"))}
	if r.Summary != "" {
		rows = append(rows, "  "+r.Summary)
	}
	order := map[string]int{"high": 0, "medium": 1, "low": 2}
	for pass := 0; pass < 3; pass++ {
		for _, f := range r.Findings {
			if o, ok := order[strings.ToLower(f.Severity)]; (ok && o != pass) || (!ok && pass != 2) {
				continue
			}
			loc := f.File
			if f.Line > 0 {
				loc += fmt.Sprintf(":%d", f.Line)
			}
			conf := ""
			if f.Confidence > 0 {
				conf = fmt.Sprintf(" · %d%%", int(f.Confidence*100+0.5))
			}
			rows = append(rows, fmt.Sprintf("  [%s] %s%s\n      %s", strings.ToUpper(f.Severity), loc, conf, f.Issue))
			if ev := strings.TrimSpace(f.Evidence); ev != "" {
				rows = append(rows, "      evidence: "+truncRight(strings.Join(strings.Fields(ev), " "), 140))
			}
		}
	}
	return strings.Join(rows, "\n")
}

// onReviewDone renders the last reviewer answer as findings.
func (m *Model) onReviewDone() {
	if !m.reviewing || m.lastFailed {
		return
	}
	for i := len(m.entries) - 1; i >= 0 && m.entries[i].kind != kUser; i-- {
		if m.entries[i].kind == kText {
			if r, ok := parseReview(m.entries[i].text); ok {
				m.entries[i].text = formatReview(r, m.turnProv.Name())
				m.entries[i].cache = ""
				if !r.Approved {
					m.toast(fmt.Sprintf("review: %s", plural(len(r.Findings), "finding")))
				}
			}
			return
		}
	}
}

// reviewer picks the provider for a review: another API provider when one
// is set up (a different model family catches different bugs), then any
// other provider, else the active one in a fresh, read-only session.
func (m *Model) reviewer() agent.Provider {
	var other, self agent.Provider
	for _, p := range m.provs {
		if !p.Available() {
			continue
		}
		_, api := p.(keyed)
		switch {
		case p == m.turnProv:
			self = p
		case api:
			return p
		case other == nil:
			other = p
		}
	}
	if other != nil {
		return other
	}
	return self
}

// gitRef accepts commit ids and branch/tag names, never options.
var gitRef = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/~^@{}-]{0,200}$`)

func gitOut(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	return string(out), err
}

// cmdReview: /review [commit <sha> | branch <base>] reviews the uncommitted
// changes, one commit or a branch against its base.
func cmdReview(m *Model, arg string) tea.Cmd {
	if m.busy {
		m.errorf("wait for the current turn to finish first")
		return nil
	}
	f := strings.Fields(arg)
	var diff, what string
	var err error
	switch {
	case len(f) >= 2 && (f[0] == "commit" || f[0] == "branch") && !gitRef.MatchString(f[1]):
		m.errorf("review: " + f[1] + " is not a commit or branch name")
		return nil
	case len(f) >= 2 && f[0] == "commit":
		what = "commit " + f[1]
		diff, err = gitOut(m.dir, "show", "--stat", "--patch", f[1])
	case len(f) >= 2 && f[0] == "branch":
		what = "this branch against " + f[1]
		diff, err = gitOut(m.dir, "diff", f[1]+"...HEAD")
	default:
		what = "the uncommitted changes"
		diff, err = gitOut(m.dir, "diff", "HEAD")
		if st, _ := gitOut(m.dir, "status", "--short"); strings.Contains(st, "??") {
			diff += "\nUntracked files (read them if relevant):\n" + st
		}
	}
	if err != nil {
		if strings.Contains(diff, "ambiguous argument 'HEAD'") || strings.Contains(diff, "unknown revision") {
			m.info("nothing committed yet · /review compares with the last commit, so commit once first")
			return nil
		}
		m.errorf("review: " + strings.TrimSpace(diff))
		return nil
	}
	if strings.TrimSpace(diff) == "" {
		m.info("nothing to review: " + what + " is empty")
		return nil
	}
	if len(diff) > 60000 {
		diff = diff[:60000] + "\n… [diff truncated; read the files for the rest]"
	}
	r := m.reviewer()
	if r == nil {
		m.errorf("review needs a model · nox-local pull qwen3.5-4b, then /doctor")
		return nil
	}
	prompt := "CODE REVIEW of " + what + ". You are an independent reviewer: read-only, do not edit anything. Read surrounding code and run read-only checks (build, tests) where useful.\n\n<diff>\n" + diff + "\n</diff>\n" + reviewContract
	return m.startTurnOn(r, prompt, agent.ModePlan, true)
}
