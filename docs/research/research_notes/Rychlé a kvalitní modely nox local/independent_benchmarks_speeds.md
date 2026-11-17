# Independent benchmarks and measured speeds for local agentic-coding models on low-end hardware (as of Nov 2026)

Labels used: **[IND]** = independent third-party measurement, **[VEN]** = vendor-reported, **[EST]** = estimate/extrapolation, **[COMM]** = community self-report (independent but informal, small n).
Target: RX 6500 XT 8 GB (Vulkan, ~100 GB/s, PCIe 3.0 x4), Ryzen 5 5600GT, 16 GB DDR4, ~17–18 GB for model+KV.

## Q1: Where do the candidates rank on independent leaderboards, and do vendor claims hold up?

### Takeaway
Independent leaderboards cover only a few of the candidates. The ones that are covered sit far below vendor headline numbers. On SWE-rebench (fresh tasks from May–Jul 2026), Qwen3.6-35B-A3B resolves **24.7%** and Qwen3.6-27B 31.2%. That is a different benchmark from SWE-bench Verified, but it is the contamination-free signal, and it puts both models roughly 30–40 points behind frontier models. Artificial Analysis rates Qwen3.8-27B clearly above Qwen3.6-35B-A3B. One independent lab (Quesma) reproduced Qwen's Terminal-Bench 2.1 result for Qwen3.8-27B at BF16. Most other candidates (Qwen3.5-4B/9B, Nemotron 3 Nano, GLM-4.7-Flash, gpt-oss-20b, Devstral Small 2, Laguna XS 2.1, Mellum 2.1) have **no current independent leaderboard score** in the sources found.

