# MEDIUM/HIGH tiers for nox-local: small-active MoE head-to-head (as of 2026-11-07)

Hardware assumptions used in all math below: RX 6500 XT 8 GB, about 100 GB/s effective VRAM bandwidth (measured: Qwen3.5-4B UD-Q4_K_XL 2.91 GB gives 37 tok/s, which is about 108 GB/s effective). DDR4-3200 dual channel is 51 GB/s theoretical and about 40-45 GB/s practical. PCIe 3.0 x4 is about 3.5 GB/s. Budget is about 17-18 GB for model plus KV, hard.

Speed model: decode time per token ≈ (bytes of active weights on GPU / 100 GB/s) + (bytes of active weights in DDR4 / ~42 GB/s), then real-world efficiency of about 50-70% of that ceiling. Bytes per token ≈ active params × average bits per weight / 8. All speed figures in these notes are MY ESTIMATES unless a source is cited.

Source-type legend: **[V]** = vendor-reported, **[I]** = independent or third-party measured, **[E]** = my estimate.

---

## Which MoE gives the best quality per tok/s on DDR4 + 8 GB VRAM?

### Takeaway
Qwen3.6-35B-A3B at UD-IQ3_XXS stays the best quality-per-byte-read option. It reads only about 1.1 GB per token, which is less than gpt-oss-20b (about 2+ GB per token) and much less than dense Qwen3.5-9B (6 GB per token). So on this machine it is probably about as fast as MEDIUM in decode, or faster, while being much stronger. No Oct-Nov 2026 small-active MoE was found that beats it on both axes. Laguna XS 2.1 and GLM-4.7-Flash are the only quality challengers, and neither is clearly better. Q2_K_XL buys about 10% speed for a doubled KLD, which is not worth it. Q3_K_XL breaks the memory budget.

### Cited Findings

