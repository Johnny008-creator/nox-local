# FAST-tier challengers for nox-local (RX 6500 XT 8 GB, Vulkan), state November 2026

Speed model used throughout: gen tok/s ≈ 100 / (GB of weights read per token from VRAM); CPU-resident bytes are read at ~40 GB/s (DDR4 practical). For MoE, "bytes read per token" = non-expert weights + active experts only. MoE kernels on llama.cpp rarely reach the dense roofline. My working assumption is that real throughput is about 50–70 % of the theoretical figure; this is an assumption, not a measurement. Baseline check: Qwen3.5-4B UD-Q4_K_XL is 2.9 GB, the formula predicts 34.5 tok/s and 37 tok/s was measured, so the formula holds for dense models on this card.

## Q1: Is there a model that beats Qwen3.5-4B on independent agentic/coding evals while staying ≥30 tok/s here?

### Takeaway
No model beats Qwen3.5-4B on *independent* evals at ≥30 tok/s. The only candidate that clearly beats it on paper is JetBrains Mellum2.1-12B-A2.5B, but all its scores are vendor-reported. It only runs at ≥30 tok/s here with some experts offloaded or with the MXFP4 quant at tiny context, and it is thinking-only. LFM2.5-8B-A1B is much faster but its own vendor's table shows it *below* Qwen3.5-4B on BFCL. Treat Mellum2.1 as a benchmark-it-yourself candidate, not a drop-in replacement.

### Cited Findings

