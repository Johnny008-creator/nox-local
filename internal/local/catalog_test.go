package local

import (
	"strings"
	"testing"

	"github.com/Johnny008-creator/N0X-Local/internal/agent"
)

func TestCatalogSane(t *testing.T) {
	seen, files := map[string]bool{}, map[string]string{}
	for _, p := range Catalog {
		if seen[p.ID] {
			t.Errorf("duplicate id %s", p.ID)
		}
		seen[p.ID] = true
		if !strings.HasSuffix(p.File, ".gguf") || p.Repo == "" || p.SizeGB <= 0 || p.Ctx < 4096 {
			t.Errorf("bad profile %+v", p)
		}
		// two profiles must never share a file in ModelDir (MTP repos reuse names)
		if other, dup := files[p.Path()]; dup {
			t.Errorf("%s and %s share %s", p.ID, other, p.Path())
		}
		files[p.Path()] = p.ID
		if p.Think == nil || p.NoThink == nil {
			t.Errorf("%s has no sampling", p.ID)
		}
	}
	if _, ok := Find("QWEN3.5-9B"); !ok {
		t.Error("Find is not case-insensitive")
	}
}

func TestServerEnvNoGTTSpill(t *testing.T) {
	got := serverEnv([]string{"HOME=/h"})
	if got[len(got)-1] != "RADV_PERFTEST=nogttspill" {
		t.Errorf("not added: %v", got)
	}
	got = serverEnv([]string{"RADV_PERFTEST=sam"})
	if got[0] != "RADV_PERFTEST=sam,nogttspill" {
		t.Errorf("not merged: %v", got)
	}
	got = serverEnv([]string{"RADV_PERFTEST=nogttspill"})
	if got[0] != "RADV_PERFTEST=nogttspill" {
		t.Errorf("duplicated: %v", got)
	}
}

// Thinking-only models must not be asked to skip thinking, or their
// reasoning lands in the answer text.
func TestMellumAlwaysThinks(t *testing.T) {
	p, _ := Find("mellum2.1-12b")
	for _, think := range []bool{false, true} {
		kw, _ := p.Sampling(think)["chat_template_kwargs"].(map[string]any)
		if kw["enable_thinking"] != true {
			t.Errorf("think=%v: enable_thinking = %v", think, kw["enable_thinking"])
		}
	}
}

func TestSamplingEffort(t *testing.T) {
	var s Server
	kw := func(m map[string]any) map[string]any { v, _ := m["chat_template_kwargs"].(map[string]any); return v }
	if got := kw(s.Sampling("gpt-oss-20b", agent.EffortLow))["reasoning_effort"]; got != "low" {
		t.Errorf("gpt-oss low = %v", got)
	}
	if got := kw(s.Sampling("gpt-oss-20b", agent.EffortHigh))["reasoning_effort"]; got != "high" {
		t.Errorf("gpt-oss high = %v", got)
	}
	if got := kw(s.Sampling("gpt-oss-20b", agent.EffortMedium))["reasoning_effort"]; got != "medium" {
		t.Errorf("gpt-oss medium = %v", got)
	}
	if m := s.Sampling("qwen3.5-4b", agent.EffortHigh); m["temperature"] != 0.6 {
		t.Errorf("qwen high should use thinking sampling: %v", m)
	}
	if got := kw(s.Sampling("qwen3.8-27b", agent.EffortLow))["enable_thinking"]; got != false {
		t.Errorf("qwen3.8-27b low enable_thinking = %v", got)
	}
	if got := kw(s.Sampling("qwen3.8-27b", agent.EffortMedium))["reasoning_effort"]; got != "low" {
		t.Errorf("qwen3.8-27b medium reasoning_effort = %v", got)
	}
	if got := kw(s.Sampling("qwen3.8-27b", agent.EffortHigh))["reasoning_effort"]; got != "high" {
		t.Errorf("qwen3.8-27b high reasoning_effort = %v", got)
	}
}
