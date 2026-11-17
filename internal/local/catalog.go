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
var gemma = map[string]any{"temperature": 1.0, "top_p": 0.95, "top_k": 64, "min_p": 0.0}

// Mellum is thinking-only: told not to think (enable_thinking=false) it gets
// an empty <think></think> and then reasons anyway, straight into the
// answer. So it always runs with thinking on, and with the larger reply
// budget that thinking needs.
var mellum = map[string]any{"temperature": 0.6, "top_p": 0.95,
	"chat_template_kwargs": map[string]any{"enable_thinking": true}, "max_tokens": 8192}

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

// coder14bArgs: the 9 GB file does not fit 8 GB of VRAM, so every GPU layer
// counts. 16k context instead of 32k halves the KV cache, a 256 MiB --fit
// margin instead of 1 GiB leaves more room for layers (36 of 49 on the
// GPU), and loading without mmap keeps the CPU layers out of the page cache.
// Measured on a 5.6k prompt: generation 3.7 → 5.3 tok/s, prompt 110 → 151
// t/s. Speculation (ngram) made no difference.
var coder14bArgs = []string{"--load-mode", "none", "--fit-target", "256"}

// qwen27bArgs: the 10.9 GB file does not fit 8 GB of VRAM. Left to --fit,
// whole layers went to the CPU, and with them the recurrent (DeltaNet)
// blocks, which the CPU runs one token at a time on a single core: a 2.4k
// prompt did not finish in 4 minutes. Every layer now stays on the GPU and
// only the FFN weights of layers 24–64 (about 4 GB) live in RAM, where the
// CPU multiplies them on all threads. Measured: prompt 70 t/s, generation
// 3 tok/s. VRAM is nearly full, so --fit is off and nothing else may use
// the GPU. MTP is off: on Qwen3.5 it halved prompt speed.
var qwen27bArgs = []string{"--load-mode", "none", "-fit", "off", "-ngl", "99",
	"-ot", `blk\.(2[4-9]|[3-6][0-9])\.ffn_(up|down|gate)=CPU`, "--spec-type", "none"}

// Qwen3.8 27B sampling: reasoning_effort low at medium effort, so it does
// not freeze in 6-minute think loops.
var qwen27bThink = map[string]any{"temperature": 0.6, "top_p": 0.95, "top_k": 20, "min_p": 0.0,
	"chat_template_kwargs": map[string]any{"reasoning_effort": "low"}}
var qwen27bFast = map[string]any{"temperature": 0.7, "top_p": 0.8, "top_k": 20, "min_p": 0.0,
	"chat_template_kwargs": map[string]any{"enable_thinking": false}}

// gpt-oss reasons through its Harmony template: reasoning_effort, not
// enable_thinking, sets how long it thinks.
func gptOSS(effort string) map[string]any {
	return map[string]any{"temperature": 1.0, "top_p": 1.0,
		"chat_template_kwargs": map[string]any{"reasoning_effort": effort}}
}

// Catalog holds the four tiers (FAST, MEDIUM, HIGH, ULTRA) plus fallbacks,
// picked in docs/research (second round). No profile pins -ngl or
// --n-cpu-moe: llama-server's --fit (on by default) sizes GPU layers and
// expert offload to the free VRAM. Only the context size is fixed.
// Speeds in the notes are generation tok/s measured on the reference machine
// (RX 6500 XT 8 GB, PCIe 3.0 x4, 16 GB DDR4) through llama-server.
var Catalog = []Profile{
	{ID: "qwen3.5-4b", Name: "Qwen3.5 4B", Repo: "unsloth/Qwen3.5-4B-MTP-GGUF", File: "Qwen3.5-4B-UD-Q4_K_XL.gguf",
		Local: "Qwen3.5-4B-MTP-UD-Q4_K_XL.gguf", SizeGB: 2.99, Ctx: 65536, Think: qwenThink, NoThink: qwenFast,
		Args: noSpec, Tested: true, Note: "FAST · 35 tok/s, 32 at 19k context · whole model on the GPU"},
	{ID: "qwen3.5-9b", Name: "Qwen3.5 9B", Repo: "unsloth/Qwen3.5-9B-MTP-GGUF", File: "Qwen3.5-9B-UD-Q4_K_XL.gguf",
		Local: "Qwen3.5-9B-MTP-UD-Q4_K_XL.gguf", SizeGB: 6.14, Ctx: 32768, Think: qwenThink, NoThink: qwenFast,
		Args: noSpec, Tested: true, Note: "MEDIUM · dense, whole model on the GPU"},
	{ID: "qwen2.5-coder-14b", Name: "Qwen2.5-Coder 14B", Repo: "unsloth/Qwen2.5-Coder-14B-Instruct-GGUF", File: "Qwen2.5-Coder-14B-Instruct-Q4_K_M.gguf",
		SizeGB: 8.99, Ctx: 16384, Think: qwenThink, NoThink: qwenFast,
		Args: coder14bArgs, Tested: true, Note: "HIGH · 5 tok/s, partly on the CPU · coding specialist"},
	{ID: "qwen3.8-27b", Name: "Qwen3.8 27B", Repo: "unsloth/Qwen3.8-27B-GGUF", File: "Qwen3.8-27B-UD-IQ3_XXS.gguf",
		SizeGB: 10.93, Ctx: 16384, Think: qwen27bThink, NoThink: qwen27bFast,
		Args: qwen27bArgs, Tested: true, Note: "ULTRA · 3 tok/s, prompt 70 t/s · slow, deep reasoning only"},
	{ID: "mellum2.1-12b", Name: "Mellum2.1 12B-A2.5B", Repo: "JetBrains/Mellum2.1-12B-A2.5B-Thinking-GGUF",
		File: "mtp/Mellum2.1-12B-A2.5B-Thinking-MTP-MXFP4_MOE.gguf", Local: "Mellum2.1-12B-A2.5B-Thinking-MTP-MXFP4_MOE.gguf",
		SizeGB: 7.27, MoE: true, Ctx: 32768, Think: mellum, NoThink: mellum,
		Note: "legacy · always thinks"},
	{ID: "gpt-oss-20b", Name: "gpt-oss 20B", Repo: "ggml-org/gpt-oss-20b-GGUF", File: "gpt-oss-20b-MXFP4.gguf",
		SizeGB: 12.11, MoE: true, Ctx: 65536, Think: gptOSS("medium"), NoThink: gptOSS("low"),
		Note: "legacy · MoE, experts in RAM"},
	{ID: "qwen3.6-35b-a3b", Name: "Qwen3.6 35B-A3B", Repo: "unsloth/Qwen3.6-35B-A3B-GGUF", File: "Qwen3.6-35B-A3B-UD-IQ3_XXS.gguf",
		SizeGB: 13.21, MoE: true, Ctx: 49152, Think: qwenThink, NoThink: qwenFast,
		Note: "legacy · MoE, experts in RAM"},
	{ID: "gemma-4-e4b", Name: "Gemma 4 E4B", Repo: "unsloth/gemma-4-E4B-it-GGUF", File: "gemma-4-E4B-it-UD-Q4_K_XL.gguf",
		SizeGB: 5.12, Ctx: 32768, Think: gemma, NoThink: gemma,
		Note: "extra · images · weak at tools"},
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