### Cited Findings
**SWE-rebench [IND]** (window 15 May–1 Jul 2026, 111 problems, 65 repos):
- Qwen3.6-27B 31.2 ± 1.68% (pass@5 57.7). Qwen3.6-35B-A3B **24.7 ± 0.79%** (pass@5 43.2). Qwen3.5-35B-A3B 17.1 ± 1.07% (pass@5 36.9). For comparison: Fable 5 64.5%, GLM-5.2 62.9%, DeepSeek-V4 Pro 40.2%, MiniMax M3 47.2% — [swe-rebench.com](https://swe-rebench.com/)
- In this window, GLM-4.7 Flash, gpt-oss-20b, gpt-oss-120b, Devstral-Small-2-24B-2512, Gemma 4 31B, Qwen3-Coder-30B-A3B and Qwen3.5-27B are listed as **N/A** (not run). **Qwen3.8 is absent.** Nemotron, Laguna and Mellum are not on the page at all — [swe-rebench.com](https://swe-rebench.com/)
- Vendor comparison: Qwen claims SWE-bench Verified 73.4 and SWE-bench Pro 49.5 for Qwen3.6-35B-A3B [VEN] — [benchlm.ai](https://benchlm.ai/models/qwen3-6-35b-a3b). The SWE-rebench number is not directly comparable, but the gap shows how much Verified is inflated or saturated.
- Older reference: BenchLM lists a SWE-rebench score of 53.7% for Qwen3.5-35B-A3B, from a different (earlier) window that cannot be compared with the current 17.1% — [benchlm.ai](https://benchlm.ai/md/models/qwen3-5-35b-a3b.md)

**Artificial Analysis [IND]**: AA appears to have **changed its index methodology**, so the numbers below conflict:
- Earlier index version (Aug–Sep 2026): Qwen3.8-27B Intelligence Index **52 (xhigh), 44 (medium), 43 (low), 35 (non-reasoning)**. Agentic Index 50.9 ("51"). Qwen3.6-27B reasoning: 38. AA quoted no Coding Index number. Non-reasoning speed 53.1 tok/s (API) — [qubrid.com summary of AA](https://www.qubrid.com/blog/qwen38-27b-benchmarks-official-and-independent-results); [AA model page](https://artificialanalysis.ai/pt/models/qwen3-8-27b)
- Current AA comparison page (new eval suite): Qwen3.8-27B Intelligence Index 20 / 17 / 16 / 10 (xhigh/medium/low/non-reasoning). Qwen3.6-35B-A3B "--" (not computed). Per-eval scores: AutomationBench-AA **48.2% vs 5.2%** (3.8-27B xhigh vs 3.6-35B-A3B), **Terminal-Bench 4.0 5.6% vs 0.0%**, GDP.pdf 16.4% vs 6.6%, AA-Briefcase 1391 vs 713. Time per task 44 min (xhigh) vs 18 min (non-reasoning) — [AA comparison](https://artificialanalysis.ai/models/releases/comparisons/qwen3-8-27b-vs-qwen3-6-35b-a3b)
- AA "Progress Index v1", small open models (4–40B): Qwen3.8-27B xhigh 303 / medium 290 / low 283 / non-reasoning 262. Qwen3.6-35B-A3B reasoning **261** / non-reasoning 246. Gemma 4 31B reasoning 254. Qwen3.5-4B/9B, Nemotron 3 Nano, GLM-4.7-Flash, gpt-oss-20b, Devstral Small 2, Laguna and Mellum are **not listed** on this page — [AA small open models](https://artificialanalysis.ai/models/open-source/small)

**Terminal-Bench**:
- Quesma [IND] ran Terminal-Bench 2.1 (89 tasks, 3 h timeout, xhigh, 98K context). Qwen3.8-27B BF16 **"replicates Qwen's reported score"** (the vendor claim is 73.0) — [Quesma](https://quesma.com/blog/qwen38-27b-quantizations-benchmarked/). The article text gives no exact % and states neither the number of runs nor the harness. This is the only independent reproduction found of any vendor agentic claim among the candidates.
- The official tbench.ai leaderboard now runs **Terminal-Bench 4.0**. The page loaded without rows, so no small-model entries could be extracted — [tbench.ai](https://www.tbench.ai/leaderboard)

**Aider polyglot**:
- No official aider leaderboard entry exists for any candidate. One developer's custom "little-coder" agent reported 78.7% on Polyglot with Qwen3.6-35B [COMM, non-standard harness]. Commenters say the Aider leaderboard is "not really maintained" — [r/LocalLLaMA snapshot Apr 2026](https://reddit.sentinel-team.org/posts/1ssilc3/snapshots/2026-04-25T00%3A46%3A56.728262Z)

**Practitioner agentic tests on 8 GB [COMM/IND, small n]** (RTX 4060 Ti 8 GB, Q4_K_M, May 2026):
- gpt-oss-20b: **10/10 agentic tasks passed** (Pi harness) — "the first viable local coding agent on 8GB VRAM". Qwen3.6-35B-A3B: completed the easy task (18 min with thinking off, 49 min with thinking on); the hard task hit the context limit and was killed after 87 min. 9B models (Qwopus3.5-9B-Coder, OmniCoder-9B): malformed patch JSON, reasoning loops, partial completions. Gemma-4-E4B: easy task done in ~30 s, hard task partial — [witcheer dataset](https://huggingface.co/datasets/witcheer/local-agentic-coding-bench-8gb-vram-2026-05)
- GLM-4.7-Flash scored 4/6 on a quality test with the "worst hallucination" — [witcheer collection](https://huggingface.co/collections/witcheer/8gb-vram-local-llms-practitioner-tested)
- From the earlier report (still relevant): Qwen3.5-4B ranked 1st of 13 local models on tool-calling (97.5%), with GLM-4.7-Flash and Nemotron 3 Nano 4B at 95% [COMM] — [jdhodges.com](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)

**Qwen3.8-27B other independent signals**: LlamaIndex ExtractBench FP8 mean 89.75. Simon Willison: an agent-loop tool call worked correctly, but 22,276 reasoning tokens went into one SVG task. Tunguz (9 tasks): ~30× slower with reasoning on [COMM] — [qubrid.com](https://www.qubrid.com/blog/qwen38-27b-benchmarks-official-and-independent-results)

### Inferences
- The quality order supported by independent evidence: **Qwen3.8-27B > Qwen3.6-27B > Qwen3.6-35B-A3B > Qwen3.5-35B-A3B** (AA plus SWE-rebench agree). How much better 3.8-27B is than 3.6-35B-A3B on agentic tasks is large (AA AutomationBench 48% vs 5%; on TB4.0 both are near zero).
- Vendor SWE-V numbers (73.4 for 35B-A3B) should not be used for ranking. On fresh tasks the 35B-A3B resolves ~1 in 4.
- gpt-oss-20b is the dark horse: no leaderboard data, but the only practitioner test on 8 GB found it the most reliable agent. Its token-efficient Harmony format and small active parameter count also make it fast.
- The Qwen3.8-27B TB2.1 73.0 claim holds at BF16 (Quesma), but how much of that survives at 3-bit and at 4–6 tok/s on this hardware is unknown. Reasoning-token volume (AA: 160M tokens vs a 48M peer median) makes it impractical as a loop driver on slow hardware.

### Gaps
- No LiveBench, BFCL v4, tau2 (independent), or LMArena WebDev numbers were retrieved for any candidate. Searches did not surface them.
- No independent scores found for Qwen3.5-4B/9B, Nemotron 3 Nano 4B/30B-A3B, Devstral Small 2, Laguna XS 2.1, Mellum2.1, or Gemma 4 E4B/12B/26B. AA's current small-model page omits them.
- AA Coding Index values for Qwen3.8-27B / Qwen3.6-35B-A3B were not visible. The AA index version change makes the old (52) and new (20) figures incomparable.
- The exact Quesma TB2.1 percentages are not in the article text, only in the charts.

## Q2: How much do 3-bit / 4-bit quants degrade agentic coding?

### Takeaway
4-bit (Q4_K_M) is independently shown to match BF16 on Terminal-Bench 2.1 for Qwen3.8-27B. 2-bit (UD-Q2_K_XL) shows a "noticeable drop" and 1-bit collapses. No independent agentic test of **3-bit** exists. KLD/retention data puts IQ3_XXS at roughly 95% retention for the 27B dense model, but sources conflict. A τ²-bench study shows 4-bit quantization leaves final scores flat while **amplifying tool-name hallucinations up to 2.5×**. Agent retries hide this, but it costs time on slow hardware.

### Cited Findings
- Quesma [IND], Qwen3.8-27B, Terminal-Bench 2.1: BF16 (55 GB) replicates the vendor score. **Q4_K_M (17 GB, Unsloth) matches BF16.** UD-Q2_K_XL (10.7 GB) shows a noticeable drop and writes ~25% more output tokens on the same solved tasks. 1-bit (UD-IQ1_S/M) collapses (tested only on GPQA/IFBench). **Q3 was not tested** — [Quesma](https://quesma.com/blog/qwen38-27b-quantizations-benchmarked/)
- arXiv 2607.27275 [IND, academic]: 4-bit PTQ vs 8/16-bit on τ²-bench (telecom, retail), dense and MoE models from two families. Quantization "looks free on the standard metric" (changes bounded within ±7.5 points). Tool-name hallucination in telecom amplified up to **2.5×** (+17.6 points per task). Shrinking the error budget from 10 to 2 reveals a **17-point score gap**. A targeted error-repair prompt removes the damage — [arXiv 2607.27275](https://arxiv.org/abs/2607.27275)
- Benjamin Marie comparison of 15 Qwen3.8-27B GGUFs (scraped table, columns unlabeled) [IND]: UD-IQ3_XXS **95.5% retention**, UD-Q3_K_XL 100.0%, UD-IQ2_XXS 74.3%, UD-IQ1_M 53.4% — [via search snippet / scuttle.klotz.me](https://scuttle.klotz.me/tags/gguf+quantization?page=0). The search snippet also reported UD-Q4_K_XL at 101.0%.
- A conflicting Reddit KLD table (May 2026, so likely Qwen3.5/3.6-27B, not 3.8) gives UD-IQ3_XXS mean KLD ~0.177 vs UD-Q3_K_XL ~0.105, with degradation "more pronounced from Q3_K_XL downward" — [r/LocalLLaMA snapshot](https://reddit.sentinel-team.org/posts/1tr9vzn/snapshots/2026-05-30T00%3A45%3A07.677242Z). llmconfigurator gives bartowski IQ3_XXS a KLD of 0.0739 — [llmconfigurator](https://llmconfigurator.com/en/blog/qwen3-8-27b-which-quant-12-16-24-32gb)
- Unsloth [VEN], Qwen3.5 ~35B-A3B KLD: IQ3_XXS 13.12 GB mean KLD **0.0501**, Q4_K_XL 19.17 GB **0.0137**. Unsloth itself warns lower KLD "doesn't necessarily translate to better real-world performance" — [Unsloth GGUF benchmarks](https://unsloth.ai/docs/models/qwen3.5/gguf-benchmarks). No KLD table found specifically for Qwen3.6-35B-A3B.
- Unsloth [VEN]: heavy quantization retains knowledge but not reliable tool use, so the model may "fail to call tools, keep calling tools or not even call them" — [Unsloth Dynamic 3.0](https://unsloth.ai/docs/basics/dynamic-3.0-ggufs)

### Inferences
- Q4 is safe. 3-bit is probably acceptable for the 27B dense model (~95% retention). Hybrid MoE at IQ3_XXS (KLD ~0.05) is likely in a similar zone, but this is **unverified on agent tasks**.
- On slow hardware, quant-induced tool-call errors cost more: each retry adds minutes. A client-side retry-with-error prompt (as in the arXiv paper) is worth implementing.
- Lower quants also increase output length (Q2: +25% tokens), which partly cancels their speed advantage.

### Gaps
- No independent agentic (SWE/TB/τ²) test of IQ3_XXS or Q3_K_XL for any candidate.
- No quant-degradation data for Qwen3.5-4B/9B, gpt-oss-20b (MXFP4 is native, so not applicable), GLM-4.7-Flash, or Nemotron.

## Q3: Measured llama.cpp speeds on low-end hardware

### Takeaway
Measured numbers for **Qwen3.6-35B-A3B with expert offload on 8 GB cards cluster around 17–36 tok/s generation** (NVIDIA CUDA; one AMD RX 7600 ROCm Q8_0 at 18–21). No Vulkan + RDNA2 + PCIe x4 measurement exists. Full-VRAM 8–9B models run ~42–50 tok/s on a 4060 Ti (288 GB/s). Dense 27B with partial offload measured **3.28 tok/s** on an 8 GB card, which supports the earlier 4–6 tok/s estimate (or lower). GLM-4.7-Flash slows down badly as context grows (big attention heads). gpt-oss-20b and Nemotron 3 Nano 30B-A3B hold speed with depth much better.

### Cited Findings
**Qwen3.6-35B-A3B, MoE offload, 8 GB class [IND/COMM]:**
| Hardware | Backend | Quant / flags | Gen tok/s | PP tok/s | Source |
|---|---|---|---|---|---|
| RTX 4060 Ti 8 GB, 32 GB RAM | llama.cpp (Windows) | Q4_K_M, `-ncmoe 30`, 32K | **35** (7.4 at full CPU offload) | n/a | [witcheer](https://huggingface.co/collections/witcheer/8gb-vram-local-llms-practitioner-tested) |
| RX 7600 8 GB (RDNA3), RAM unspecified | ROCm 7 / HIP | **Q8_0** (37 GB), `-ngl 999 --n-cpu-moe 36 --no-mmap -ctk q8_0 -ctv q8_0 -t 12` | **18–21** (13 with active terminal) | n/a | [ai-manual.ru](https://ai-manual.ru/article/optimizatsiya-inferensa-qwen-36-35b-a3b-na-radeon-rx-7600-21-tokens-s-llamacpp-i-rocm-7/) |
| GTX 1060 6 GB, i3-8100, 24 GB RAM | CUDA (TurboQuant fork) | Q4_K_M, `--n-cpu-moe 36 --no-mmap`, tuned -ngl | 3 → 10 → 13.5 → **17** (step by step) | n/a | [jpmrblood](https://jpmrblood.codeberg.page/2026/06/02-llama-cpp-35b-model-6gb-vram-guide/) |
| RTX 2070 Super 8 GB, 32 GB DDR4 | llama.cpp | Q4_XL-MTP, KV q8 | ~**20** with MTP | n/a | [r/LocalLLaMA](https://reddit.sentinel-team.org/posts/1tjh7az/snapshots/2026-05-21T23%3A11%3A41.516272Z) |
| RTX 3060 Ti 8 GB, 32 GB DDR4-3200 | n/s | Q6, `n-cpu-moe 38` | ~33 (earlier report) | **700–750** on a ~50K prompt | [r/LocalLLaMA](https://reddit.sentinel-team.org/posts/1srijdf/snapshots/2026-04-25T00%3A46%3A56.728262Z) |
| RTX 4060 8 GB, DDR5 | CUDA | `--n-cpu-moe` | 30–35. Vulkan commenter: ~32 | n/a | [r/LocalLLaMA](https://reddit.sentinel-team.org/posts/1t7l56a/snapshots/2026-05-15T23%3A40%3A01.25497Z) |
| RTX 3070 Laptop 8 GB | CUDA | `--n-cpu-moe` | 4.02 → **32.82** | 110 → 341 | [XDA (earlier report)](https://www.xda-developers.com/i-stopped-thinking-my-8gb-gpu-was-too-small-after-learning-to-offload-moe-llm-layers-to-ram/) |

**Other models on 8 GB (RTX 4060 Ti 8 GB, Q4_K_M, LM Studio/llama.cpp) [COMM]** — [witcheer collection](https://huggingface.co/collections/witcheer/8gb-vram-local-llms-practitioner-tested):
- Nemotron-3-Nano-4B **80.7** tok/s. Gemma-4-E4B 68.5 (6.0 GB VRAM). Qwen3.5-9B **44.2** (6.2 GB VRAM). OmniCoder-9B 42. Qwen3-8B 50.8.
- GLM-4.7-Flash **29.9** tok/s at `ncmoe=35`, −14.7% at 24K. Gemma-4-26B-A4B 29.3 at `ncmoe 23`. LFM2-24B-A2B 52.2 at `ncmoe 22`.
- **Qwen3.6-27B dense, `ngl=20`: 3.28 tok/s** (the MoE sibling is 10.8× faster).
- gpt-oss-20b at `ncmoe=30`: speed not stated, only 1.8 GB VRAM used. Another source: ~88.9 tok/s on a 4060 Ti 8 GB with full offload (per search summary; unverified).
- Scaling to 4060 Ti bandwidth (288 GB/s) vs RX 6500 XT (~100 GB/s): full-VRAM models should run at ~0.35× on the target [EST]. The measured 37 tok/s for Qwen3.5-4B on the target fits that ratio.

**llama-bench with depth (Vulkan, AMD Renoir iGPU, UMA DDR4, 8 threads, `-ngl 100 -ub 512 -b 2048 --mmap 0`), FA on, pp512 / tg128** [IND] — [llama.cpp #19081](https://github.com/ggml-org/llama.cpp/issues/19081):
| Depth | gpt-oss-20B MXFP4 (11.3 GiB) | Nemotron-3-Nano 30B-A3.5B Q8_0 (31.3 GiB) | Qwen3-30B-A3B Q8_0 | GLM-4.7-Flash Q8_0 (29.7 GiB) |
|---|---|---|---|---|
| 0 | 146.7 / **16.6** | 112.5 / 12.1 | 114.3 / 13.4 | 86.1 / 11.0 |
| 2048 | 120.4 / 16.1 | 101.3 / 12.0 | 55.5 / 11.7 | 35.4 / 8.8 |
| 8096 | 77.3 / **14.9** | 77.9 / **11.5** | 20.8 / 8.9 | 12.9 / **5.6** |
- With FA off at 8K depth, GLM-4.7-Flash tg drops to **2.19**. The maintainer attributes this to GLM's large attention head size (512/576), and the issue was closed "not planned" — [#19081](https://github.com/ggml-org/llama.cpp/issues/19081)
- The RX 6500 XT Vulkan baseline (Llama 2 7B Q4_0): tg128 27.8, pp512 255 — [llama.cpp #10879](https://github.com/ggml-org/llama.cpp/discussions/10879) (from the earlier report).

**CPU-only DDR4**: no clean desktop DDR4 CPU-only llama-bench was found for any candidate. Renoir iGPU UMA numbers above (DDR4, shared memory) are the closest proxy: ~11–17 tok/s tg for ~3B-active MoE at Q8/MXFP4.

### Inferences
- For the target machine [EST]: Qwen3.6-35B-A3B IQ3_XXS with expert offload should land below the NVIDIA 30–35 range because of 100 GB/s VRAM, Vulkan, and PCIe x4. The earlier 12–20 tok/s estimate remains plausible. The RX 7600 ROCm Q8_0 result (18–21) is an encouraging upper anchor.
- The Renoir data suggests pure CPU on DDR4-3200 dual channel could reach ~12–16 tok/s for gpt-oss-20b, ~10–12 for Nemotron 30B-A3B, and ~10–13 for Qwen-A3B at short context [EST]. Prefill would be only ~100–150 t/s, so a 20K agent prompt would need ~2–3 min.
- **Long-context stability ranking** (what matters for agents): gpt-oss-20b ≈ Nemotron 3 Nano 30B (hybrid Mamba) > Qwen A3B > GLM-4.7-Flash (worst). The Qwen3.5/3.6 hybrid DeltaNet architecture should scale better than the Qwen3-30B row shown, but no deep-context measurement was found.
- Dense 27B on 8 GB: ~3.3 tok/s measured on a 288 GB/s card with DDR (unknown speed). Expect ~2.5–4 tok/s on the target [EST], plus very long reasoning traces, so it is not viable as an interactive tier.

### Gaps
- No measurement on RDNA2 Vulkan with MoE offload, and none on a PCIe x4 card. Prefill under offload over x4 remains the largest unknown.
- No llama-bench for Qwen3.6-35B-A3B at IQ3_XXS on any 8 GB card. Most reports are Q4_K_M or higher.
- No measured numbers for Devstral Small 2, Laguna XS 2.1, or Mellum2.1 on low-end hardware.
- No clean desktop CPU-only DDR4 numbers.

## Q4: Community consensus (2026) for 8 GB VRAM / 16 GB RAM agentic coding

### Takeaway
r/LocalLLaMA threads from Sep–Nov 2026 could not be retrieved. The available community/practitioner sources converge on: **Qwen3.5-9B Q4** as the default full-VRAM 8 GB pick, **Qwen3.6-35B-A3B with `--n-cpu-moe`** as the "big model on a small GPU" pick, and **gpt-oss-20b** as the most reliable agent in one hands-on 8 GB test. Guides list Devstral Small 2 as a 16 GB-VRAM model, so it is out of scope here. Gemma-4-26B-A4B and GLM-4.7-Flash get mixed reviews (hallucination, tool-call failures, context slowdown).

### Cited Findings
- 2026 guides: "Qwen 3.5 9B Q4 is the top pick for 8GB VRAM". Devstral Small 2 24B Q4_K_M (~15 GB) is "the strongest coder that fits a 16GB card". gpt-oss-20b (~16 GB) is "built for agentic multi-file work". For 16 GB RAM without a GPU: Qwen3.5-9B Q6_K [guides, low confidence] — [atomic.chat 8GB](https://atomic.chat/blog/guides/best-local-llm-8gb); [localaimaster](https://localaimaster.com/vram/best-coding-llm-8gb-vram); [zimaspace](https://shop.zimaspace.com/blogs/tech-ai-hub/best-local-coding-models-8gb-12gb-16gb-24gb-vram). The search summary flagged some guides as unreliable (they cite nonexistent models).
- witcheer (practitioner, 8 GB): gpt-oss-20b is "the first viable local coding agent on 8GB VRAM". 9B models had trouble with tool-call JSON. Thinking mode must be turned off for 9B coders (`-rea off`). Qwen3.6-35B-A3B was reliable but slow, and thinking mode tripled its task time — [witcheer dataset](https://huggingface.co/datasets/witcheer/local-agentic-coding-bench-8gb-vram-2026-05)
- InsiderLLM guide on running Qwen3.6 35B MoE locally recommends offload setups — [insiderllm](https://insiderllm.com/guides/best-way-run-qwen-3-6-35b-moe-locally/)

### Inferences
- The new evidence most likely to change the earlier decision is gpt-oss-20b. It is native MXFP4 (11.3 GiB file, no quant degradation question), the fastest ~3B-active MoE at depth in measured llama-bench, and the only 8 GB agent run with 10/10 passes. It deserves an A/B test against Qwen3.6-35B-A3B IQ3_XXS for the HIGH tier, and it could serve as the non-Qwen "insurance" model.
- Nemotron 3 Nano 30B-A3B holds tg flat with depth (12.1 → 11.5 tok/s at 8K), but it has no independent coding/agent scores, so it remains a speed candidate only.
- Qwen3.8-27B: independently the smartest candidate (AA, TB2.1 reproduced at BF16), but measured dense-offload speeds (~3 tok/s) plus heavy reasoning make it a planner/reviewer at best. The earlier report's framing holds.

### Gaps
- No Sep–Nov 2026 r/LocalLLaMA consensus threads were retrieved. The community picture comes from spring/summer 2026 snapshots and guides.
- No OpenCode/Cline-specific local comparisons with numbers were found in this pass.
