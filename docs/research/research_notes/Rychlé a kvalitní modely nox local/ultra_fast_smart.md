# ULTRA tier for nox-local: smarter than HIGH, still ≥ ~10 tok/s (state: 7 Nov 2026)

Hardware model used for all estimates (from the earlier report): GPU ~100 GB/s effective, DDR4 ~40–45 GB/s effective, PCIe 3.0 x4, budget ~17–18 GB for weights+KV+buffers, single model resident. Decode time per token ≈ (bytes read from VRAM / 100 GB/s) + (bytes read from RAM / ~42 GB/s), and real llama.cpp reaches roughly 35–60 % of that ceiling. All tok/s figures for this machine are **estimates**. None were measured.

## Q1: Are there larger MoE models (Oct–Nov 2026) that fit ~17 GB and beat Qwen3.6-35B-A3B?

### Takeaway
Only one new candidate fits. **Recursive Flint 40B-A2.5B** came out on 21 Oct 2026, its UD-IQ3_XXS file is 14.5 GB, and it has fewer active parameters than HIGH. Its vendor numbers put it modestly above Qwen3.6-35B-A3B (TB2.1 54.9, SWE-V 74.2) but below Qwen3.8-27B. It should run at about HIGH speed (est. 12–22 tok/s). Every score is vendor-only, and llama.cpp support is one day old with known problems. I found no Qwen3.8 small MoE, no new GLM Air/Flash, and no Nemotron 3.5 MoE that fits.

