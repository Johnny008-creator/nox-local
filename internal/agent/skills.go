package agent

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Johnny008-creator/nox-local/internal/paths"
)

// Skill is a folder with a SKILL.md (frontmatter: name, description) that
// tells the agent how to do one kind of task. NØX lists them in the system
// prompt; the model reads the file itself when a task matches.
type Skill struct {
	Name, Desc, Path, Source string
}

// SkillDirs are searched in order; the first skill with a name wins.
// ~/.claude/skills is shared so skills installed for Claude Code work here
// too.
func SkillDirs(dir string) [][2]string {
	h, _ := os.UserHomeDir()
	return [][2]string{
		{filepath.Join(dir, ".nox", "skills"), "project"},
		{paths.Sub("skills"), "nox-local"},
		{filepath.Join(dir, ".claude", "skills"), "project-claude"},
		{filepath.Join(h, ".claude", "skills"), "claude"},
	}
}

// Skills lists available skills, sorted by name.
func Skills(dir string) []Skill {
	seen := map[string]bool{}
	var out []Skill
	for _, d := range SkillDirs(dir) {
		files, _ := filepath.Glob(filepath.Join(d[0], "*", "SKILL.md"))
		for _, f := range files {
			s, ok := readSkill(f)
			if !ok || seen[s.Name] {
				continue
			}
			s.Source = d[1]
			seen[s.Name] = true
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func readSkill(path string) (Skill, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Skill{}, false
	}
	defer f.Close()
	s := Skill{Path: path, Name: filepath.Base(filepath.Dir(path))}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	if !sc.Scan() || strings.TrimSpace(sc.Text()) != "---" {
		return s, true // no frontmatter: still usable, named after its folder
	}
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok || strings.HasPrefix(line, " ") {
			continue
		}
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		switch strings.TrimSpace(k) {
		case "name":
			if v != "" {
				s.Name = v
			}
		case "description":
			s.Desc = v
		}
	}
	return s, true
}

// skillDesc is how much of each description goes into the prompt. Small
// models have 32–64K windows, so the index is a compact menu: enough to spot
// a match, the details are in SKILL.md.
const skillDesc = 110

// skillIndex is the prompt section listing skills.
func skillIndex(dir string) string {
	var b strings.Builder
	for _, s := range Skills(dir) {
		d := strings.Join(strings.Fields(s.Desc), " ")
		if r := []rune(d); len(r) > skillDesc {
			d = string(r[:skillDesc-1]) + "…"
		}
		b.WriteString("- " + s.Name + ": " + d + " (" + filepath.ToSlash(s.Path) + ")\n")
	}
	if b.Len() == 0 {
		return ""
	}
	return "\n\n# Skills\nIf the task matches a skill below, Read its SKILL.md first and follow it.\n" + b.String()
}
