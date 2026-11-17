# Medium and high tier local LLMs for nox-local (agentic coding, llama.cpp Vulkan, RX 6500 XT 8 GB + 16 GB RAM), November 2026

Hardware details from the coordinator (verified): Ryzen 5 5600GT (6C/12T, AM4) with dual-channel DDR4, about 51 GB/s at 3200 MT/s. The GPU sits on PCIe 3.0 x4. The desktop uses about 6 GB RAM at idle, so **about 17–18 GB is realistically left for model plus KV cache** (8 GB VRAM plus about 9–10 GB RAM). All fit judgements below assume this 17–18 GB budget, not 24 GB.

Research ran on 2026-11-07, using about 16 searches and fetches. Model cards were fetched directly for Qwen3.8-27B, Qwen3.6-35B-A3B and Qwen3.5-9B, plus the Unsloth Qwen3.6 guide. Everything else comes from search snippets or secondary sources and is marked as such.

## Q1: Which 2026 models exist and are best for agentic coding at these sizes? (includes evaluation of the current catalog: qwen3.5-9b, qwen3.6-35b-a3b, gemma-4-*, qwen3.8-27b)

### Takeaway
- **HIGH tier: keep qwen3.6-35b-a3b, run at a 3-bit quant.** It is the strongest coding/agent model whose per-token cost (3B active) suits DDR4 plus 8 GB VRAM. Its vendor-reported scores are SWE-bench Verified 73.4, Terminal-Bench 2.0 51.5 and MCPMark 37.0.
- **Qwen3.8-27B is the smartest model in range** (Terminal-Bench 2.1 73.0, SWE-bench Pro 61.7, LCB v6 90.3). However, it is dense: on this machine roughly half its layers would sit in DDR4, which makes it too slow for agent loops. Treat it as an opt-in "slow/max" tier, not the default HIGH.
- **MEDIUM tier: keep qwen3.5-9b.** It fits fully in VRAM, and its BFCL-V4 66.1 and TAU2 79.1 make it the community-default 8 GB agent model.
- **Gemma 4 should not serve as a coding-agent tier.** Gemma4-26B-A4B scores SWE-bench Verified 17.4 and Gemma4-31B scores 52.0 on Qwen's comparison table.
- **No official Qwen3.8 9B or 35B-A3B exists.** Only Qwen3.8-27B and Qwen3.8-2.4T-A95B were released.

### Cited Findings

