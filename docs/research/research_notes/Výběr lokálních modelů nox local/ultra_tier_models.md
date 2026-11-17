# ULTRA tier: the strongest local LLM for agentic coding on 8 GB VRAM + 16 GB RAM (state as of November 2026)

**Verified hardware (from the coordinator):** Ryzen 5 5600GT (AM4, 6C/12T), DDR4 dual-channel (~51 GB/s theoretical at 3200 MT/s), GPU RX 6500 XT 8 GB on PCIe 3.0 x4, so no CUDA (llama.cpp Vulkan or ROCm). The desktop idles at ~6 GB RAM. **Realistic budget for weights + KV + compute buffers is about 17–18 GB** (8 GB VRAM + ~9–10 GB RAM), not 20–21 GB.

Research was capped at ~15 search/fetch calls. Many sources are secondary blogs, so every figure has its source next to it. Inferences and estimates are marked separately.

---

## 1. Which candidates actually exist in November 2026 (verification)

### Takeaway
Verified: **Qwen3.6-35B-A3B** (Apr 2026, hybrid DeltaNet MoE, Apache-2.0), **Qwen3.8-27B** (dense, 14 Aug 2026), **NVIDIA Nemotron 3.5 Bolt 74B-A9B "24GB" GGUF** (15 Oct 2026), Qwen3-Coder-Next 80B-A3B, and Gemma 4 26B-A4B. "nemotron-bolt-74b" exists, and its correct name is `NVIDIA-Nemotron-3.5-Bolt-74B-A9B-24GB-GGUF`. At ~22 GB it does not fit this machine.

### Cited Findings