### Cited Findings
**Recursive Flint (40B-A2.5B)**
- Released 21 Oct 2026 under Apache 2.0. It is Recursive's first model, and the weights are on HF — [MarkTechPost (title/date)](https://www.marktechpost.com/2026/10/21/recursive-flint-local-coding-model/), [orcarouter](https://www.orcarouter.ai/blog/flint-40b-a2-5b-local-coding)
- Architecture: 39.8B total, ~2.5B active per token (active body ~2.2B), 49 layers (2 dense + 47 MoE), 256 routed experts with top-6 plus 1 shared expert. 35 sliding-window layers (window 513) and 14 full-attention layers, 32 Q heads / **4 KV heads**, head_dim 128, documented context 262,144 tokens — [HF RecursiveSI/flint](https://huggingface.co/RecursiveSI/flint)
- Vendor benchmarks: **Terminal-Bench 2.1 54.9 %** (Terminus 2 harness; solutions found via web retrieval count as failures), **SWE-bench Verified 74.2 %** (mini-swe-agent), "SWE-bench Pro v2" 89.7 %. Settings: temp 1.0, top-p 0.95, top-k 20, avg pass@1 over ≥5 runs. The card recommends ≥32K output budget for hard problems — [HF RecursiveSI/flint](https://huggingface.co/RecursiveSI/flint)
- "SWE-bench Pro v2" is **not** the SWE-bench Pro that other labs publish, so the 89.7 cannot be compared with Qwen3.8-27B's 61.7. Flint is **not in the Artificial Analysis index**, and no third party has re-run its scores. Recursive's own claim is narrower: it leads MoE models **under 50B total**. **Qwen3.8-27B scores above Flint on a six-benchmark mean** — [orcarouter (search summary of multilingual versions)](https://www.orcarouter.ai/it/blog/recursive-flint-v1-0-release)
- Recursive says Flint "may need more guidance than the larger closed-source coding models" — [MiraFlow/aggregators via search](https://miraflow.ai/blog/recursive-flint-explained-laptop-coding-model-moe-2026), [orcarouter](https://www.orcarouter.ai/blog/flint-40b-a2-5b-local-coding)
- Vendor speed: >125 tok/s on an M4 Pro MacBook Pro and >150 tok/s on a 64 GB RTX Spark (short context). Recursive claims "up to 2× comparable models, >5× dense 25–40B" — [orcarouter](https://www.orcarouter.ai/blog/flint-40b-a2-5b-local-coding)
- Unsloth GGUF sizes: UD-IQ1_M 11.4 GB, **UD-IQ2_XXS 11.9 GB**, **UD-Q2_K_XL 13.6 GB**, **UD-IQ3_XXS 14.5 GB**, UD-Q3_K_XL 18.5 GB, UD-Q4_K_XL 25.1 GB — [HF unsloth/flint-GGUF](https://huggingface.co/unsloth/flint-GGUF)
- llama.cpp: PR #30928 "models: Add Flint model (FlintForCausalLM)" was **merged 21 Oct 2026** (commit a37cc46). Tests passed on CPU/CUDA/Metal, and **Vulkan is not mentioned**. Expert-activation auto-scaling (an F16 overflow workaround) **slows inference 10–30 %**, and ggerganov called it a "workaround until we have a better fix". Linked bug **#31000**: turns where tool-call XML ends up inside reasoning and finishes with `stop` without dispatching the tool. Root cause not isolated — [llama.cpp PR #30928](https://github.com/ggml-org/llama.cpp/pull/30928)
- Third-party GGUF notes: the shipped chat template uses an escape that llama.cpp's Jinja engine cannot parse, so `--jinja` fails on it. The model has no BOS token — [search summary of terelabs/flint-GGUF](https://huggingface.co/terelabs/flint-GGUF). One community quant reports ~96–100 tok/s generation at IQ4 on unknown hardware — [romanerm/flint-GGUF](https://huggingface.co/romanerm/flint-GGUF)
- Not on SWE-rebench (window 15 May–1 Jul 2026). That window lists Qwen3.6-27B 31.2 % ± 1.68 and **Qwen3.6-35B-A3B 24.7 % ± 0.79**. Qwen3.8-27B, Flint, Nemotron and Laguna are absent — [swe-rebench.com](https://swe-rebench.com/)

**NVIDIA Nemotron 3.5 Lightning 30B-A3B (GA 11 Aug 2026), rejected**
- Hybrid Mamba-2 + MoE + Attention (`nemotron_h_moe`), 30B/3B active, has MTP layers, NVFP4 pre-training recipe (not described as QAT) — [HF unsloth/NVIDIA-Nemotron-3.5-Lightning-30B-A3B-GGUF](https://huggingface.co/unsloth/NVIDIA-Nemotron-3.5-Lightning-30B-A3B-GGUF)
- On NVIDIA's own table it is **weaker than HIGH**: SWE-V 51.56 vs Qwen3.6-35B-A3B 70.12, TB2.1 24.58 vs 44.38 — same source
- GGUFs are large because quants barely shrink: UD-IQ2_XXS 19.4 GB, UD-IQ3_XXS 19.8 GB, so it does not fit — same source

**Nemotron 3.5 Bolt 74B-A9B**: the Unsloth GGUF still exists ([unsloth listing](https://huggingface.co/models?author=unsloth&sort=created&search=GGUF)). It was rejected in the earlier report at 22 GB, and nothing has changed.

**Qwen 3.8 family**: the HF Qwen org lists only Qwen3.8-27B (dense), Qwen3.8-Flash-Next (~180B) and Qwen3.8-2.4T-A95B. There is **no small Qwen3.8 MoE**, and the newest A3B is Qwen-AgentWorld-35B-A3B (Jun 2026) — [HF Qwen models](https://huggingface.co/models?author=Qwen&sort=created)

**GLM**: searches surface only GLM-5 / 5.1 / 5.2 (hundreds of GB, e.g. GLM-5.1 2-bit needs a 24 GB GPU + 256 GB RAM). I found no GLM Air/Flash successor to GLM-4.7-Flash — [dev.to GLM-5.1](https://dev.to/jovan_chan_9500711396d4e6/glm-51-review-2026-mit-744b-moe-that-tops-swe-bench-pro-3bfl), [DataCamp GLM-5.2](https://www.datacamp.com/tutorial/how-to-run-glm-5-2-locally)

**Others named in Recursive's chart** (Laguna XS 2.1, Muse Glimmer ~28–30B) appear only as names. Laguna XS 2.1's Q4_K_M is ~20 GB (earlier report). Unsloth lists Muse-Glimmer-30B at 28B params, and its architecture and scores were not verified — [unsloth listing](https://huggingface.co/models?author=unsloth&sort=created&search=GGUF)

### Inferences
- **Flint UD-IQ3_XXS memory at 32K (estimate):** KV cache = 14 full-attn layers × 4 KV × 128 × 2 (K+V) × ~1.06 B (q8_0) ≈ 15 KB/token, so ~0.48 GB at 32K. The SWA layers add ~35 × 513 × 1 KB ≈ 18 MB. Total resident ≈ 14.5 + 0.5 KV + 0.5–1.0 compute buffers/driver ≈ **15.5–16 GB**. That is ~1–1.5 GB more than HIGH (~14–14.7 GB), still inside the 17–18 GB budget, but it leaves only ~1.5–2 GB RAM for browser and MCP. UD-Q2_K_XL (13.6 GB) would use about the same memory as HIGH. I found no KLD/perplexity data for Flint quants, so it is unknown how much quality the 2-bit and 3-bit files lose.
- **Flint speed estimate:** active ≈ 2.5B params at ~2.9 bpw ≈ 0.9 GB read per token, vs ~1.1 GB for Qwen3.6-35B-A3B IQ3_XXS. With ~7 GB of weights on the GPU and ~7.5 GB of experts in RAM, ~0.45 GB comes from RAM and ~0.45 GB from VRAM. That gives 4.5 ms + 10.7 ms ≈ 15 ms, a ceiling of ~65 tok/s, so ~25–35 tok/s at 40–55 % efficiency. After that come the 10–30 % auto-scaling penalty and the immature Vulkan path. Realistic range: **~12–22 tok/s, about the same as HIGH**. The 14 full-attention layers (vs 10 in Qwen3.6-35B-A3B) make prefill and long-context decode somewhat heavier.
- **Quality vs HIGH:** on vendor numbers Flint is about +1 SWE-V (74.2 vs 73.4, both self-reported) and about +3 to +10 TB2.x, but the harnesses differ: Flint TB2.1 54.9 (Recursive), Qwen3.6-35B-A3B TB2.0 51.5 (Qwen), TB2.1 44.38 (NVIDIA). That is a "maybe slightly smarter coder", not a generational jump, and it is far below Qwen3.8-27B's vendor TB2.1 73.0. **Flint is a sidegrade or alternative HIGH, not a true ULTRA.** It is also a non-Qwen family, which the earlier report valued as a hedge against template regressions.

### Gaps
- No independent benchmark for Flint (AA, SWE-rebench, Aider, LiveBench): none exists as of late Oct 2026.
- No Vulkan or AMD measurement for Flint, and no KLD per quant.
- The search index had almost no r/LocalLLaMA content from Sep–Nov 2026, so other Oct–Nov releases (Ling/Ring, Hunyuan, ERNIE, Seed, Mistral MoE, Kimi/MiniMax mini) **could not be checked**. Targeted searches returned nothing relevant in the 40–60B / 3–6B active range.
- The Recursive chart with peer scores (Qwen 3.5/3.6/3.8, Laguna XS 2.1, Muse Glimmer, Gemma 4) exists only as an image, so the numbers were not extracted.

## Q2: Is there a natively low-bit / QAT model that fits ~17 GB?

### Takeaway
I found no model natively trained or QAT'd at ~2–3 bits that fits. The NVFP4 models (Flint, Nemotron 3.5) are 4-bit and land at ~23 GB, so on this box they still need Unsloth's 2–3-bit post-training quants.

### Cited Findings
- Flint's final NVFP4 model is claimed to be "within 0.7 points of the 16-bit model" (vendor). The NVFP4 GGUF is **22.86 GB** and the int4 q4_1 GGUF is 25.99 GB, both over budget — [orcarouter](https://www.orcarouter.ai/blog/flint-40b-a2-5b-local-coding)
- Nemotron 3.5 Lightning: "pre-training used an NVFP4 recipe" and NVIDIA recommends its NVFP4 release. QAT is not described, and the smallest GGUF is 19.4 GB — [HF unsloth Nemotron Lightning](https://huggingface.co/unsloth/NVIDIA-Nemotron-3.5-Lightning-30B-A3B-GGUF)
- Nemotron 3.5 Bolt 74B-A9B 24GB-GGUF (Q2_0 tensor type) is 22 GB (earlier report) — [HF Nemotron Bolt](https://huggingface.co/nvidia/NVIDIA-Nemotron-3.5-Bolt-74B-A9B-24GB-GGUF)

### Inferences
- A "Bolt-style" low-bit native model at ≤17 GB would need ≤~45B params at 3 bpw, and none was found.

### Gaps
- I could not check whether NVIDIA released a smaller Bolt-style (Q2_0) variant after Oct 2026.

## Q3: Can speed boosters make Qwen3.8-27B or Qwen3.6-35B-A3B fast enough here?

### Takeaway
MTP in llama.cpp has been in mainline since 16 May 2026 (`--spec-type draft-mtp`). It gives a real **1.5–2.4× on dense Qwen3.6-27B when fully on GPU**, ~1.45× on 35B-A3B, and +30 % or even a slowdown on weaker Vulkan/APU setups. Applied to a half-offloaded dense 27B here, that turns an est. 4–6 tok/s into **an est. ~6–10 tok/s for code, less in tool-heavy turns**. It reaches the ~10 tok/s target only in the best case. The "4.8×" headline figure is misleading.

### Cited Findings
- PR #22673 "llama + spec: MTP Support" by am17an was **merged into master on 16 May 2026** (commit 4f13cb7). Follow-up refinements landed in PR #23269. The flag was renamed to `--spec-type draft-mtp` on 2026-05-13 — [llama.cpp PR #22673](https://github.com/ggml-org/llama.cpp/pull/22673), [search summary incl. LocalAI bump](https://gitea.varghacsongor.hu/GithubMirror/LocalAI/src/commit/3cba35ed320baa69728605c040e80bc3a84fae9c/core)
- Controlled-ish measurements (Qwen3.6-27B, temp 0, median of 5, `--spec-draft-n-max 2`): Strix Halo ROCm Q4_K_M **11.7 → 21.2 (1.81×)**, Q8_0 7.4 → 18.1 (2.44×). RTX 3090 Q4_K_M 38.7 → 59.5 (**1.54×**; the article's "2.17×" does not match its own data). M2 Max ~12 → 9–10 (slowdown). Community acceptance is **79–89 % for code** and **62–70 % for factual/structured output**. Tool-heavy agentic work is expected to gain less or even slow down (author's inference). Launch builds had a pp regression, and fixed builds give 1.5–1.8×. **MTP only works for Qwen3.5/3.6 weights**, and other models error out — [SpecPicks](https://specpicks.com/reviews/llama-cpp-mtp-support-landed-qwen3-6-27b-at-2-44-on-a-strix-halo-2-17-on-a-rtx-3090-rig-news)
- The "4.8× faster Qwen3.6-27B on Strix Halo" post compares a **Q8_K_XL baseline without MTP (6.2 tok/s)** against **Q4_K_M + q4_0 KV with MTP (29.8 tok/s)**. Most of the gain is the smaller quant. The like-for-like Q8_0 comparison gives ~3.0×. It used am17an's unmerged branch on Vulkan RADV with coopmat — [sleepingrobots](http://sleepingrobots.com/dreams/mtp-qwen36-strix-halo/)
- Reddit reports: 7900 XTX Vulkan Qwen3.6-27B 37 → 80 tok/s, **35B-A3B 118 → 171 tok/s (~1.45×)**. AMD APU Vulkan "maximum 30 % increase". Intel Arc 140V Vulkan was slower with MTP. One run had acceptance 0.745. MTP needs single-slot mode (`-np 1`), and mmproj crashes with MTP — [r/LocalLLaMA snapshot](https://reddit.sentinel-team.org/posts/1tes1wx/snapshots/2026-05-23T00%3A36%3A34.219378Z), [mer.vin](https://mer.vin/2026/05/run-qwen-3-6-mtp-in-llama-cpp-faster-local-inference-with-built-in-speculative-decoding/)
- The earlier report found that a separate draft model on Vulkan came out at 0.95× ([InventiveHQ](https://inventivehq.com/blog/llama-cpp-speculative-decoding-consumer-gpu)), and that `--spec-type ngram-mod` is draftless and helps repeated code blocks ([llama.cpp speculative.md](https://sources.debian.org/src/llama.cpp/8064%2Bdfsg-1/docs/speculative.md)). I found no new measurements for either in this round.

### Inferences
- **Qwen3.8-27B UD-IQ3_XXS (~11–12 GB) baseline here:** ~6.5 GB on the GPU + ~5 GB in RAM gives 65 ms + 120 ms ≈ 185 ms per token, a ceiling of ~5.4 tok/s, so **~4–5 tok/s** realistic. The ceiling is set by RAM bandwidth, and only a smaller quant moves it.
- **With MTP (draft-n-max 2):** each step reads all weights once and verifies 3 positions. At 0.75 acceptance the expected yield is 1 + 0.75 + 0.56 ≈ 2.3 tokens per step, minus MTP head and CPU-compute overhead (6 cores computing 3 positions of a 13B-parameter CPU share is no longer purely bandwidth-bound). That suggests **~1.5–2× → ~6–10 tok/s for code-writing turns** and ~1.2–1.5× (~5–7 tok/s) for tool-calling/JSON turns. This holds only if Qwen3.8-27B ships MTP heads and the MTP GGUF (+~1 GB per Unsloth) still fits. The earlier report said Unsloth gives ~1.4× for dense models. Inference, not measured.
- **For a smaller dense quant:** a UD-IQ2-class Qwen3.8-27B (~8–9 GB, size not verified) would put ~75 % on the GPU, giving ~7–8 tok/s baseline and ~10–14 with MTP, at an unknown quality cost (2-bit dense degradation is unmeasured).
- **Qwen3.6-35B-A3B + MTP with experts in RAM:** verifying 3 tokens routes each position to different experts. The union of experts read from RAM grows, so the gain should be below the 1.45× seen fully on GPU, probably ~1.1–1.35×. That adds ~2–5 tok/s on top of HIGH. Worth A/B testing, but it does not change the tier structure.

### Gaps
- No MTP measurement anywhere with **partial CPU offload** (dense or MoE), on **RDNA2 without coopmat**, or on a PCIe x4 card.
- Not verified: whether Qwen3.8-27B has MTP heads and whether an Unsloth MTP GGUF exists, and the exact sizes of UD-IQ2 quants for Qwen3.8-27B.
- No fresh measurements of ngram-mod gains on agentic edit workloads.

## Q4: Is Qwen3.6-35B-A3B at Q3_K_XL with thinking a sensible ULTRA?

### Takeaway
No. It adds +3.6 GB for a small KLD gain and breaks the RAM budget (~18.3 GB resident). Thinking-on with the same weights is a mode of HIGH and needs no separate tier.

### Cited Findings
- Unsloth KLD (analogous Qwen3.5 ~35B-A3B): IQ3_XXS 0.050, Q3_K_XL 0.031, Q4_K_M 0.019 — [Unsloth GGUF benchmarks](https://unsloth.ai/docs/models/qwen3.5/gguf-benchmarks)
- File sizes: UD-IQ3_XXS 13.2 GB, UD-Q3_K_XL 16.8 GB — [HF unsloth/Qwen3.6-35B-A3B-GGUF](https://huggingface.co/unsloth/Qwen3.6-35B-A3B-GGUF)
- Independent signal: on SWE-rebench Qwen3.6-35B-A3B scores 24.7 % vs dense Qwen3.6-27B 31.2 % (May–Jul 2026 window). Dense 27B-class models are clearly smarter than the 35B-A3B on uncontaminated tasks — [swe-rebench.com](https://swe-rebench.com/)

### Inferences
- **Candidate summary for the writer:**

| Candidate | Release | Total/active | Quant / file | Resident @32K (est.) | Est. tok/s here | Quality evidence |
|---|---|---|---|---|---|---|
| Qwen3.6-35B-A3B (current HIGH) | Apr 2026 | 35B/3B | UD-IQ3_XXS 13.2 GB | ~14–14.7 GB | 12–20 (+MTP maybe 14–25) | SWE-V 73.4 (vendor), TB2.1 44.38 (NVIDIA), SWE-rebench 24.7 % (indep.) |
| **Recursive Flint** | 21 Oct 2026 | 39.8B/2.5B | UD-IQ3_XXS 14.5 GB (or Q2_K_XL 13.6) | ~15.5–16 GB | ~12–22 | TB2.1 54.9, SWE-V 74.2 (vendor only); vendor admits Qwen3.8-27B is better on a 6-bench mean |
| Qwen3.8-27B dense (current ULTRA) | ~14 Aug 2026 | 27B dense | UD-IQ3_XXS ~11–12 GB | ~12–13 GB | 4–5; with MTP ~6–10 (code) | TB2.1 73.0, SWE-Pro 61.7 (vendor only) |
| Qwen3.8-27B smaller quant + MTP | – | 27B dense | IQ2-class ~8–9 GB (size unverified) | ~10 GB | ~10–14 (speculative) | unknown 2-bit degradation |
| Qwen3.6-35B-A3B Q3_K_XL | Apr 2026 | 35B/3B | 16.8 GB | ~18.3 GB (over budget) | ~11–18 | KLD 0.031 vs 0.050 |
| Nemotron 3.5 Lightning | 11 Aug 2026 | 30B/3B | ≥19.4 GB | doesn't fit | – | SWE-V 51.6 (weaker) |

- **Synthesis:** there is no model in Nov 2026 that is both clearly smarter than HIGH and ≥10 tok/s on this box. The realistic options:
  - (a) Keep Qwen3.8-27B as ULTRA and enable MTP, which brings it to ~6–10 tok/s for code. Optionally test a lower quant to reach ~10+.
  - (b) Add Flint as an "ULTRA-fast" or alternative HIGH: about HIGH speed, possibly slightly better at coding on vendor numbers, unverified, with fresh llama.cpp support that has a tool-call bug (#31000), a Jinja template issue and an unknown Vulkan path.
  - (c) Make ULTRA a *mode* (HIGH model + thinking + MTP) rather than a new model.
- Decision should rest on a local benchmark: llama-bench + 10–15 real NØX tasks comparing Flint IQ3_XXS vs Qwen3.6-35B-A3B IQ3_XXS vs Qwen3.8-27B IQ3_XXS+MTP.

### Gaps
- No independent head-to-head data between Flint and the Qwens (AA, SWE-rebench, Aider and LiveBench all lack Flint and Qwen3.8-27B).
- No RX 6500 XT measurement for any candidate except the 4B.