**Qwen3.8 family (verify-existence result)**
- Qwen3.8-27B exists. It is dense, 27B parameters (28B BF16 in the safetensors listing), Apache-2.0, with a 262,144-token native context extensible to 1M via YaRN. It is a hybrid of 16 blocks × [3 × Gated DeltaNet→FFN + 1 × Gated Attention→FFN], 64 layers, hidden size 5120, trained with MTP, and has a vision encoder. — [HF Qwen/Qwen3.8-27B](https://huggingface.co/Qwen/Qwen3.8-27B)
- Qwen3.8-27B scores, vendor-reported: Terminal-Bench 2.1 (Terminus) 73.0, SWE-bench Pro 61.7, NL2Repo 42.3, DeepSWE 1.1 42.2, QwenSWEBench 79.0, IFBench 79.5, GPQA-D 89.2, HLE 30.8, LiveCodeBench v6 90.3. The card gives no SWE-bench Verified, tau2, BFCL or MCP figures. — [HF Qwen/Qwen3.8-27B](https://huggingface.co/Qwen/Qwen3.8-27B)
- Qwen3.8-27B modes: thinking is on by default and can be disabled per request with `enable_thinking`. `reasoning_effort` accepts xhigh (default), medium or low. `preserve_thinking` defaults to true. Recommended thinking sampling: temp 1.0, top_p 0.95, top_k 20, min_p 0. Recommended non-thinking sampling: temp 0.7, top_p 0.8, top_k 20, presence_penalty 1.5. — [HF Qwen/Qwen3.8-27B](https://huggingface.co/Qwen/Qwen3.8-27B)
- Release date is about August 14, 2026; secondary sources disagree between Aug 3, 14 and 15. — [GitHub QwenLM/Qwen3.8 (via search)](https://github.com/QwenLM/Qwen3.8), [datanorth.ai](https://datanorth.ai/news/alibaba-releases-qwen3-8-27b), [gigazine](https://gigazine.net/gsc_news/en/20260817-qwen3-8-27b/)
- The only official Qwen3.8 releases are 27B (2026-08-14) and 2.4T-A95B (2026-08-12). "Qwen3.8-9B" GGUFs on HF are third-party distills, for example empero-ai/Qwen3.8-9B-Distill-GGUF, not from the Qwen org. Users are requesting a 3.8 35B-A3B in HF discussions. — [GitHub QwenLM/Qwen3.8](https://github.com/QwenLM/Qwen3.8), [HF collections listing](https://huggingface.co/qujincheng/collections), [HF activity](https://huggingface.co/FlyDogDaDa/activity/all)
- One aggregator ranks Qwen3.8 27B as the #1 local coding model overall, but only for a 24 GB GPU or 32 GB Mac (Ollama size 18 GB). — [promptquorum](https://www.promptquorum.com/local-llms/best-local-llms-for-coding)

**Qwen3.6-35B-A3B (catalog HIGH)**
- 35B total, 3B active, 256 experts (8 routed + 1 shared active), 40 layers, hidden size 2048. Hybrid Gated DeltaNet/Gated Attention with MoE, MTP included, 262K native context (1.01M with YaRN), Apache-2.0. Released April 2026 (2026-04-16 per the Qwen GitHub listing) as the first open-weight Qwen3.6. — [HF Qwen/Qwen3.6-35B-A3B](https://huggingface.co/Qwen/Qwen3.6-35B-A3B), [GitHub QwenLM/Qwen3.8 listing (via search)](https://github.com/QwenLM/Qwen3.8)
- Vendor-reported table (columns: Qwen3.6-35B-A3B / Qwen3.5-27B / Gemma4-31B / Qwen3.5-35B-A3B / Gemma4-26B-A4B):
  - SWE-bench Verified: 73.4 / 75.0 / 52.0 / 70.0 / 17.4
  - SWE-bench Multilingual: 67.2 / 69.3 / 51.7 / 60.3 / 17.3
  - SWE-bench Pro: 49.5 / 51.2 / 35.7 / 44.6 / 13.8
  - Terminal-Bench 2.0: 51.5 / 41.6 / 42.9 / 40.5 / 34.2
  - TAU3-Bench: 67.2 / 68.4 / 67.5 / 68.9 / 59.0
  - MCPMark: 37.0 / 36.3 / 18.1 / 27.0 / 14.2
  - MCP-Atlas: 62.8 / 68.4 / 57.2 / 62.4 / 50.0
  - LiveCodeBench v6: 80.4 / 80.7 / 80.0 / 74.6 / 77.1
  
  — [HF Qwen/Qwen3.6-35B-A3B](https://huggingface.co/Qwen/Qwen3.6-35B-A3B)
- Unsloth total-memory guide (RAM+VRAM): 35B-A3B needs 17 GB at 3-bit, 23 GB at 4-bit, 30 GB at 6-bit. 27B needs 15 GB at 3-bit, 18 GB at 4-bit. MTP variants add about 1 GB. Quants offered: UD-Q2_K_XL, UD-Q4_K_XL (the tutorial default), UD-IQ4_NL_XL, Q4_K_M, plus mmproj-F16 for vision. — [Unsloth Qwen3.6 guide](https://unsloth.ai/docs/models/qwen3.6)
- Unsloth sampling for Qwen3.6: thinking at temp 1.0 (0.6 for precise coding), top_p 0.95, top_k 20; non-thinking at temp 0.7, top_p 0.8, presence_penalty 1.5. Thinking toggle: `--chat-template-kwargs '{"enable_thinking":false}'`. Also supports `preserve_thinking`. MTP speculative decoding: `--spec-type draft-mtp --spec-draft-n-max 2`, giving about 1.15–1.2× speedup for the MoE and about 1.4× for dense models. — [Unsloth Qwen3.6 guide](https://unsloth.ai/docs/models/qwen3.6)
- Unsloth's Qwen3.6 uploads improved nested-object parsing for tool calls and support the `developer` role for agentic tools. Known issue: low context length or certain backends can produce gibberish; the suggested fix is `--cache-type-k bf16 --cache-type-v bf16`. — [Unsloth Qwen3.6 guide](https://unsloth.ai/docs/models/qwen3.6)

**Qwen3.6-27B (dense sibling, superseded by 3.8-27B)**
- Secondary reports: SWE-bench Verified 77.2 and Terminal-Bench 2.0 59.3. The source notes no independent reproduction as of May 2026. Q4_K_M GGUF is about 16.8 GB. — [braindetox](https://braindetox.kr/en/posts/qwen_3_6_27b_local_2026.html), [dev.to](https://dev.to/jangwook_kim_e31e7291ad98/qwen3-coder-27b-dense-model-that-beats-397b-moe-2026-94p)

**Qwen3.5-9B (catalog MEDIUM)**
- 9B (10B in the BF16 listing), released February 2026 (2026-03-02 per the Qwen GitHub listing), Apache-2.0, vision encoder, 262K native context (1.01M with YaRN). 32 layers, hybrid of 8 × [3 DeltaNet + 1 Gated Attention], with Gated Attention using 4 KV heads × head_dim 256. — [HF Qwen/Qwen3.5-9B](https://huggingface.co/Qwen/Qwen3.5-9B)
- Benchmarks (Qwen3.5-9B vs GPT-OSS-20B vs Qwen3.5-4B): LiveCodeBench v6 65.6 / 74.6 / 55.8; TAU2-Bench 79.1 / – / 79.9; BFCL-V4 66.1 / – / 50.3. Against gpt-oss-20b it scores MMLU-Pro 82.5 vs 74.8, GPQA-D 81.7 vs 71.5, IFEval 91.5 vs 88.2, AA-LCR 63.0 vs 30.7. The card has no SWE-bench or Terminal-Bench figures. — [HF Qwen/Qwen3.5-9B](https://huggingface.co/Qwen/Qwen3.5-9B)

**Gemma 4 (catalog gemma-4-*)**
- Official sizes: E2B, E4B, 12B, 26B A4B and 31B, each with MTP draft models. Small models get 128K context and mid-size models 256K. — [Google Gemma docs (via search)](https://ai.google.dev/gemma/docs/core?authuser=0&hl=ja)
- Gemma4-26B-A4B: bartowski Q4_K_M is 17.04 GB. Google's QAT Q4_0 is about 14.4 GB plus a 1.2 GB mmproj. The Unsloth UD-IQ2_XXS is 9.88 GB and flagged as low quality. — [bartowski GGUF (via search)](https://huggingface.co/bartowski/google_gemma-4-26B-A4B-it-GGUF)
- Tool-call loops in LM Studio were traced to a bundled llama.cpp that lacked the Gemma-4-specific parser; a current llama-server fixed multi-step tool chains. This is anecdotal. — [r/LocalLLaMA snapshot](https://reddit.sentinel-team.org/posts/1sc4gui/snapshots/2026-04-09T16%3A11%3A00.655487Z)
- Agentic coding scores are weak (from Qwen's comparison table above): 26B-A4B gets SWE-V 17.4 and MCPMark 14.2; 31B gets SWE-V 52.0. — [HF Qwen/Qwen3.6-35B-A3B](https://huggingface.co/Qwen/Qwen3.6-35B-A3B)

**Alternatives**
- **Laguna XS 2.1** (Poolside, released Jul 2, 2026). 33B total / 3B active, 256K context, OpenMDW-1.1 license, vendor SWE-bench Verified 70.9 and Multilingual 63.1. — [promptquorum](https://www.promptquorum.com/local-llms/best-local-llms-for-coding)
  - Official GGUFs come only as BF16 and Q4_K_M (about 20 GB Ollama size; one aggregator estimates about 22.5 GB VRAM at Q4_K_M). Support was not in upstream llama.cpp at last check: you need to build from PR #25165. — [HF poolside/Laguna-XS-2.1-GGUF](https://huggingface.co/poolside/Laguna-XS-2.1-GGUF), [friendli](https://friendli.ai/models/Fdxyra/Laguna-XS-2.1), [willitrunai](https://willitrunai.com/zh/models/laguna-xs-2.1)
- **Devstral Small 2** (Mistral). 24B dense, Apache-2.0, 15 GB Ollama size, vendor SWE-bench Verified 65.8%. Recommended for "agentic work on 16 GB". — [promptquorum](https://www.promptquorum.com/local-llms/best-local-llms-for-coding); Unsloth's tool-calling guide uses it as its llama-server example — [Unsloth tool-calling guide](https://unsloth.ai/docs/basics/tool-calling-guide-for-local-llms)
- **gpt-oss-20b.** 21B total / 3.6B active MoE, described as "a solid 16 GB pick"; LiveCodeBench v6 74.6. — [promptquorum](https://www.promptquorum.com/local-llms/best-local-llms-for-coding), [HF Qwen/Qwen3.5-9B](https://huggingface.co/Qwen/Qwen3.5-9B)
- **GLM-4.7-Flash.** 30B MoE, about 12–14 GB VRAM at Q4 per one guide. In a 13-model local tool-calling test (March 2026) it tied for first at 95% but failed a simultaneous multi-tool case. — [pondhouse-data](https://www.pondhouse-data.com/blog/glm-4-7-flash-local-opencode), [jdhodges tool-calling test](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)
- **Nemotron 3 Nano 30B-A3B.** 85% in the same tool-calling test, weaker on multi-tool calls. In OpenCode it held about 100 tok/s as context grew, while GLM-4.7-Flash dropped from 80 to 20 tok/s. GLM followed the agent framework more precisely, including sub-agent delegation. — [jdhodges](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/), [Medium OpenCode test](https://medium.com/@christian.rute/opencode-local-llms-a-practical-test-with-glm-4-7-flash-and-nemotron-c3ba2cbda43f)
- **Qwen3-Coder-Next.** 80B / 3B active, 52 GB, Feb 2026. Too large for this budget. — [promptquorum](https://www.promptquorum.com/local-llms/best-local-llms-for-coding)

### Inferences

**Memory estimates.** KV figures are computed from the configs above and are not measured. Qwen hybrids keep standard KV only in the 1-in-4 Gated Attention layers; the DeltaNet layers hold a small constant state.

| Model | KV per token (f16) | KV at 32K (f16 / q8_0) | KV at 64K (f16 / q8_0) | Total estimate |
|---|---|---|---|---|
| Qwen3.5-9B | 8 attn layers × 4 KV heads × 256 dim × 2 (K,V) × 2 B ≈ 32 KB | ≈ 1.0 GB / ≈ 0.5 GB | ≈ 2.0 GB / ≈ 1.0 GB | Q4_K_M ~5.5–6 GB weights + KV + ~0.5–1 GB compute buffers → fits 8 GB at 32K, tight at 64K f16, OK at 64K with q8_0 KV. Consistent with a third-party figure of "6.96 GB VRAM at 32K" ([localaimaster](https://localaimaster.com/vram/best-coding-llm-8gb-vram)) |
| Qwen3.8-27B | Assumes 16 attn layers × 4 KV heads × 256 (KV-head count not confirmed for 3.8) ≈ 64 KB | ≈ 2 GB / ≈ 1 GB | ≈ 4 GB / ≈ 2 GB | 3-bit total about 15 GB (Unsloth figure for same-architecture 3.6-27B) + KV → about 16–17 GB at 32K q8_0. That is the entire budget, with about half the weights in DDR4 |
| Qwen3.6-35B-A3B | About 10 attention layers, KV-head count not verified on card; likely ≤ 32 KB | well under 1 GB / ~0.3–0.5 GB | ~1–1.5 GB / ~0.5–0.8 GB | 3-bit total about 17 GB per Unsloth. UD-IQ4_XS (16.5 GiB ≈ 17.7 GB file) plus KV overshoots the 17–18 GB budget, so **use a 3-bit quant (UD-Q3_K_XL / IQ3_XXS class), or Q2_K_XL if memory pressure appears** |

- **Qwen3.5-9B KV is cheap.** 64K context is affordable even in 8 GB VRAM with q8_0 KV.
- **Ranking for this machine.**
  - HIGH: Qwen3.6-35B-A3B at 3-bit.
  - HIGH alternatives: gpt-oss-20b (a smaller footprint leaves headroom for 64K+ context; benchmark data for agent tasks was not found) and GLM-4.7-Flash.
  - Opt-in MAX: Qwen3.8-27B at 3-bit, slow.
  - MEDIUM: Qwen3.5-9B.
- **Laguna XS 2.1 is attractive on SWE-V** (70.9 with 3B active), but its official Q4_K_M (~20 GB) exceeds the budget and it needs a non-mainline llama.cpp build. Watch it rather than adopt it now.
- **Devstral Small 2 (24B dense) has the same offload-speed problem** as Qwen3.8-27B, and lower scores than Qwen3.6-35B-A3B on SWE-V (65.8 vs 73.4).
- **Gemma 4 entries in the catalog can be dropped or demoted** to a general/vision role.

### Gaps
- No exact GGUF file sizes were found for Qwen3.6-35B-A3B UD-Q3_K_XL or IQ3, nor for Qwen3.8-27B quants. Check the HF file listings (unsloth/Qwen3.6-35B-A3B-GGUF, unsloth or bartowski Qwen3.8-27B-GGUF).
- The number of KV heads for Qwen3.6-35B-A3B and Qwen3.8-27B was not on the fetched cards, so the KV numbers above are estimates.
- No SWE-bench Verified, BFCL or tau2 figures exist for Qwen3.8-27B. Aider polyglot numbers for 2026 models were not found.
- No Gemma 4 12B coding or agent benchmarks were found. No 2026 Nemotron or Granite coding scores were found, and nothing on Seed-OSS in 2026.
- No independent (non-vendor) SWE-bench reproductions were found for these models.

## Q2: Dense vs MoE on this hardware (MoE with ~3B active and experts in RAM vs 9–14B dense in VRAM)

### Takeaway
On DDR4 at about 51 GB/s with 8 GB VRAM, **a ~3B-active MoE with experts offloaded via `--n-cpu-moe` is the only way to run ~35B-class quality at interactive speed**. Expect roughly 10–20 tok/s. Dense 24–27B with partial offload drops to low single digits, or below 1 tok/s. A 9B dense model fully in VRAM stays the fastest option, with the best prompt processing, which makes it the right MEDIUM.

### Cited Findings
- **XDA test, RTX 3070 Laptop with 8 GB VRAM**, Qwen3.6-35B-A3B at Unsloth UD-IQ4_XS (16.5 GiB). With `-ngl 99` and no expert offload it generated 4.02 tok/s. Adding `--n-cpu-moe 28` raised generation to 32.82 tok/s and prompt processing from 109.66 to 341.37 tok/s, with `-ctk q8_0 -ctv q8_0`. A dense model on the same setup managed 0.75 tok/s. System RAM type was not stated. — [XDA Developers](https://www.xda-developers.com/i-stopped-thinking-my-8gb-gpu-was-too-small-after-learning-to-offload-moe-llm-layers-to-ram/), [WindowsForum summary](https://windowsforum.com/news/llama-cpp-n-cpu-moe-run-qwen3-35b-faster-on-8gb-rtx-laptop-gpus.449607/)
- **A community imatrix repo suggests `-ngl 99 --n-cpu-moe 48`** for Qwen3.6-35B-A3B and reports about 9–13 tok/s on an i7-6700HQ with 48 GB DDR4 and a GTX 1070. This is third-party. — [DuoNeural GGUF](https://huggingface.co/DuoNeural/Qwen3.6-35B-A3B-Code-imatrix-GGUF)
- **Qwen3.6-27B dense on 16 GB cards: speed reports conflict.** One Reddit claim is about 40 tok/s with a trimmed 15.1 GB Q4_K_M. A cited measurement is about 25–26 tok/s at 65K. An estimate for a 16 GB RX 7600 XT is about 5 tok/s. — [braindetox](https://braindetox.kr/en/posts/qwen_3_6_27b_local_2026.html), [willitrunai](https://willitrunai.com/ja/can-run/qwen-3.6-27b-on-rx-7600-xt-16gb)
- **No DDR4-specific benchmark of 35B-A3B was found.** Generation speed depends heavily on RAM bandwidth. — [search summary of apxml/llamabuilds](https://apxml.com/models/qwen36-35b-a3b)

### Inferences
- **35B-A3B on this machine.** About 3B active at ~3.5–4.5 bpw means ~1.3–1.7 GB read per token. Attention and shared weights stay in VRAM, so roughly 1 GB per token comes from DDR4. At 51 GB/s peak (about 30–35 GB/s effective), the ceiling is about 25–30 tok/s; **a realistic expectation is about 12–20 tok/s**. The 6-core Zen 3 CPU may also limit it. That is slower than the XDA laptop (likely faster RAM) but usable.
- **Qwen3.8-27B at 3-bit.** About 12–13 GB of weights with only about 6 GB in VRAM means roughly 6–7 GB read per token from DDR4, so **about 4–6 tok/s at best**. Thinking mode adds thousands of tokens per turn, which makes agent loops painfully slow. Use it only as an opt-in "max quality, slow" tier.
- **PCIe 3.0 x4 (~3.9 GB/s) can hurt prompt processing** when llama.cpp copies CPU-resident expert weights to the GPU for large batches (op offload). Benchmark both with and without `--no-op-offload`, and with larger `-ub`, on the 35B-A3B. Long agent prompts (10–30K tokens of tool output) make prompt processing speed as important as generation speed.
- **Tuning `--n-cpu-moe`.** Raise it until VRAM fits with the chosen context. The model has 40 layers; values between about 28 and 40 bracket the community settings for 8 GB.
- **Qwen3.5-9B fully in VRAM** on the RX 6500 XT (~144 GB/s VRAM bandwidth; the GPU spec is from memory, not sourced here) should beat the MoE on prompt processing and match or beat it on generation, at clearly lower quality. That fits the MEDIUM role.
- **The RX 6500 XT runs at PCIe 4.0 x4 natively**, but the coordinator says the slot runs at 3.0 x4. Avoid any configuration that streams weights over PCIe every token, such as `-ngl` partial plus mmap thrashing.

### Gaps
- No measured numbers exist for an RX 6500 XT with the Vulkan backend running a 35B-A3B MoE. The user should run `llama-bench` with `--n-cpu-moe` sweeps.
- No data was found on Vulkan vs ROCm for MoE offload on RDNA2 (gfx1034).

## Q3: Which work reliably with llama.cpp --jinja tool calling and MCP clients?

### Takeaway
- **Qwen3.5/3.6 tool calling works on recent llama.cpp builds** (which auto-derive the XML tool-call parser from the template). There are recurring reports of JSON-argument parse failures and template bugs; community-hardened templates and recent GGUF re-uploads fix most of them.
- **Gemma 4 needs a llama.cpp build with its dedicated parser.**
- **Laguna XS 2.1 needs a PR build.**
- **GLM-4.7-Flash** and Devstral Small 2 have Unsloth tool-calling guides for llama-server.

### Cited Findings
- **Recent llama.cpp builds auto-derive the tool-call parser from the Qwen 3.6 template.** Older builds, such as old ik_llama, parse only Hermes-JSON; for those, use `--chat-template-kwargs '{"tool_call_format":"json"}'`. A truncated tool call shows up as "JSON parsing failed", so the max output tokens must cover reasoning plus the largest tool call. — [Moore2877 Qwen agentic template](https://huggingface.co/Moore2877/Qwen-Agentic-Chat-Template-llamacpp)
- **Reported error with Qwen:** "Failed to parse tool call arguments as JSON" caused by triple quotes inside tool-call JSON. It hit about 80% of sessions on llama.cpp b9159 with template v16, and template v19 mostly fixed it. — [froggeric fixed templates discussion](https://huggingface.co/froggeric/Qwen-Fixed-Chat-Templates/discussions/20)
- **The official Qwen Jinja template has XML vs JSON tool-call format mismatches.** The "buun" hardened template lists 25 fixes. — [besthub.dev](https://besthub.dev/articles/the-underrated-lifesaving-template-for-qwen-local-deployment-cbc65bddf654), [spiritbuun template](https://huggingface.co/spiritbuun/buun-Qwen3.6-chat_template)
- **Qwen3-Coder-Next tool-call errors (Feb 2026)** were fixed by llama.cpp PR #19324 plus GGUF re-uploads. — [r/LocalLLaMA snapshot](https://reddit.sentinel-team.org/posts/1qx4alp/snapshots/2026-02-06T08%3A30%3A23.666946Z)
- **Unsloth's Qwen3.6 GGUFs don't work in Ollama** because of the separate mmproj files; use llama.cpp-compatible backends. — [Unsloth Qwen3.6 guide (via search)](https://unsloth.ai/docs/models/qwen3.6)
- **Gemma 4 tool loops disappear on a current llama.cpp** that has the Gemma 4 parser. — [r/LocalLLaMA snapshot](https://reddit.sentinel-team.org/posts/1sc4gui/snapshots/2026-04-09T16%3A11%3A00.655487Z)
- **Laguna XS 2.1 needs llama.cpp PR #25165**; the example run uses `llama-server --jinja`. — [HF poolside/Laguna-XS-2.1-GGUF](https://huggingface.co/poolside/Laguna-XS-2.1-GGUF), [friendli](https://friendli.ai/models/Fdxyra/Laguna-XS-2.1)
- **Unsloth tool-calling guide:** covers GLM-4.7-Flash, Nemotron 3 Nano and Devstral 2 with llama-server. Sampler settings matter; for GLM-4.7-Flash it suggests temp 0.7, top_p 1.0 and min-p 0.01. — [Unsloth tool-calling guide](https://unsloth.ai/docs/basics/tool-calling-guide-for-local-llms)
- **OpenCode with llama.cpp:** both GLM-4.7-Flash and Nemotron-3-Nano-30B-A3B worked. GLM was more precise with the agent framework; Nemotron was faster. — [Medium](https://medium.com/@christian.rute/opencode-local-llms-a-practical-test-with-glm-4-7-flash-and-nemotron-c3ba2cbda43f)

### Inferences
- **For nox-local, pin a minimum llama.cpp build** and always run with `--jinja`. Optionally ship a vetted Qwen 3.6 chat template, such as froggeric v19+ or buun, via `--chat-template-file`.
- **Set the output token limit generously** to cover thinking plus the tool call.
- **Keep `preserve_thinking`** in mind for multi-turn agent use (more tokens, possibly better accuracy).
- **Consider `reasoning_effort=medium`/`low` on Qwen3.8-27B** to cut latency on slow hardware.
- **MCP:** Qwen3.6-35B-A3B's MCPMark 37.0 and MCP-Atlas 62.8 are the best MCP-specific numbers in range. Gemma4-26B-A4B scores far lower (14.2 / 50.0).

### Gaps
- No specific llama.cpp GitHub issue was found confirming which build fixed the Qwen3.6 XML tool parsing.
- No data on gpt-oss-20b harmony-format tool calling reliability in llama.cpp in 2026 was found in this pass.
- No Cline or Continue-specific compatibility reports were found. One source notes Continue.dev has had no update since June 19, 2026 — [promptquorum](https://www.promptquorum.com/local-llms/best-local-llms-for-coding).

## Q4: Community consensus (2026) for "best coding model for 8 GB VRAM" and "for 16–24 GB total memory"

### Takeaway
- **8 GB:** guides converge on Qwen3.5-9B Q4_K_M for chat and agent coding, with Qwen2.5-Coder-7B for FIM autocomplete.
- **16–24 GB:** guides name Qwen3.6-35B-A3B with MoE offload, gpt-oss-20b, Devstral Small 2 and GLM-4.7-Flash. Qwen3.8-27B is the 24 GB-GPU pick. Actual r/LocalLLaMA threads were scarcely surfaced; most evidence is guide/blog aggregation.

### Cited Findings
- **8 GB picks.** Qwen3.5-9B Q4_K_M is the most common pick; one guide says "6.96 GB VRAM at 32K", leaving little headroom. It is usable for agent loops, not for tab-complete. Qwen2.5-Coder 7B is the usual FIM choice. Falcon H1R 7B and Qwen3 8B appear as alternatives. — [localaimaster](https://localaimaster.com/vram/best-coding-llm-8gb-vram), [insiderllm](https://insiderllm.com/guides/best-local-coding-models-2026/), [lmsa.app](https://lmsa.app/blog/coding-locally-in-2026-the-best-lm-studio-models-for-your-8gb-vram-gpu/), [mayhemcode](https://www.mayhemcode.com/2026/06/best-local-llms-for-4gb-6gb-and-8gb.html)
- **promptquorum tiers (updated Oct/Nov 2026):**
  - 8 GB: Qwen3 8B
  - 16 GB: Devstral Small 2, or gpt-oss:20b
  - 24 GB GPU: Qwen3.8 27B, Laguna XS 2.1 or Qwen3-Coder 30B
  - Autocomplete: Codestral 22B
  
  — [promptquorum](https://www.promptquorum.com/local-llms/best-local-llms-for-coding)
- **The 8 GB laptop plus `--n-cpu-moe` setup is popularised as the way to run a 35B MoE "faster than I can read"** (XDA). — [XDA](https://www.xda-developers.com/i-stopped-thinking-my-8gb-gpu-was-too-small-after-learning-to-offload-moe-llm-layers-to-ram/)
- **Unsloth says the 35B-A3B GGUF runs on "22GB RAM" setups.** — [Unsloth Qwen3.6 guide (via search)](https://unsloth.ai/docs/models/qwen3.6)

### Inferences
- **Recommended nox-local catalog:**
  - **MEDIUM = qwen3.5-9b** (UD-Q4_K_XL or Q4_K_M, all layers on GPU, 32–64K with q8_0 KV).
  - **HIGH = qwen3.6-35b-a3b** (3-bit UD quant, `-ngl 99 --n-cpu-moe ~30–40`, 32K context with q8_0 KV, `--jinja`, MTP optional).
  - **Optional MAX/slow = qwen3.8-27b** (3-bit, partial offload, `reasoning_effort` medium). Label it "slow on this hardware".
  - **Drop gemma-4-26b/31b as coding tiers.**
  - **Watch:** Laguna XS 2.1 (once merged upstream and smaller quants exist), and any official Qwen3.8 small/MoE release.
- **The user's "24 GB total" is effectively 17–18 GB.** That rules out 4-bit 35B-A3B (Unsloth says 23 GB) and 4-bit Laguna XS (~20 GB), which some guides recommend for "16–24 GB".

### Gaps
- No direct r/LocalLLaMA thread content from Sept–Nov 2026 was retrieved, so the consensus is drawn from guides that may lag.
- Whether a Qwen3.8 35B-A3B or 9B is planned is unknown; there are only community requests.
