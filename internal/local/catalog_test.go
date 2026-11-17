package local

import (
	"strings"
	"testing"

	"github.com/Johnny008-creator/nox-local/internal/agent"
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

// Every profile still in the catalog must be one a user can actually pick,
// and Saluki's template needs an explicit effort or it thinks for minutes.
func TestSamplingEffort(t *testing.T) {
	var s Server
	kw := func(m map[string]any) map[string]any { v, _ := m["chat_template_kwargs"].(map[string]any); return v }
	if m := s.Sampling("qwen3.5-4b", agent.EffortHigh); m["temperature"] != 0.6 {
		t.Errorf("qwen high should use thinking sampling: %v", m)
	}
	if m := s.Sampling("qwen3.5-4b", agent.EffortLow); m["temperature"] != 0.7 {
		t.Errorf("qwen low should use fast sampling: %v", m)
	}
	if got := kw(s.Sampling("saluki-27b", agent.EffortHigh))["reasoning_effort"]; got != "low" {
		t.Errorf("saluki high reasoning_effort = %v, want low", got)
	}
	if got := kw(s.Sampling("saluki-27b", agent.EffortHigh))["enable_thinking"]; got != true {
		t.Errorf("saluki high enable_thinking = %v, want true", got)
	}
	if got := kw(s.Sampling("saluki-27b", agent.EffortLow))["enable_thinking"]; got != nil {
		t.Errorf("saluki low should leave thinking off via the request, got %v", got)
	}
}

// The five models the catalog is built around are the ones on the machine;
// a typo in an id would fall through to "model not downloaded".
func TestCatalogIDs(t *testing.T) {
	for _, id := range []string{"qwen3.5-4b", "qwen3.5-9b", "ornith-1.5-9b", "nemotron-bolt-e4b", "saluki-27b"} {
		if _, ok := Find(id); !ok {
			t.Errorf("catalog is missing %s", id)
		}
	}
	if len(Catalog) != 5 {
		t.Errorf("catalog has %d profiles, want 5", len(Catalog))
	}
}