**Mellum2.1-12B-A2.5B-Thinking (JetBrains)**
- Released October 8, 2026, Apache 2.0 — [runaihome](https://www.runaihome.com/blog/jetbrains-mellum-2-1-local-coding-llm-gpu-guide-2026/). Tech report (Mellum2) arXiv 2605.31268 — [HF card](https://huggingface.co/JetBrains/Mellum2.1-12B-A2.5B-Thinking)
- 12B total / 2.5B active. 28 layers, 64 experts with 8 active, GQA 32Q/4KV heads, 131K context, 1024-token sliding window on 3 of every 4 layers (so KV cache is small), vocab 98,304 — [HF card](https://huggingface.co/JetBrains/Mellum2.1-12B-A2.5B-Thinking)
- Vendor table: LCB v6 82.0 / SWE-bench Verified 47.0 / Terminal-Bench 2.1 17.4 / BFCL v4 62.3. The same table gives Qwen3.5-9B 75.4 / 50.0 / 21.7 / 58.5 and Gemma 4 E4B 69.4 / 23.0 / 3.4 / 52.5. Qwen3.5-4B is not in the table, and no tau2 is reported — [HF card](https://huggingface.co/JetBrains/Mellum2.1-12B-A2.5B-Thinking)
- WorkBench 44.6 (Mellum2 Thinking 45.1) and ToolHop 49.1, both vendor-reported — [HF MTP card](https://huggingface.co/JetBrains/Mellum2.1-12B-A2.5B-Thinking-MTP). A local-AI blog notes it "does not lead its group" on WorkBench/ToolHop — [bodegaone](https://www.bodegaone.ai/blog/run-mellum-2-1-locally) (the blog discloses it builds a competing local coding tool)
- Independent evals: none as of Oct 15, 2026. SWE-V and LCB are vendor-only, and no community tok/s figures exist — [runaihome](https://www.runaihome.com/blog/jetbrains-mellum-2-1-local-coding-llm-gpu-guide-2026/). My searches found no independent llama.cpp tool-call test and no r/LocalLLaMA benchmark threads — [search summary incl. diego-rodriguez vLLM write-up](https://www.diego-rodriguez.work/blog/mellum-2-1-2026-local-coding-agent-vllm), which says it did not verify tool calling itself
- GGUF (official JetBrains repo): MXFP4_MOE ~7.0 GB, Q4_K_M 7.52 GiB (8.07 GB; 88.0 % top-token match against full precision), Q6_K ~10.9 GB, Q8_0 ~12.9 GB — [runaihome](https://www.runaihome.com/blog/jetbrains-mellum-2-1-local-coding-llm-gpu-guide-2026/); Q4_K_M = 8,071,295,264 bytes — [search summary of GGUF card / modelfit](https://modelfit.io/blog/jetbrains-mellum2-1-12b-coding-model-mac/)
- On 8 GB VRAM: "MXFP4_MOE may load on 8GB but ~4K context or less", with 12 GB as the realistic minimum — [runaihome](https://www.runaihome.com/blog/jetbrains-mellum-2-1-local-coding-llm-gpu-guide-2026/) (this assumes no expert offload)
- llama.cpp: needs a build from June 5, 2026 or later (architecture PR #23966). The Metal NaN bug #30579 is open but does not affect Vulkan/CUDA. A Radeon 780M Vulkan warmup hang (#24664) was fixed in June 2026 — [runaihome](https://www.runaihome.com/blog/jetbrains-mellum-2-1-local-coding-llm-gpu-guide-2026/)
- Tool-call format: the vLLM path uses `--tool-call-parser hermes` and reasoning parser `qwen3` — [HF card](https://huggingface.co/JetBrains/Mellum2.1-12B-A2.5B-Thinking). In llama.cpp, tool calling requires `--jinja` templates — [bodegaone](https://www.bodegaone.ai/blog/run-mellum-2-1-locally). Hermes-style `<tool_call>` is a format llama.cpp's generic parser handles, but I found no confirmation that it works for this model specifically.
- A separate MTP variant exists: one extra block, 1.4–1.8x decode, ~90 % acceptance, and llama.cpp support via `--spec-type draft-mtp --spec-draft-n-max 1`. The official GGUF repo ships the MTP head as a draft file — [HF MTP card](https://huggingface.co/JetBrains/Mellum2.1-12B-A2.5B-Thinking-MTP)
- Only a Thinking variant is described, with no non-thinking instruct — [HF MTP card](https://huggingface.co/JetBrains/Mellum2.1-12B-A2.5B-Thinking-MTP). Community GGUFs named "Mellum2-12B-A2.5B-Instruct" exist, but they are the older Mellum2 — [HF SandLogic](https://huggingface.co/SandLogicTechnologies/Mellum2-12B-A2.5B-Instruct-GGUF)
- One community speed run: ~50 tok/s on a 16 GB Mac mini M4 (Q4_K_M, ~120 GB/s memory) — [search summary/modelfit](https://modelfit.io/blog/jetbrains-mellum2-1-12b-coding-model-mac/)

**LFM2.5-8B-A1B (Liquid AI)**
- Released May 28, 2026, reasoning-only (always emits a chain of thought), 128K context, 8B total with "A1B" active. License is "lfm1.0" (Liquid's own license, not Apache) — [Liquid blog](https://www.liquid.ai/blog/lfm2-5-8b-a1b), [HF GGUF](https://huggingface.co/LiquidAI/LFM2.5-8B-A1B-GGUF)
- Vendor table: BFCLv3 64.79, BFCLv4 49.73, τ² Telecom 88.07, IFBench 56.47. **The same table gives Qwen3.5-4B BFCLv3 71.06, BFCLv4 54.01 and τ² Telecom 87.72**, i.e. the vendor itself shows Qwen3.5-4B ahead on BFCL and tied on tau2. The chart and table disagree slightly (48.50 vs 49.73 BFCLv4). No coding benchmarks (LCB/HumanEval) are published — [Liquid blog](https://www.liquid.ai/blog/lfm2-5-8b-a1b). τ² Retail is 39.82 — [search summary of Liquid table](https://www.liquid.ai/en/blog/lfm2-5-8b-a1b)
- GGUF sizes: Q4_0 4.84 GB, Q4_K_M 5.16 GB, Q5_K_M 6.03 GB, Q6_K 6.96 GB, Q8_0 9.01 GB — [HF GGUF](https://huggingface.co/LiquidAI/LFM2.5-8B-A1B-GGUF)
- Day-one llama.cpp support. Speed: 146 tok/s on Ryzen AI Max+ 395, 253 on M5 Max — [Liquid blog](https://www.liquid.ai/blog/lfm2-5-8b-a1b)
- Tool calls are Pythonic calls between special tokens (vLLM `pythonic` parser, SGLang `lfm2` parser). The tokenizer was updated to fix tool calling in llama.cpp, so re-download older files. I found no independent tool-call evals — [search summary of Liquid/Together pages](https://www.together.ai/models/lfm2-5-8b-a1b)

**Other names checked**
- Granite 4.1 (Apr 29, 2026) is **dense only** at 3B/8B/30B, with BFCL v3 60.8 / 68.3 / 73.7. There is no small 4.1 MoE. The MoE line is Granite 4.0 (H-Small 32B-A9B, H-Tiny 7B-A1B) — [NYU Shanghai RITS summary](https://rits.shanghai.nyu.edu/ai/ibm-releases-granite-4-1-dense-8b-matches-prior-32b-moe-flagship/) (secondary source). Granite-4.0-H-Tiny scores τ² Telecom 16.67 and BFCLv4 28.52 in Liquid's table — [Liquid blog](https://www.liquid.ai/blog/lfm2-5-8b-a1b)
- Ling-mini-2.0 is 16B total / 1.4B active — [HF](https://huggingface.co/inclusionAI/Ling-mini-2.0). A self-published Mac mini benchmark shows ~50 tok/s at IQ4_NL/Q4_K_S, with a low composite quality score (24–26) and tool calling inferred rather than measured — [HF SUMMARY.md](https://huggingface.co/Manojb/macmini-16gb-bench-gguf-mlx/raw/main/SUMMARY.md)
- Trinity Nano (6B/1B active) is mentioned in Raschka's survey, but I found no coding/tool scores or GGUF data — [Raschka](https://magazine.sebastianraschka.com/p/a-dream-of-spring-for-open-weight)
- Nemotron 3 Nano 4B (Mar 2026) is a hybrid Mamba-Transformer with Q4_K_M ~2.8 GB, the same speed class as Qwen3.5-4B, and no BFCL/tau2 published — [NVIDIA HF blog](https://huggingface.co/blog/nvidia/nemotron-3-nano-4b). Nemotron 3 Nano 30B-A3B is ~23 GB at Q4_K_M and out of budget — [localmodel.run](https://localmodel.run/model/nemotron-3-nano-30b-a3b)
- gpt-oss-20b (21B/3.6B): Liquid's table gives BFCLv4 49.88 and τ² Telecom 57.24, below Qwen3.5-4B — [Liquid blog](https://www.liquid.ai/blog/lfm2-5-8b-a1b). It is ~12.8 GB, so not a fast-tier option.
- Cohere North Mini Code (30B-A3B, June 2026) scores AA Coding Index 33.4, but the size is out of budget for the fast tier — [Artificial Analysis](https://artificialanalysis.ai/articles/north-mini-code-cohere-s-small-coding-focused-moe-model)
- I found no small Qwen MoE (<30B total) in the Qwen3.5/3.6 lineups. Qwen3.5 sizes are 0.8B, 2B, 4B, 9B, 27B, 35B-A3B, 122B-A10B and 397B-A17B — [Unsloth Qwen3.6 docs (MTP GGUF list)](https://unsloth.ai/docs/models/qwen3.6)
- Independent tool-call harness: Qwen3.5-4B placed 1st of 13 local models (97.5 %, 39/40) and beat GLM-4.7-Flash, Nemotron 3 Nano 30B and Mistral Small 3.2. Its one failure was a parallel multi-tool call. This harness did not test LFM2.5, Mellum2.1 or Qwen3.5-9B — [jdhodges](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)

### Inferences
- **Mellum2.1 speed estimate (my math, not measured).** Active ≈ 2.5B params × ~4.85 bits (Q4_K_M) / 8 ≈ 1.5 GB per token.
  - Fully on GPU: 100 / 1.5 ≈ 66 tok/s theoretical, ~35–45 realistic.
  - Fully on GPU is impossible at Q4_K_M: 8.07 GB of weights alone exceeds the ~7.2 GB usable VRAM.
  - With `--n-cpu-moe` offloading ~6 of 28 layers' experts (~1.5–1.7 GB to RAM): CPU reads ~6/28 of the active expert bytes (~0.18 GB) at 40 GB/s = ~4.5 ms, and the GPU reads ~1.3 GB at 100 GB/s = ~13 ms. That totals ~17.5 ms, or ~57 tok/s theoretical, ~30–40 realistic.
  - MXFP4_MOE (~7.0 GB) with fewer offloaded layers is the alternative.
  - Optional MTP gives ×1.4–1.8 if it works on Vulkan, which is unverified.
- **Thinking-only matters for "fast".** Mellum2.1 and LFM2.5 both always reason before answering. Qwen3.5-4B runs with thinking off in nox. If Mellum emits a few hundred reasoning tokens per turn, 35 tok/s raw can feel *slower* per tool step than Qwen3.5-4B at 37 tok/s with no thinking.
- **Quality vs Qwen3.5-4B.** On JetBrains' own table Mellum2.1 is roughly Qwen3.5-9B-class: better LCB (+6.6) and BFCL v4 (+3.8), slightly worse SWE-V (−3) and TB2.1 (−4.3). That is well above Qwen3.5-4B (vendor BFCL v4 50.3, LCB 55.8). Harnesses differ, though: JetBrains reports Qwen3.5-9B LCB 75.4, while Qwen's own card says 65.6. Cross-table comparisons are therefore loose.
- **LFM2.5-8B-A1B estimate.** Active ~1–1.5B × 4.85/8 ≈ 0.6–0.9 GB per token, fully in VRAM (5.16 GB file + small KV). That gives ~110–165 tok/s theoretical and plausibly 60–90 tok/s real. It is very fast but not better: it loses on BFCL in its own vendor's table, has no coding numbers, is reasoning-only and uses a non-Apache license. It could be a "draft/utility" model (summaries, commit messages, routing), not a coding fast tier.
- Ling-mini-2.0 (16B, ~9+ GB at Q4) does not fit in VRAM. It has no tool evals and a weak community quality score, so I reject it.

### Gaps
- No independent (non-vendor) coding or tool-call eval exists for Mellum2.1 or LFM2.5-8B-A1B. BFCL leaderboard, SWE-rebench and Aider entries for them were not found.
- I could not confirm that llama.cpp's server parses Mellum2.1's hermes-style tool calls correctly. The tool calls might also end up inside `<think>` blocks, as with Qwen3.5 bug #20837.
- Exact active parameter count of LFM2.5-8B-A1B: the name says A1B, but the page I fetched gave no precise number.
- No tau2 figure for Mellum2.1.
- No measured Vulkan/RDNA2 tok/s exists for any MoE candidate. The ×0.5–0.7 MoE efficiency factor is an assumption.
- Trinity Nano, SmallThinker, ERNIE small MoE: no usable data found. Other researchers may cover them.

## Q2: Is Qwen3.5-9B (~16–18 tok/s est.) worth being the fast tier, or is a small MoE a better fast+good tradeoff?

### Takeaway
Plain Qwen3.5-9B (~17 tok/s) is too slow for a fast tier. It becomes competitive only if MTP works on Vulkan for it (~25–30 tok/s est.). The best fast+good tradeoff to *test* is: (a) Qwen3.5-4B + MTP (and/or ngram-mod), which keeps proven tool calling and gains speed, and (b) Mellum2.1 (MXFP4 or Q4_K_M with light expert offload) as the quality challenger. Keep Qwen3.5-4B as default until a local A/B shows Mellum2.1 wins on nox's own tool loop.

### Cited Findings
- Qwen3.5-9B vs 4B (Qwen vendor numbers): BFCL-V4 66.1 vs 50.3, LCB v6 65.6 vs 55.8, TAU2 79.1 vs 79.9 — [HF Qwen3.5-9B](https://huggingface.co/Qwen/Qwen3.5-9B), [HF Qwen3.5-4B](https://huggingface.co/Qwen/Qwen3.5-4B) (from the prior nox report)
- Unsloth publishes MTP GGUFs for the Qwen3.5 family including 4B and 9B, and reports `--spec-draft-n-max 2` as best in most setups — [Unsloth Qwen3.6 docs](https://unsloth.ai/docs/models/qwen3.6) (via search summary)
- llama.cpp MTP landed as PR #22673 (spring 2026). Typical acceptance for Qwen3.5/3.6 MTP is 70–85 %. Qwen3.6-27B on an RTX 3090 went 38→65 tok/s (1.71x), and Qwen3.5-27B on an M4 Pro went 15.3→23.3 (1.5x) — [Substack](https://johnpaulwile.substack.com/p/multi-token-prediction-mtp-in-llamacpp); a ~1.85x claim — [note.com](https://note.com/hacklog_stealth/n/n43be4a291d09?hl=en-US)
- **Vulkan risk:** issue #23774 reports AMD/RADV MTP generation dropping 22–33 % from b9265 to b9354, recovering partly by b9544. Other users saw larger regressions on later builds (b9592+) on Gemma 4. The issue was auto-closed "not planned" on Sep 5, 2026 without a confirmed fix. Some users pin b9570 — [llama.cpp #23774](https://github.com/ggml-org/llama.cpp/issues/23774)
- I found no Vulkan or 4B/9B-specific MTP speedup measurement — [search summary](https://github.com/ggml-org/llama.cpp/issues/23774)
- MTP flag spelling differs across sources (`--spec-type mtp` vs `draft-mtp`). JetBrains and the official docs list `draft-mtp` — [llama.cpp speculative.md](https://github.com/ggml-org/llama.cpp/blob/master/docs/speculative.md), [HF Mellum MTP](https://huggingface.co/JetBrains/Mellum2.1-12B-A2.5B-Thinking-MTP)
- Independent harness: Qwen3.5-9B is called "acceptable, faster" in one tool-call benchmark write-up, with 27B dense recommended where it fits — [openagent-bot](https://openagent-bot.pages.dev/blog/local-llm-tool-calling-benchmark-toolcall-15) (secondary source, not verified in depth)

### Inferences
- **Qwen3.5-9B UD-Q4_K_XL:** 5.96 GB → 100 / 5.96 ≈ 16.8 tok/s. With MTP at ×1.5–1.8 and the extra head read per step: ~24–30 tok/s *if* the Vulkan MTP path performs. That is borderline for the ≥30 target, and VRAM is already tight at 32K (MTP adds a layer plus its KV).
- **Qwen3.5-4B + MTP:** 37 × 1.5–1.8 ≈ 55–65 tok/s, with tool calling unchanged (output is mathematically identical with greedy verification). This is the cheapest speed win, but it must be benchmarked on the pinned build because of #23774.
- **Ranking for the fast tier** (quality at ≥30 tok/s, my judgement):
  1. Qwen3.5-4B (+MTP/ngram): proven, independently tested tool calling.
  2. Mellum2.1-12B-A2.5B: likely higher quality, unproven tool parsing, thinking overhead.
  3. Qwen3.5-9B+MTP: higher quality, speed at risk.
  4. LFM2.5-8B-A1B: fastest, not better.
- **Possible restructure:** if Mellum2.1 tests well, it could replace Qwen3.5-9B as the *MEDIUM* tier rather than FAST. It is ~9B-class on vendor evals at roughly 2x the 9B's speed.

### Gaps
- No measured MTP acceptance rate or speed for Qwen3.5-4B/9B on RDNA2 Vulkan.
- Unknown whether MTP works with Qwen3.5's hybrid DeltaNet layers in the current build for the small sizes. Unsloth says the GGUFs exist; I saw no 4B run reports.

## Q3: Independent tool-call tests (community harnesses, BFCL, tau2) and speed tricks

### Takeaway
Independent tool-call evidence exists only for Qwen3.5-4B (1st/13, 97.5 %) and older models. For Mellum2.1 and LFM2.5 everything is vendor-reported. On the speed side, `ngram-mod` speculative decoding is a free, low-risk addition for code-editing loops. MTP is the bigger win but is Vulkan-risky.

### Cited Findings
- jdhodges harness (LM Studio, 40 cases): Qwen3.5-4B 97.5 %, Nemotron 3 Nano 4B 95 %, GLM-4.7-Flash 95 % — [jdhodges](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)
- A "Part 2" with LFM2.5/Mellum/9B was not found — [search, no result](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)
- Community report: Qwen3.5 think tags leak into text in some servers. Unsloth GGUFs are recommended, and LM Studio parsed Qwen3.5 XML tool calls more reliably than raw llama.cpp at the time — [reddit snapshot](https://reddit.sentinel-team.org/posts/1sdhvc5/snapshots/2026-04-09T16%3A11%3A00.655487Z) (via search summary)
- llama.cpp speculative types (official doc):
  - `ngram-mod` (defaults n-match 24, n-min 48, n-max 64; shared ~16 MB hash pool; recommended for iterating over code/text and reasoning repetition; dense models can use lower n-min/n-max; MoE needs long drafts)
  - `ngram-simple` (recommended for source-code rewriting, e.g. `--spec-draft-n-max 64`)
  - `ngram-map-k`, `ngram-cache`
  - `draft-mtp` (uses the model's own MTP heads)
  - `draft-simple`, `draft-eagle3`, `draft-dflash`, `draft-dspark` (DSpark only for Qwen3-backbone drafts)
  - `--spec-default` enables ngram-mod
  - There is no spec type literally named "lookup"
  - Source: [llama.cpp speculative.md](https://github.com/ggml-org/llama.cpp/blob/master/docs/speculative.md)
- Mellum2.1 MTP: ~90 % acceptance, 1.4–1.8x — [HF MTP card](https://huggingface.co/JetBrains/Mellum2.1-12B-A2.5B-Thinking-MTP) (vendor)

### Inferences
- For nox's FAST tier, `--spec-type ngram-mod` (or `ngram-simple` for edit/rewrite turns) costs ~16 MB and is lossless. It speeds up the common agent pattern of re-emitting file content or diffs, so it is a safe default to A/B.
- Suggested A/B on this machine, all on the same pinned llama.cpp build:
  1. Qwen3.5-4B baseline
  2. Qwen3.5-4B + `draft-mtp --spec-draft-n-max 2`
  3. Qwen3.5-4B + `ngram-mod`
  4. Mellum2.1 MXFP4_MOE / Q4_K_M with `--n-cpu-moe` sweep 0–8, ± MTP draft
  5. Qwen3.5-9B + MTP
  Measure tok/s, *time per tool step* (including reasoning tokens) and the tool-call parse rate on nox's own task set.

### Gaps
- BFCL official leaderboard, SWE-rebench and the Aider leaderboard were not checked directly in this pass (tool-call budget). I could not confirm whether any of these candidates are listed there.
- No independent tau2 results for any candidate other than vendor tables.