**Qwen3.6-35B-A3B (current HIGH)**
- Release April 16, 2026, Apache 2.0, 36B total / 3B active, 262K context, text+image+video input — [Artificial Analysis comparison](https://artificialanalysis.ai/models/comparisons/qwen3-6-35b-a3b-vs-gpt-oss-20b); [DEV review](https://dev.to/czmilo/qwen36-35b-a3b-complete-review-alibabas-open-source-coding-model-that-beats-frontier-giants-4382)
- [V] SWE-bench Verified 73.4, Terminal-Bench 2.0 51.5, MCPMark 37.0, TAU3 67.2, LCB v6 80.4 — [HF Qwen/Qwen3.6-35B-A3B](https://huggingface.co/Qwen/Qwen3.6-35B-A3B) (from the earlier nox report)
- [I] SWE-rebench (BenchLM mirror, snapshot July 29, 2026): predecessor **Qwen3.5-35B-A3B 53.7%**, Qwen3.5-27B 58.9%, MiniMax M2.7 51.9%, Gemma 4 31B 41.6%. No Qwen3.6-35B-A3B entry was found in that snapshot, and no November snapshot was found — [BenchLM SWE-rebench](https://benchlm.ai/md/benchmarks/swe-rebench.md)
- [I] Artificial Analysis now uses a new benchmark set. Qwen3.6-35B-A3B vs gpt-oss-20b: AutomationBench-AA 5% vs 0%, GDP.pdf 7% vs 2%, HLE-Diamond 7% vs 5%, AA-Briefcase Elo 713 vs 0, "Terminal-Bench 4.0" 0% vs 0%. The Intelligence/Coding/Agentic Index showed "no data" on the fetched page. Low confidence: the page rendered oddly, so treat these only as directional (Qwen > gpt-oss) — [Artificial Analysis](https://artificialanalysis.ai/models/comparisons/qwen3-6-35b-a3b-vs-gpt-oss-20b)
- Quant sensitivity, Unsloth KLD for the analogous Qwen3.5-35B-A3B (mean KLD / 99.9% KLD / size): UD-IQ2_XXS 0.185 / 4.22 / 9.09 GB; **UD-Q2_K_XL 0.097 / 2.91 / 12.04 GB**; **UD-IQ3_XXS 0.050 / 1.53 / 13.12 GB**; UD-IQ3_S 0.046 / 1.42 / 14.13 GB; UD-Q3_K_M 0.032 / 0.97 / 15.54 GB; **UD-Q3_K_XL 0.031 / 0.95 / 16.06 GB**; MXFP4_MOE 0.027 / 0.78 / 18.17 GB; Q4_K_M 0.019 / 0.55 / 18.49 GB. Unsloth calls IQ3_XXS the compromise and notes that I-quants are about 5-10% slower at inference — [Unsloth GGUF benchmarks](https://unsloth.ai/docs/models/qwen3.5/gguf-benchmarks)
- Qwen3.6 Unsloth file sizes: UD-Q2_K_XL 12.3 GB, UD-IQ3_XXS 13.2 GB, UD-Q3_K_XL 16.8 GB — [HF unsloth/Qwen3.6-35B-A3B-GGUF](https://huggingface.co/unsloth/Qwen3.6-35B-A3B-GGUF) (from the earlier report)
- [I] Measured offload speeds on 8 GB NVIDIA (not this hardware): RTX 3070 Laptop went from 4.02 to 32.82 tok/s with `--n-cpu-moe`; RTX 3070 Ti IQ3_XXS peaked at 43.8 tok/s at `-ncmoe 23` and dropped to 19.8 when VRAM overflowed; 3060 Ti + DDR4-3200 got about 33 tok/s — [XDA](https://www.xda-developers.com/i-stopped-thinking-my-8gb-gpu-was-too-small-after-learning-to-offload-moe-llm-layers-to-ram/), [Medium](https://medium.com/coding-nexus/one-flag-tripled-my-tokens-per-second-on-an-8gb-gpu-238f55d1e3bd), [r/LocalLLaMA](https://reddit.sentinel-team.org/posts/1srijdf/snapshots/2026-04-25T00%3A46%3A56.728262Z)
- [I] On a 3090 24 GB, moving 12 layers of FFNs to CPU cut Qwen3.6-35B-A3B from 139.6 to about 66 tok/s. No 8 GB or DDR4 results were found in this Oct 2026 measured roundup — [DEV measured Oct 2026](https://dev.to/rigbase/what-gpu-do-you-need-for-qwen38-qwen36-gemma-4-and-gpt-oss-locally-measured-oct-2026-1eg6)

**gpt-oss-20b**
- Release August 2025, Apache 2.0, 21B total / 3.6B active, 131K context, text only — [Artificial Analysis](https://artificialanalysis.ai/models/comparisons/glm-4-7-flash-vs-gpt-oss-20b)
- [V] SWE-bench Verified: 60.7 per the OpenAI model card as cited by an HN commenter (unverified), but **34.0** in Z.ai's GLM-4.7-Flash comparison table. tau2 47.7 (Z.ai table) — [HN](https://news.ycombinator.com/item?id=46679872), [llm-stats](https://llm-stats.com/models/compare/glm-4.7-flash-vs-gpt-oss-20b)
- [I] Measured on an **RTX 5060 8 GB + 32 GB DDR5**: Unsloth F16 GGUF (experts are MXFP4 natively), `-ngl 999 --n-cpu-moe 18 -c 65536 -fa on --jinja --cache-reuse 256`, reasoning_effort low → **33-35 tok/s**, about 1.2 GB VRAM headroom at 64K. Without the CPU spill it ran out of memory. A 26K-in / 2.7K-out call took 90-125 s; `--cache-reuse` cut a dry run from 145 s to 50 s — [hindsight #2525](https://github.com/vectorize-io/hindsight/discussions/2525)
- [I] The official llama.cpp guide gives `--n-cpu-moe` 12/16/22 depending on context — [llama.cpp #15396](https://github.com/ggml-org/llama.cpp/discussions/15396)
- [I] One guide reports decode falling to about 9 tok/s at 128K context on consumer cards — [runaihome](https://runaihome.com/blog/gpt-oss-20b-local-ai-hardware-guide-2026/)
- llama.cpp has a native Harmony parser for its tool calls — [function-calling.md](https://raw.githubusercontent.com/ggml-org/llama.cpp/master/docs/function-calling.md)

**GLM-4.7-Flash**
- Release January 2026, MIT, 31.2B total / 3B active, 200K context, text only — [Artificial Analysis](https://artificialanalysis.ai/models/comparisons/glm-4-7-flash-vs-gpt-oss-20b)
- [V] (Z.ai table) SWE-bench Verified **59.2** vs Qwen3-30B-A3B 22.0, gpt-oss-20b 34.0, Nemotron-3-Nano 38.8; tau2 **79.5** vs 49.0 / 47.7 / 49.0 — [llm-stats](https://llm-stats.com/models/compare/glm-4.7-flash-vs-nemotron-3-nano-30b-a3b)
- [I] Cloud output speed: GLM-4.7-Flash 116 tok/s vs gpt-oss-20b 148 tok/s; end-to-end 37.4 s vs 58.0 s, because GLM reaches the answer faster (time to first answer token 20 s vs 43 s, so fewer reasoning tokens) — [Artificial Analysis](https://artificialanalysis.ai/models/comparisons/glm-4-7-flash-vs-gpt-oss-20b)
- [I] Tied for first (95%) in an independent local tool-calling test. Followed the OpenCode agent framework well, but decode dropped from 80 to 20 tok/s as context grew — [jdhodges](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/), [Medium OpenCode test](https://medium.com/@christian.rute/opencode-local-llms-a-practical-test-with-glm-4-7-flash-and-nemotron-c3ba2cbda43f)

**Nemotron 3 Nano 30B-A3B**
- [V] SWE-Bench (OpenHands) 38.76; TauBench V2 average 49.04 — [arXiv 2512.20848](https://arxiv.org/pdf/2512.20848)
- [I] Artificial Analysis Intelligence Index 52 at launch (old index version), and the highest cloud output speed in this group at 220.8 tok/s — [NVIDIA Newsroom on X](https://x.com/nvidianewsroom/status/2000603537999343657), [llm-stats/AA via search](https://llm-stats.com/models/compare/glm-4.7-flash-vs-nemotron-3-nano-30b-a3b)

**Laguna XS 2.1 (poolside)**
- Release July 2, 2026, open weights (no license found on the aggregator page), FP8 variant available — [benchmarklist](https://benchmarklist.com/models/poolside-laguna-xs-2.1/)
- [V] SWE-bench Verified 70.8, SWE-bench Multilingual 63.2, SWE-bench Pro 47.6, Terminal-Bench 2.0 37.5. [I] ObviousBench 92.4%, Context Arena 14.0% — [benchmarklist](https://benchmarklist.com/models/poolside-laguna-xs-2.1/)
- Artificial Analysis Coding Index and Intelligence Index listed as not available — [magica](https://magica.com/blog/compare/laguna-xs-2-1-vs-nemotron-3-ultra-550b-a55b)
- It is in the Ollama library — [ollama](https://ollama.com/library/laguna-xs-2.1). The earlier report said the official GGUF Q4_K_M is about 20 GB and needs llama.cpp PR #25165 — [HF poolside GGUF](https://huggingface.co/poolside/Laguna-XS-2.1-GGUF)

**Qwen3-Coder small MoE**
- Qwen3-Coder-30B-A3B-Instruct (older, 2025) has Unsloth GGUFs — [HF](https://huggingface.co/unsloth/Qwen3-Coder-30B-A3B-Instruct-GGUF). It scores SWE-V 22.0 in Z.ai's table — [llm-stats](https://llm-stats.com/models/compare/glm-4.7-flash-vs-nemotron-3-nano-30b-a3b). Note that the table label says "Qwen3-30B-A3B", which may be the general model rather than the Coder model.
- Qwen3-Coder-Next is 80B-A3B: [I] SWE-rebench Pass@5 64.6%, Pass@1 40.0% — [aiproductivity](https://aiproductivity.ai/news/qwen3-coder-next-tops-swe-rebench-pass-5/). At 80B total, even 2-bit (~25+ GB) breaks the 17-18 GB budget [E].
- Community fine-tune Qwopus3.6-35B-A3B-Coder-MTP-GGUF (about July 2026, experimental, thinking-off coding agent) — [HF Jackrong](https://huggingface.co/Jackrong/Qwopus3.6-35B-A3B-Coder-MTP-GGUF)

### Inferences (speed math, all [E])

| Model / quant | File | Active params × bpw → bytes per token | Split (GPU / DDR4) at ~32K ctx | Ceiling | Realistic gen tok/s here | Quality signal |
|---|---|---|---|---|---|---|
| Qwen3.5-9B UD-Q4_K_XL (dense) | 5.96 GB | 9B × ~5.3 → **~6.0 GB** | 6.0 GB GPU / 0 | 100/6.0 ≈ 17 | **14-17** (anchored to measured 4B at 37) | [V] SWE-V ~50 (per JetBrains table), BFCL-V4 66.1 |
| Qwen3.6-35B-A3B UD-IQ3_XXS | 13.2 GB | 3B × ~3.0 → **~1.1 GB** | ~0.55 GB GPU (attention, shared expert, ~10 GPU layers of experts) / ~0.55 GB DDR4 (routed experts of ~30 layers) | 5.5 ms + 13 ms ≈ 18.5 ms → ~54 | **15-25** (efficiency 30-45% because of CPU expert compute on 6 cores, Vulkan, I-quant penalty 5-10%) | [V] SWE-V 73.4; [I] predecessor SWE-rebench 53.7 |
| Qwen3.6-35B-A3B UD-Q2_K_XL | 12.3 GB | 3B × ~2.8 → ~1.05 GB | ~1 GB more fits on GPU, ~0.45 GB DDR4 | ~60 | **17-27** (about +10%; K-quant avoids the I-quant penalty) | KLD 0.097 (about 2× IQ3_XXS), 99.9% KLD 2.91 |
| Qwen3.6-35B-A3B UD-Q3_K_XL | 16.8 GB | 3B × ~3.8 → ~1.43 GB | ~0.6 GB GPU / ~0.85 GB DDR4 | 6 + 20 ms → ~38 | **12-18** | KLD 0.031. **Model+KV about 18.3 GB, which breaks the budget** |
| gpt-oss-20b MXFP4 | ~12.8 GB | 3.6B active; experts ~2.4B × 4.25 → ~1.3 GB, plus attention/unembedding at 8-16 bit ~0.6-1 GB → **~2.0-2.3 GB** | n-cpu-moe ~18/24: ~1.1 GB GPU / ~0.95 GB DDR4 | 11 + 23 ms → ~30 | **16-22** (scaled from measured 33-35 on RTX 5060 GDDR7 448 GB/s + DDR5 ~70 GB/s: both memories here are ~1.6-4× slower) | [V] SWE-V 34-60.7 (conflicting), tau2 47.7 |
| GLM-4.7-Flash (31.2B/3B) UD-Q3_K_XL or IQ3_XXS | ~13-14 GB [unverified size] | 3B × ~3.2 → ~1.2 GB | similar to Qwen3.6 | ~50 | **13-22**, falls more with depth (80→20 tok/s reported on a fast GPU) | [V] SWE-V 59.2, tau2 79.5 |
| Nemotron 3 Nano 30B-A3B Q3/Q4 | ~15-18 GB at Q4 [unverified] | ~3.5B active × ~3.5 → ~1.5 GB | similar | ~40 | **13-20**; Mamba layers keep speed flat with depth | [V] SWE-V 38.8, tau2 49 |
| Laguna XS 2.1 (33B-A3B) ~Q3 | ~14-15 GB [unverified]; Q4_K_M ~20 GB | ~3B × ~3.3 → ~1.25 GB | similar | ~45 | **13-22**, if llama.cpp support is merged | [V] SWE-V 70.8, SWE-Pro 47.6, TB2 37.5 |

- **Key inversion:** Qwen3.6-35B-A3B IQ3_XXS reads about 1.1 GB per token, while dense Qwen3.5-9B reads about 6 GB. Even with half of that 1.1 GB coming from slow DDR4, the MoE's decode ceiling (~54 tok/s) is about 3× higher than the 9B's (~17). The 3060 Ti + DDR4 analog measured ~33 tok/s for 35B-A3B. That suggests HIGH may decode **as fast as or faster than MEDIUM** on this box. MEDIUM's real advantages are (a) prefill: everything stays on the GPU, so there is no PCIe x4 expert traffic, and (b) about 7 GB less RAM used.
- **gpt-oss-20b is dominated by Qwen3.6-35B-A3B IQ3_XXS:** more bytes per token (larger active, 4.25-bit experts, 8/16-bit attention) and lower vendor coding scores. It is not faster and not smarter here. Its only advantages are a smaller file (more headroom for 64K) and the mature Harmony parser.
- **GLM-4.7-Flash** is the only plausible "same speed, different strengths" alternative: [V] tau2 79.5 vs Qwen3.6 TAU3 67.2 (different benchmark versions, so not comparable), and it is less verbose in reasoning per AA. Its coding is clearly lower (SWE-V 59.2 vs 73.4, both vendor). Keep it as the cross-family fallback, not the primary.
- **Laguna XS 2.1** has vendor SWE-V about equal to Qwen3.6 (70.8 vs 73.4) and similar active size, so it gives no speed gain. Its runtime support is less mature. No reason to switch.
- **Quant choice for HIGH:** stay at UD-IQ3_XXS. Q2_K_XL only saves ~0.9 GB and maybe 10% speed, at double the KLD and nearly double the 99.9% tail KLD (2.91 vs 1.53); tool-call JSON is the kind of output that tail errors break. Q3_K_XL gives ~40% lower KLD than IQ3_XXS but adds ~3.6 GB and costs ~25% decode speed. UD-IQ3_S (14.1 GB, KLD 0.046) and UD-Q3_K_M (15.5 GB, KLD 0.032) are intermediate options if RAM allows. Q3_K_M is about 1.2 GB smaller than Q3_K_XL with nearly the same KLD, which makes it the best "quality" step if ~2.3 GB more RAM can be spared.
- **Prefill over PCIe 3.0 x4:** for every offloaded MoE, llama.cpp copies CPU-resident expert weights to the GPU for large batches. That is about 6-7 GB of experts in RAM ÷ 3.5 GB/s ≈ **~2 s per ubatch pass**. With `-ub 2048`, a 20K-token prompt needs ~10 passes, so ~20 s of transfer plus compute. If transfer is not used (`--no-op-offload`), the CPU computes the experts at maybe 50-150 t/s. Either way the first turn costs tens of seconds to minutes. This applies equally to Qwen3.6, gpt-oss, GLM, Nemotron and Laguna. Only dense-on-GPU models (Qwen3.5-9B) avoid it.

### Gaps
- No November 2026 SWE-rebench, Aider polyglot or LiveBench snapshot was found for any of these models. The latest SWE-rebench data is a July 29, 2026 third-party mirror without Qwen3.6-35B-A3B, GLM-4.7-Flash, gpt-oss-20b, Laguna XS 2.1 or Mellum 2.1.
- Artificial Analysis Coding/Agentic Index numbers could not be extracted: the comparison pages showed "no data" for the indices and a new benchmark set (AutomationBench-AA, GDP.pdf, HLE-Diamond, "Terminal-Bench 4.0") with values that look low and are hard to attribute.
- KLD at 3-bit was found only for Qwen3.5-35B-A3B (as a proxy for 3.6). No KLD tables were found for GLM-4.7-Flash, Nemotron 3 Nano, Laguna XS 2.1, gpt-oss-20b (MXFP4 is native, so there is no low-bit variant worth using) or Mellum.
- Exact GGUF sizes for GLM-4.7-Flash, Nemotron 3 Nano and Laguna XS 2.1 at 3-bit were not verified in this session. The table values are estimates from parameter counts.
- **Not researched within the tool budget (no sources found or fetched):** Devstral Small 2, Ling/Ring flash/mini, ERNIE-4.5-21B-A3B successors, Granite 4 MoE. From background knowledge only (unverified): Devstral Small is a ~24B dense model (it would decode like the ULTRA dense tier, ~4-6 tok/s), and Granite 4.0-H-Small is 32B with ~9B active (about 3× the bytes per token of A3B models). If correct, neither competes on speed. Ling-mini-2.0 (16B/1.4B active) would be very fast but has no evidence of agentic-coding strength. Verify before using any of this.
- No small-active MoE coding model released in October-November 2026 was found other than Mellum 2.1 (October 8, 2026). The search did not confirm any others; they may still exist.

---

## Does gpt-oss-20b or GLM-4.7-Flash beat Qwen3.5-9B as MEDIUM (faster or smarter)?

### Takeaway
Probably smarter, but not clearly faster, and they lose MEDIUM's two real advantages: all-GPU prefill and a small RAM footprint. Both need about 12-14 GB and expert offload, which makes them "HIGH-class" deployments. If you are going to offload anyway, Qwen3.6-35B-A3B IQ3_XXS dominates gpt-oss-20b and edges GLM-4.7-Flash on coding. The one model that could genuinely change MEDIUM is **JetBrains Mellum 2.1 (12B-A2.5B, MXFP4_MOE ~7.0 GB)**. It is likely about 2× faster than Qwen3.5-9B with slightly lower vendor coding scores, but it is completely unverified independently.

### Cited Findings
- [V] Qwen3.5-9B: BFCL-V4 66.1, LCB v6 65.6, TAU2 79.1 — [HF Qwen/Qwen3.5-9B](https://huggingface.co/Qwen/Qwen3.5-9B) (from the earlier report). A third party measured 6.96 GB VRAM at 32K — [localaimaster](https://localaimaster.com/vram/best-coding-llm-8gb-vram)
- Mellum 2.1: released **October 8, 2026**, Apache 2.0, 12B total / ~2.5B active, 64 experts with 8 routed, 131,072 context, 3 of 4 layers use a 1,024-token sliding window — [runaihome Mellum guide](https://runaihome.com/blog/jetbrains-mellum-2-1-local-coding-llm-gpu-guide-2026/)
- [V] Mellum 2.1: SWE-bench Verified 47%, LCB v6 82.0. In JetBrains' own table, **Qwen3.5-9B scores SWE-V 50%** and LCB v6 75.4, and leads Mellum on SWE-bench Pro, AIME and GPQA Diamond. "No independent SWE-bench run existed as of October 15" — [runaihome](https://runaihome.com/blog/jetbrains-mellum-2-1-local-coding-llm-gpu-guide-2026/)
- Mellum 2.1 GGUF sizes: MXFP4_MOE ~7.0 GB, Q4_K_M 7.52 GiB, Q6_K ~10.9 GB, Q8_0 ~12.9 GB. The llama.cpp `mellum` architecture was merged June 5, 2026 (PR #23966). There is an open Metal NaN bug for the Q8_0/Q6_K/Q4_K_M quants (#30579; Mac only). No measured tok/s had been published — [runaihome](https://runaihome.com/blog/jetbrains-mellum-2-1-local-coding-llm-gpu-guide-2026/); every benchmark is vendor-reported — [toolhalla](https://toolhalla.ai/blog/can-you-run-mellum-2-1-locally); sources disagree on the minimum llama.cpp build — [modelfit](https://modelfit.io/blog/jetbrains-mellum2-1-12b-coding-model-mac/)
- gpt-oss-20b at 8 GB VRAM needs `--n-cpu-moe ~18` and runs out of memory without it — [hindsight #2525](https://github.com/vectorize-io/hindsight/discussions/2525)
- [V] GLM-4.7-Flash SWE-V 59.2, tau2 79.5; gpt-oss-20b SWE-V 34.0 (Z.ai table) or 60.7 (OpenAI card via an HN comment) — [llm-stats](https://llm-stats.com/models/compare/glm-4.7-flash-vs-gpt-oss-20b), [HN](https://news.ycombinator.com/item?id=46679872)

### Inferences
- **Speed math [E]:**
  - Qwen3.5-9B: about 6.0 GB per token, all on GPU → **14-17 tok/s**. Prefill fully on GPU at ~300-350 t/s (the 4B measured 685 t/s pp512).
  - Mellum 2.1 MXFP4_MOE 7.0 GB: active 2.5B × ~4.25 bpw ≈ **1.33 GB per token**. The sliding-window attention keeps KV tiny. At 32K, put ~6.2 GB on the GPU and offload ~1 GB of experts with a small `--n-cpu-moe` (2-4 layers): GPU ~1.15 GB/100 = 11.5 ms, DDR4 ~0.19 GB/42 = 4.5 ms → ceiling ~60 tok/s → realistic **30-40 tok/s**, about 2× the 9B. Prefill penalty is small, because only ~1 GB of experts would cross PCIe (~0.3 s per ubatch).
  - gpt-oss-20b: **16-22 tok/s** (above), so not faster than the 9B in a meaningful way, and it adds prefill penalty plus ~6 GB RAM.
  - GLM-4.7-Flash: **13-22 tok/s**, same remarks.
- **Quality:** on vendor numbers, GLM-4.7-Flash (SWE-V 59.2) > Qwen3.5-9B (~50, per JetBrains' table) ≥ Mellum 2.1 (47) > gpt-oss-20b (34 or 60.7, conflicting). None of these are independent.
- **Recommendation logic for MEDIUM:** MEDIUM should be the "everything on GPU, low prefill cost, low RAM" tier. Under that rule, the candidates are Qwen3.5-9B (proven, Qwen tool format shared with other tiers) and Mellum 2.1 (faster, needs ~1 GB offload, unproven tool calling). gpt-oss-20b and GLM-4.7-Flash belong in a "HIGH alternative" slot. Mellum 2.1 is worth an A/B benchmark: `llama-bench` plus the nox tool-calling suite. Do not switch on vendor numbers alone.
- If MEDIUM's role is redefined as "fast HIGH" (decode speed only), Qwen3.6-35B-A3B UD-Q2_K_XL with high `--n-cpu-moe` would likely beat Qwen3.5-9B on both speed and quality, except for prefill and RAM.

### Gaps
- No independent benchmark or llama.cpp tool-call reliability report for Mellum 2.1 was found. Its chat template and tool-call format, and whether llama.cpp has a native parser for it, are unknown.
- Whether Mellum 2.1 is a thinking or instruct model by default was not verified (community GGUFs of Mellum2 exist in both Instruct and Thinking variants).
- Whether llama.cpp's Vulkan backend handles MXFP4_MOE efficiently on RDNA2 is unverified.

---

## Measured (not estimated) speeds on similar low-end setups (8 GB GPU + DDR4)

### Takeaway
Very few measured points exist, and none are on AMD, Vulkan or PCIe x4. The best measured analogs: Qwen3.6/3.5-35B-A3B on 8 GB NVIDIA cards with expert offload reaches 33-44 tok/s, and gpt-oss-20b on an RTX 5060 8 GB + DDR5 reaches 33-35 tok/s. Scaled to this box's ~4× slower VRAM and DDR4, both land around 15-25 tok/s. No measured 8 GB numbers were found for GLM-4.7-Flash, Nemotron 3 Nano, Laguna XS 2.1 or Mellum 2.1.

### Cited Findings
- [I] gpt-oss-20b, RTX 5060 8 GB, 32 GB DDR5, `--n-cpu-moe 18`, 64K ctx: **33-35 tok/s** gen; 26K-token prompt calls 90-125 s total — [hindsight #2525](https://github.com/vectorize-io/hindsight/discussions/2525)
- [I] Qwen3.6-35B-A3B (or 3.5) on 8 GB NVIDIA: 3070 Laptop **32.82 tok/s** / pp 341; 3070 Ti IQ3_XXS **43.8 tok/s** peak (19.8 on VRAM overflow); 3060 Ti + **DDR4-3200** **~33 tok/s**, pp 700-750 (PCIe x16) — [XDA](https://www.xda-developers.com/i-stopped-thinking-my-8gb-gpu-was-too-small-after-learning-to-offload-moe-llm-layers-to-ram/), [Medium](https://medium.com/coding-nexus/one-flag-tripled-my-tokens-per-second-on-an-8gb-gpu-238f55d1e3bd), [r/LocalLLaMA](https://reddit.sentinel-team.org/posts/1srijdf/snapshots/2026-04-25T00%3A46%3A56.728262Z)
- [I] Estimate-only source: gpt-oss-20b on 8 GB with CPU offload at ~19 tok/s — [modelfit](https://modelfit.io/blog/gpt-oss-20b-vram-requirements/)
- [I] Reference full-GPU numbers: gpt-oss-20b on an RTX 3060 12 GB 83.96 tok/s; Gemma 4 12B on an RTX 3060 12 GB 38.5 tok/s; Strix Halo Vulkan/ROCm Qwen3.6-35B-A3B tg128 50.2-62.4 — [DEV measured Oct 2026](https://dev.to/rigbase/what-gpu-do-you-need-for-qwen38-qwen36-gemma-4-and-gpt-oss-locally-measured-oct-2026-1eg6)
- [I] RX 6500 XT Llama 2 7B Q4_0 Vulkan: tg128 27.8, pp512 255 — [llama.cpp #10879](https://github.com/ggml-org/llama.cpp/discussions/10879); this machine measured Qwen3.5-4B UD-Q4_K_XL at 37 tok/s gen and 685 t/s pp512 (earlier nox report)

### Inferences
- Scaling rule [E]: the RTX 5060 has ~448 GB/s VRAM and DDR5 ~65-80 GB/s. This box has ~100 GB/s VRAM and ~42 GB/s DDR4. For offloaded MoE where DDR4 dominates the per-token time, expect roughly 0.5-0.65× the DDR5 rig's speed → gpt-oss-20b **~17-22 tok/s**. Against the 3060 Ti + DDR4 rig (448 GB/s VRAM, same DDR4), Qwen3.6-35B-A3B IQ3_XXS should lose ~20-40% from slower VRAM, 6 vs more cores, and Vulkan → **~18-26 tok/s**. That is slightly above the earlier report's 12-20 estimate.
- Prefill is where this box will differ most from the analogs: the 3060 Ti reached pp 700-750 on PCIe x16 (~13-16 GB/s), while this card's PCIe 3.0 x4 is ~4× narrower. Expect pp ~150-300 t/s for offloaded MoE [E]. That makes the earlier report's "prefill is the bottleneck" conclusion stronger.

### Gaps
- No measurement on RDNA2/Vulkan with MoE offload, and none on a PCIe x4 card, was found for any model.
- No 8 GB measurements for GLM-4.7-Flash, Nemotron 3 Nano 30B-A3B, Laguna XS 2.1 or Mellum 2.1.
- r/LocalLLaMA threads from September-November 2026 did not appear in searches.