**Qwen3.6-35B-A3B**
- Released April 2026, after Qwen3.5 (Feb 2026). 35B total / 3B active, 256 experts (8 routed + 1 shared), 40 layers in a hybrid layout: 10× (3× Gated DeltaNet→MoE + 1× Gated Attention→MoE). So only **10 of 40 layers have a KV cache**. Native context 262,144 tokens (YaRN to ~1M). Apache-2.0. Includes a vision encoder. — [HF unsloth/Qwen3.6-35B-A3B-GGUF](https://huggingface.co/unsloth/Qwen3.6-35B-A3B-GGUF)
- Vendor benchmarks: SWE-bench Verified **73.4**, Terminal-Bench 2.0 **51.5**, TAU3-Bench 67.2, LiveCodeBench v6 **80.4**. BFCL not reported. — [HF model card](https://huggingface.co/unsloth/Qwen3.6-35B-A3B-GGUF)
- Unsloth GGUF sizes: UD-IQ2_XXS 10.8 GB, UD-IQ2_M 11.5 GB, UD-Q2_K_XL 12.3 GB, **UD-IQ3_XXS 13.2 GB**, **UD-Q3_K_XL 16.8 GB**, UD-Q4_K_M 22.1 GB, UD-Q4_K_XL 22.4 GB. — [HF unsloth GGUF](https://huggingface.co/unsloth/Qwen3.6-35B-A3B-GGUF)
- An UD-IQ4_XS (16.5 GiB) also exists and has been tested on an 8 GB GPU. — [windowsforum](https://windowsforum.com/news/llama-cpp-n-cpu-moe-run-qwen3-35b-faster-on-8gb-rtx-laptop-gpus.449607/)
- Tool calling: `qwen3_coder` parser (vLLM/SGLang), thinking on by default (disable via `chat_template_kwargs: {"enable_thinking": false}`). `preserve_thinking` keeps reasoning across turns for agents. No `/think` `/nothink` switches. Unsloth notes fixes for developer-role support and parsing of nested objects in tool calls. — [HF model card](https://huggingface.co/unsloth/Qwen3.6-35B-A3B-GGUF)

**NVIDIA Nemotron 3.5 Bolt 74B-A9B (24GB GGUF)**
- HF: `nvidia/NVIDIA-Nemotron-3.5-Bolt-74B-A9B-24GB-GGUF`, released 15 Oct 2026. 74B total / 9B active, Mamba-2–Transformer hybrid LatentMoE with a vision encoder (C-RADIO v4-H). Compressed from Nemotron 3.5 Super VL 120B-A12B. Top-K 9, context up to 256K. License **OpenMDW-1.1**. — [HF model card](https://huggingface.co/nvidia/NVIDIA-Nemotron-3.5-Bolt-74B-A9B-24GB-GGUF)
- Natively 2-bit (experts IQ2_XXS, QAT/QAD). The official GGUF is **22.02 GB**. Arch `nemotron_h_moe`. llama.cpp PR #30520 was merged 15 Oct, so a build from that date or later is required. LM Studio cannot yet read the Q2_0 tensor type. Open Metal segfault (#30610). There is no smaller official quant, and 16 GB cards are out. — [runaihome.com](https://runaihome.com/blog/nemotron-35-bolt-74b-24gb-local-ai-2026/)
- NVIDIA's official recipes start at a **28GB tier** (RTX discrete 32GB+, 64K ctx, q8_0 KV, DFlash draft) and a 32GB tier (48GB+, 262K). The card has **no --n-cpu-moe/partial-offload guidance**. Everything uses `-ngl 999`. — [HF model card](https://huggingface.co/nvidia/NVIDIA-Nemotron-3.5-Bolt-74B-A9B-24GB-GGUF); the same doubt about a real 24 GB fit is raised by [SmartScope](https://smartscope.blog/en/blog/nemotron-3-5-bolt-24gb-gpu-compatibility-ollama-2026/)
- Vendor benchmarks (Bolt IQ2 vs BF16 competitors): Terminal-Bench 2.1 **50.98** (Qwen3.6-35B-A3B 44.38, Gemma4 26B-A4B 37.22), Tau3 Banking 17.01 (10.52), GPQA-D 86.43 (83.40), BrowseComp 66.75 (48.74), MCP-Atlas 52.20 (51.2 measured by NVIDIA; Qwen's official figure is 62.8), PinchBench 87.82 (Qwen 88.07). No SWE-bench, BFCL, LCB or Aider. — [HF model card](https://huggingface.co/nvidia/NVIDIA-Nemotron-3.5-Bolt-74B-A9B-24GB-GGUF)
- Third-party memory estimate: ~21.1 GB at 8K ctx, ~22.6 GB at 262K with q8_0 KV (only the attention layers have KV, Mamba state is fixed). — [runaihome.com](https://runaihome.com/blog/nemotron-35-bolt-74b-24gb-local-ai-2026/). Artificial Analysis Intelligence Index 21 (per the same article, unverified).
- Tool calling: `--jinja`, OpenAI `tools`/`tool_choice`, reasoning via `enable_thinking` (+ `medium_effort`). — [HF model card](https://huggingface.co/nvidia/NVIDIA-Nemotron-3.5-Bolt-74B-A9B-24GB-GGUF)

**Qwen3.8-27B (dense)**
- Open weights 14 Aug 2026, 27B **dense**, VLM, 262K ctx (to 1M), Apache-2.0, thinking levels xhigh/medium/low. — [Gigazine](https://gigazine.net/gsc_news/en/20260817-qwen3-8-27b/)
- Vendor: Terminal-Bench 2.1 **73.0** (Qwen3.6-27B 63.4, Opus 4.6 Max 78.2), SWE-bench Pro **61.7** (3.6-27B 53.5), OSWorld 84.3, IFBench 79.5. SWE-bench Verified is not published. — [Gigazine](https://gigazine.net/gsc_news/en/20260817-qwen3-8-27b/); SWE-Pro ran on the Claude Code harness, with no independent verification — [search summary of powerdrill/benchlm](https://benchlm.ai/models/qwen3.8-27b)
- Unsloth GGUF: UD-Q3_K_XL **13.4 GB**, UD-IQ3_XXS ~11.9 GB (secondary sources, check against HF). — [pasqualepillitteri.it](https://pasqualepillitteri.it/en/news/11335/qwen3-8-27b-run-local-16gb-lm-studio-unsloth); [orcarouter](https://www.orcarouter.ai/blog/qwen-3-8-27b-unsloth). LM Studio says the minimum quant needs ~17 GB RAM. — [Gigazine](https://gigazine.net/gsc_news/en/20260817-qwen3-8-27b/)
- No Qwen3.8 MoE in the 35B-A3B class was found. The only open MoE is Qwen3.8-2.4T-A95B (data center). — [Gigazine](https://gigazine.net/gsc_news/en/20260817-qwen3-8-27b/)

**Qwen3-Coder-Next 80B-A3B**
- 80B MoE / 3B active, 256K ctx, agentic coding. Unsloth publishes Dynamic GGUFs with `-ot` / `--n-cpu-moe` recipes. — [Unsloth docs](https://unsloth.ai/docs/models/tutorials/qwen3-coder-next.md)
- Community: IQ2_XXS/IQ3_XXS with `--n-cpu-moe 10/18`, ~22 t/s (hardware not stated). — [indras-mirror blog](https://indras-mirror-website.pages.dev/blog-qwen3-coder-next-iq2-local.html). Qwen3-Next-80B-A3B ~22 tok/s on a 16 GB card. — [reddit snapshot](https://reddit.sentinel-team.org/posts/1rei65v/snapshots/2026-02-25T19%3A22%3A50.330313Z)

**Gemma 4 26B-A4B**
- 25.2B total / 3.8B active. Unsloth UD-IQ2_XXS 9.88 GB, bartowski Q4_K_M 17.04 GB, Google QAT Q4_0 ~14.4 GB + 1.2 GB mmproj. — [search summary: unsloth](https://huggingface.co/unsloth/gemma-4-26B-A4B-it-GGUF), [bartowski](https://huggingface.co/bartowski/google_gemma-4-26B-A4B-it-GGUF), [wz-it.com](https://wz-it.com/knowledge/ki/gemma-4-vllm-w4a16-statt-gguf/)
- April 2026 report: on llama.cpp + Unsloth IQ4_XS the model failed to call tools ~95% of the time and looped (anecdotal, fix status unknown). — [reddit snapshot](https://reddit.sentinel-team.org/posts/1sihwo8/snapshots/2026-04-17T23%3A20%3A42.645991Z)
- In NVIDIA's comparison, Gemma4 26B-A4B BF16 is clearly below Qwen3.6-35B-A3B on Terminal-Bench 2.1 (37.22 vs 44.38). — [HF Nemotron card](https://huggingface.co/nvidia/NVIDIA-Nemotron-3.5-Bolt-74B-A9B-24GB-GGUF)

**Other names from the brief**
- Nemotron 3.5 Lightning 30B-A3B exists (BF16 Terminal-Bench 2.1 23.46, well below Qwen3.6). — [HF Nemotron card](https://huggingface.co/nvidia/NVIDIA-Nemotron-3.5-Bolt-74B-A9B-24GB-GGUF). A Nemotron 3.5 Bolt E4B small model also exists. — [SmartScope](https://smartscope.blog/en/blog/nemotron-3-5-bolt-24gb-gpu-compatibility-ollama-2026/)
- Mentioned in aggregator lists: LFM2.5 8B-A1B (~5.5 GB, for agents/tool calling), gpt-oss-20b (~16 GB). — [search summary of guides, e.g. insiderllm](https://insiderllm.com/guides/best-local-coding-models-2026/)

### Inferences
- In the 3B-active MoE class that fits 17–18 GB, Qwen3.6-35B-A3B is the strongest candidate with verified numbers. Nemotron Bolt is "smarter" on agentic benchmarks (Terminal-Bench 2.1 +6.6 pts over Qwen3.6 BF16), but it physically does not fit this machine.
- Qwen3.8-27B dense is probably the smartest model that fits as a file (Terminal-Bench 2.1 73 vs 44–51), but its speed on this hardware rules it out for agentic loops (see section 2).

### Gaps
- I did not check GLM-4.x Air / GLM-5 small, Ling/Ring, ERNIE, Hunyuan, Kimi/MiniMax small variants, Seed-OSS-36B, Mistral/Devstral 2026, or the gpt-oss-120b file size (I assume ~60+ GB MXFP4, so it is out, but I did not verify that this session) because of the tool-call cap.
- No Qwen3.6-27B GGUF sizes were found. The Qwen3.8-27B architecture (hybrid DeltaNet or not) is unconfirmed. There is no confirmed Qwen3.8 MoE.
- BFCL and Aider polyglot scores were found for none of the candidates.

---

## 2. What is the largest model that really fits, and how fast is it?

### Takeaway
On a 17–18 GB budget, the largest sensible choice is **Qwen3.6-35B-A3B at UD-IQ3_XXS (13.2 GB)** or UD-IQ4_XS (16.5 GiB, borderline), with experts offloaded to DDR4 via `--n-cpu-moe`. Expect roughly **15–30 tok/s generation** (estimate). Nemotron Bolt (22 GB) and anything 70B+ would have to swap through mmap and would thrash.

### Cited Findings
- 8 GB laptop GPU (RTX 3070 mobile) + Qwen3.6-35B-A3B UD-IQ4_XS (16.5 GiB), `--n-cpu-moe 28`: **~33 tok/s** (4 tok/s without it). — [windowsforum](https://windowsforum.com/news/llama-cpp-n-cpu-moe-run-qwen3-35b-faster-on-8gb-rtx-laptop-gpus.449607/)
- RTX 3060 12 GB + 32 GB **DDR4**: ~38 tok/s with the experts of 24 of 40 layers in RAM. — [insiderllm](https://insiderllm.com/guides/best-way-run-qwen-3-6-35b-moe-locally/)
- RTX 3090 24 GB, full GPU, 65K ctx: 81–102 tok/s (reference point). — [aminrj.com](https://aminrj.com/posts/llamacpp-qwen36-35b/)
- **llama.cpp mmap with a model larger than RAM:** an 18.4 GB dense model on a 16 GB machine ran at **0.017 tok/s** (thrashing). — [search summary, mac-code](https://github.com/Turcy76/mac-code). With MoE, only the active experts are read. Gemma 4 26B-A4B on an 8 GB Jetson with an mmap `--cpu-moe` baseline ran at 1.94 tok/s; an expert-streaming fork reached 2.95 tok/s. — [dev.to](https://dev.to/anyesh/running-a-26b-moe-on-an-8-gb-jetson-by-streaming-experts-from-ssd-2jpf). An upstream proposal for expert-aware SSD streaming notes that llama.cpp keeps all experts in memory. — [llama.cpp discussion #27149](https://github.com/ggml-org/llama.cpp/discussions/27149)
- MoE models far larger than RAM: typically 1–3 tok/s on NVMe. Dense spillover is far worse. — [same discussion/summary](https://github.com/ggml-org/llama.cpp/discussions/27149)

### Inferences (estimates, not measured on this HW)
- **How mmap works:** llama.cpp mmaps the GGUF. Weights offloaded to the GPU are copied into VRAM, and the CPU part lives in the page cache. When the CPU part plus KV exceeds free RAM, the kernel evicts pages and re-reads them from disk on every token. For MoE that means random reads of the experts actually being used. Viable only as an emergency fallback (1–3 tok/s), and useless for agentic coding with tens of thousands of prompt tokens. Practical rule: **the CPU part of the file must fit in free RAM**. A small overshoot (hundreds of MB of rarely used experts) is tolerable. GBs are not.
- **Nemotron Bolt on this HW:** 22.02 GB file + KV + buffers ≈ 23 GB against a 17–18 GB budget, so ~5 GB would be streamed from SSD every token, with 9B active (~2.7 GB read per token at ~2.4 b/w). Estimate ≤2–3 tok/s plus a degraded desktop. **Not viable.** It would also need Vulkan/ROCm support for the new Q2_0 tensor type, which is unverified.
- **Qwen3.6-35B-A3B memory (estimate):** KV only for 10 attention layers. If the config matches Qwen3-Next (2 KV heads × head_dim 256, unverified for 3.6), KV is ≈ 20 KB/token at f16, so **16K ≈ 0.33 GB, 32K ≈ 0.65 GB, 64K ≈ 1.3 GB** (half that with q8_0 KV). The DeltaNet state is a few tens of MB. Compute buffers on Vulkan are ~0.5–1 GB.
  - UD-IQ3_XXS 13.2 GB + 64K KV q8_0 (~0.65) + buffers (~0.8) ≈ **14.7 GB**. Fits comfortably with headroom.
  - UD-IQ4_XS 16.5 GiB (~17.7 GB) + 32K KV ≈ 19 GB. **Over budget**, so mmap would page cold experts out. Risky.
  - UD-Q3_K_XL 16.8 GB + 32K KV + buffers ≈ 18.3 GB. **Borderline**. Only with the desktop trimmed back.
- **Speed on THIS HW (estimate):** 3B active at ~3.2 b/w ≈ 1.2 GB read per token. With about half the experts in DDR4 (effective ~35–40 GB/s), the CPU side reads ~0.6–0.8 GB/token, a theoretical ceiling of ~45 tok/s. The RX 6500 XT has only a 64-bit bus (~144 GB/s), Vulkan is slower than CUDA, and a 6-core CPU limits compute. Realistically **~15–30 tok/s generation**. **Prompt processing is the bigger problem for agents:** with experts in RAM and PCIe 3.0 x4 (~3.5 GB/s), large prompt batches run either on the CPU or by copying experts over PCIe. Expect PP in the low hundreds of tok/s, so a 30K-token context can take minutes to load the first time. Prompt caching (`--cache-reuse`, slot save) is essential.
- **Qwen3.8-27B dense on this HW:** UD-Q3_K_XL 13.4 GB, of which ~6 GB goes to VRAM and ~7 GB to DDR4. Every token reads the whole CPU part (~7 GB / ~40 GB/s), so **≈ 4–6 tok/s**, with PP even worse. Too slow for an agentic loop (it would suit a "slow planner/reviewer" role at most). IQ3_XXS 11.9 GB only improves this slightly.
- **Qwen3-Coder-Next 80B-A3B:** even UD-IQ2_XXS is probably ~25 GB (estimate from 80B × ~2.4 b/w, no file size verified), which does not fit 17–18 GB. Out.

### Gaps
- No measurements exist for AMD Vulkan + 8 GB + DDR4 + `--n-cpu-moe`. Every tok/s number above is extrapolated from NVIDIA setups.
- The exact Qwen3.6-35B-A3B config (KV heads, head_dim) was not verified on HF. The KV estimate is an inference.

---

## 3. 2-bit 70–110B MoE vs 4-bit 30–35B MoE for coding agents

### Takeaway
There is no clean apples-to-apples eval. The best available data point is Nemotron Bolt, a **2-bit QAT 74B-A9B that beats BF16 Qwen3.6-35B-A3B** on agentic benchmarks (Terminal-Bench 2.1 50.98 vs 44.38). It is a QAT/QAD model, though, not a post-hoc quant. For ordinary post-hoc quants, KLD rises sharply below 3 bits, and Unsloth recommends ≥IQ3_XXS for experts. On this machine the question is moot anyway: a 2-bit 70B+ model does not fit 17–18 GB.

### Cited Findings
- Unsloth KLD table (Qwen3.5 ~35B-A3B MoE, inferred from the sizes): IQ2_XXS 9.09 GB mean KLD **0.1846**, Q2_K_XL 12.04 GB 0.097, **IQ3_XXS 13.12 GB 0.0501**, Q3_K_XL 16.06 GB 0.0308, Q4_K_M 18.49 GB 0.0192, Q4_K_XL 19.17 GB 0.0137. — [Unsloth Qwen3.5 GGUF benchmarks](https://unsloth.ai/docs/models/qwen3.5/gguf-benchmarks)
- Unsloth: 2-bit degrades noticeably more than 3-bit. ffn_* experts at ~IQ3_XXS are the best size/KLD trade-off. attn_* and ssm_out are sensitive. Prefer Q4_K over MXFP4. — [Unsloth](https://unsloth.ai/docs/models/qwen3.5/gguf-benchmarks)
- Counter-example: on MiniMax-M2.5, Unsloth IQ2_XXS beat AesSedai IQ3_S on LiveCodeBench v6 and MMLU-Pro while being 11 GB smaller. PPL/KLD did not predict real results. — [Unsloth](https://unsloth.ai/docs/models/qwen3.5/gguf-benchmarks)
- Unsloth Dynamic 3-bit DeepSeek V3.1 scored 75.6% on Aider polyglot, and Unsloth argues KLD/Aider beat perplexity as metrics. — [Unsloth Dynamic docs](https://unsloth.ai/docs/basics/unsloth-dynamic-2.0-ggufs)
- Nemotron Bolt IQ2 (QAD) vs Qwen3.6-35B-A3B BF16: TB2.1 50.98 vs 44.38, Tau3 17.01 vs 10.52, PinchBench 87.82 vs 88.07. — [HF Nemotron card](https://huggingface.co/nvidia/NVIDIA-Nemotron-3.5-Bolt-74B-A9B-24GB-GGUF)

### Inferences
- With natively trained 2-bit (QAT/QAD), a large MoE wins. With post-hoc 2-bit of a large MoE, the result is closer to a draw and model-specific (the MiniMax case). Expect higher variance in tool-call formatting (JSON errors, loops) at IQ2.
- For this HW, the relevant comparison is within one model: Qwen3.6-35B-A3B IQ3_XXS (KLD ~0.05) vs IQ2 (~0.18) vs Q3_K_XL (~0.03). **IQ3_XXS is the sweet spot**. Q3_K_XL is a small quality gain at +3.6 GB, which is right at the budget limit.

### Gaps
- No direct eval exists of a "2-bit 80B post-hoc MoE vs 4-bit 35B MoE" on SWE-bench/Aider. The Unsloth KLD table is for Qwen3.5, not 3.6.

---

## 4. llama.cpp support and tool-call templates

### Takeaway
Qwen3.6-35B-A3B: mature support (qwen35moe/hybrid DeltaNet, Unsloth template fixes). Run with `--jinja`. Nemotron Bolt: brand new (build ≥ 15 Oct 2026, new Q2_0 type, early bugs). Gemma 4 26B-A4B: reported tool-call loops.

### Cited Findings
- Qwen3.6: `qwen3_coder` tool parser, Unsloth fixes for developer role and nested-object tool calls, thinking on by default, `preserve_thinking` for agents. — [HF](https://huggingface.co/unsloth/Qwen3.6-35B-A3B-GGUF). Unsloth also mentions a tool-calling chat template fix that affects all quant uploaders (Qwen3.5). — [Unsloth](https://unsloth.ai/docs/models/qwen3.5/gguf-benchmarks)
- Nemotron Bolt: PR #30520 (15 Oct 2026), LM Studio lacks Q2_0, Metal segfault #30610, `--jinja` required. — [runaihome](https://runaihome.com/blog/nemotron-35-bolt-74b-24gb-local-ai-2026/), [HF](https://huggingface.co/nvidia/NVIDIA-Nemotron-3.5-Bolt-74B-A9B-24GB-GGUF)
- Gemma 4 26B-A4B: ~95% tool-call failures/loops on llama.cpp + IQ4_XS (April 2026, anecdote). — [reddit snapshot](https://reddit.sentinel-team.org/posts/1sihwo8/snapshots/2026-04-17T23%3A20%3A42.645991Z)

### Inferences: recommended config for this machine (untested)
```
llama-server -m Qwen3.6-35B-A3B-UD-IQ3_XXS.gguf --jinja \
  -ngl 99 --n-cpu-moe <tune ~20-28> -c 65536 \
  -fa on --cache-type-k q8_0 --cache-type-v q8_0 \
  -t 6 --no-mmproj (or omit the mmproj, saves VRAM/RAM) \
  --cache-reuse 256
```
- Tune `--n-cpu-moe` downward until VRAM is full (~7.3 GB used). Watch `free -h` so RAM doesn't hit swap.
- Fallback when RAM is tight: UD-Q2_K_XL (12.3 GB) or IQ2_M (11.5 GB), at a cost in quality (KLD ~0.1).
- "Smart but slow" alternative for planning/review: Qwen3.8-27B UD-IQ3_XXS (~11.9 GB) at ~4–6 tok/s (estimate).

### Gaps
- Status of Vulkan (RADV) / ROCm on the RX 6500 XT (gfx1034, not officially supported by ROCm) for hybrid DeltaNet kernels and the Q2_0 type was not verified.
- No current (Nov 2026) llama.cpp issues on Qwen3.6 tool calling were found.
