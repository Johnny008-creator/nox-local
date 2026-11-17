// Package local runs GGUF models on this machine through llama.cpp's
// llama-server: a curated catalog with launch flags tuned for small GPUs,
// downloads from Hugging Face, and the server's lifecycle.
package local

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Profile is one model with the llama-server settings that make it fit an
// 8 GB GPU next to 16 GB of RAM.
type Profile struct {
	ID   string
	Name string
	Repo string // Hugging Face repo
	File string // GGUF path in the repo
	// Local is the file name in ModelDir when it differs from the repo's
	// (MTP repos reuse the plain file names; some files sit in subfolders).
	Local  string
	SizeGB float64
	MoE    bool // mixture of experts: experts may live in system RAM
	Ctx    int  // context window to launch with
	// Args are extra llama-server flags, for the rare model that needs them.
	Args []string
	// Sampling per mode, as recommended by the model authors.
	Think, NoThink map[string]any
	Tested         bool   // measured on the reference machine
	Note           string // one line for the picker
}

var qwenThink = map[string]any{"temperature": 0.6, "top_p": 0.95, "top_k": 20, "min_p": 0.0}
var qwenFast = map[string]any{"temperature": 0.7, "top_p": 0.8, "top_k": 20, "min_p": 0.0}
var nemotron = map[string]any{"temperature": 0.6, "top_p": 0.95}

// noSpec turns speculative decoding off. Measured with Qwen3.5-4B on the
// RX 6500 XT (llama-server, 64k context, 2026-11-09):
//
//	                   2.4k prompt     10k prompt     19k prompt
//	MTP + ngram        43 tok/s        –              11 tok/s, prompt 263 t/s
//	MTP                52              43             12, prompt 263
//	ngram              39              –              17, prompt 512
//	none               35              34             32, prompt 513
//
// MTP only wins on short prompts. Past ~10k tokens it collapses, halves
// prompt processing, and the server stays slow after that. An agent spends
// most of a session past 10k, so the Qwen3.5 profiles run without it.
var noSpec = []string{"--spec-type", "none"}

// Saluki's template defaults to reasoning_effort xhigh, which on a 27B at
// IQ2 means minutes of thinking per turn. low keeps it usable and still
// reasons; enable_thinking has to be repeated because a sampling entry
// replaces the whole chat_template_kwargs map.
var salukiThink = map[string]any{"temperature": 0.6, "top_p": 0.95, "top_k": 20, "min_p": 0.0,
	"chat_template_kwargs": map[string]any{"enable_thinking": true, "reasoning_effort": "low"}}

// Catalog is the set of models this machine is set up for. Each profile
// fixes only the context window and, where needed, extra llama-server flags:
// --fit (on by default) places the layers and expert offload to the free
// VRAM. Speeds are generation tok/s measured on the reference machine
// (RX 6500 XT 8 GB, PCIe 3.0 x4, 16 GB DDR4) through llama-server.
var Catalog = []Profile{
	{ID: "qwen3.5-4b", Name: "Qwen3.5 4B", Repo: "unsloth/Qwen3.5-4B-MTP-GGUF", File: "Qwen3.5-4B-UD-Q4_K_XL.gguf",
		Local: "Qwen3.5-4B-MTP-UD-Q4_K_XL.gguf", SizeGB: 2.99, Ctx: 65536, Think: qwenThink, NoThink: qwenFast,
		Args: noSpec, Tested: true, Note: "FAST · 3 GB · 32 tok/s, prompt 590 t/s · whole model on the GPU"},
	{ID: "qwen3.5-9b", Name: "Qwen3.5 9B", Repo: "unsloth/Qwen3.5-9B-MTP-GGUF", File: "Qwen3.5-9B-UD-Q4_K_XL.gguf",
		Local: "Qwen3.5-9B-MTP-UD-Q4_K_XL.gguf", SizeGB: 6.14, Ctx: 32768, Think: qwenThink, NoThink: qwenFast,
		Args: noSpec, Tested: true, Note: "BALANCED · 6 GB · 9 tok/s, prompt 380 t/s · better Czech, longer reasoning"},
	{ID: "ornith-1.5-9b", Name: "Ornith 1.5 9B", Repo: "leirbag99/Ornith-1.5-9B-IQ4_XS-NO-VISION-GGUF",
		File: "ornith-1.5-9b-iq4_xs-imat.gguf", Local: "Ornith-1.5-9B-IQ4_XS.gguf",
		SizeGB: 5.24, Ctx: 32768, Think: qwenThink, NoThink: qwenFast,
		Args: noSpec, Tested: true, Note: "BALANCED · 5 GB · 22 tok/s, prompt 425 t/s · Qwen3.5 fine-tune, IQ4_XS"},
	{ID: "nemotron-bolt-e4b", Name: "Nemotron 3.5 Bolt E4B", Repo: "unsloth/NVIDIA-Nemotron-3.5-Bolt-E4B-GGUF",
		File:   "NVIDIA-Nemotron-3.5-Bolt-E4B-UD-Q4_K_XL.gguf",
		SizeGB: 5.06, Ctx: 32768, Think: nemotron, NoThink: nemotron,
		Args: noSpec, Tested: true, Note: "QUICK PREFILL · 5 GB · 8 tok/s, prompt 655 t/s · NVIDIA on a Gemma 4 base"},
	{ID: "saluki-27b", Name: "Underdog Saluki 27B", Repo: "ConwayResearch/Underdog-Saluki-27B-1.0",
		File:   "Underdog-Saluki-27B-1.0-IQ2-mix.gguf",
		SizeGB: 7.90, Ctx: 16384, Think: salukiThink, NoThink: qwenFast,
		// The 7.9 GB file does not fit in 8 GB of VRAM. With mmap, the layers
		// --fit keeps in RAM get evicted from the page cache and the server
		// stops answering; loading them outright gave 3.2 tok/s instead.
		Args: append([]string{"--load-mode", "none"}, noSpec...), Tested: true, Note: "DEEP · 8 GB · 3 tok/s, prompt ~5 t/s · 27B at IQ2, the smartest that fits"},
}

// Find returns the profile with this id (case-insensitive).
func Find(id string) (Profile, bool) {
	for _, p := range Catalog {
		if strings.EqualFold(p.ID, id) {
			return p, true
		}
	}
	return Profile{}, false
}

// ModelDir is where GGUF files live: $NOX_MODELS or ~/models.
func ModelDir() string {
	if d := os.Getenv("NOX_MODELS"); d != "" {
		return d
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "models")
}

// Path is where this profile's GGUF file is (or will be) stored.
func (p Profile) Path() string {
	if p.Local != "" {
		return filepath.Join(ModelDir(), p.Local)
	}
	return filepath.Join(ModelDir(), filepath.Base(p.File))
}

// Installed reports whether the GGUF file is fully downloaded.
func (p Profile) Installed() bool {
	fi, err := os.Stat(p.Path())
	return err == nil && fi.Size() > int64(p.SizeGB*0.95e9)
}

// URL is the direct download link.
func (p Profile) URL() string {
	return fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", p.Repo, p.File)
}

// Sampling returns the request parameters for thinking or fast mode.
func (p Profile) Sampling(think bool) map[string]any {
	if think {
		return p.Think
	}
	return p.NoThink
}
