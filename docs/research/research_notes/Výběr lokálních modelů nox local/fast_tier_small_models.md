# Fast-tier small local LLMs for nox-local (1–5B dense / ~1–3B active MoE), as of Nov 2026

Target hardware: AMD RX 6500 XT, 8 GB VRAM (memory bandwidth ~144 GB/s), 16 GB RAM, llama.cpp Vulkan backend, OpenAI-compatible `llama-server --jinja`. Baseline already in catalog: Qwen3.5-4B UD-Q4_K_XL (measured on this GPU: 37 tok/s gen, 685 tok/s pp512) and Qwen3.5-9B.

Research budget note: ~20 tool calls. Primary sources were HF model cards (Qwen, Google, NVIDIA), NVIDIA HF blog, llama.cpp GitHub issues (checked via `gh`), and one reproducible third-party tool-calling eval. r/LocalLLaMA was reachable only through mirrors and aggregator blogs; numbers from aggregators are flagged.

---

## Q1. Which small models released in 2026 are best at tool calling and coding?

### Takeaway
No newer small model clearly beats Qwen3.5-4B for this tier. It is still the best-documented 1–5B model for both tool use (TAU2 79.9) and coding (LiveCodeBench v6 55.8). There is no Qwen3.6 or Qwen3.8 at 4B or below. The real alternatives are NVIDIA Nemotron 3 Nano 4B (hybrid Mamba, very small KV cache, strong in one independent tool-call test), Gemma 4 E4B (better multimodal, clearly weaker agentic score), and Qwen3.5-2B or LFM2.5-1.2B for the "ultra-fast, trivial tasks" sub-tier. JetBrains Mellum2.1 (12B-A2.5B MoE, Oct 2026) is the strongest small-active coding model on paper, but it does not fit fully in 8 GB with 32K context.

### Cited Findings

**What exists (and what doesn't)**
- Qwen3.5 small dense models (0.8B, 2B, 4B, 9B) were released as open weights on 2026-03-02 — [try-qwen-ai release history](https://try-qwen-ai.com/updates/release-history/); [llm-stats](https://llm-stats.com/models/compare/qwen3.5-397b-a17b-vs-qwen3.5-4b)
- Qwen3.6 open weights shipped only as 27B dense and 35B-A3B MoE (April 2026), with no 4B variant — [InsiderLLM Qwen guide](https://insiderllm.com/guides/qwen-models-guide/); [Ollama qwen3.6](https://ollama.com/library/qwen3.6)
- "Qwen3.8": one guide says there is a 27B open release (Aug 2026), and Wikipedia mentions a previewed Qwen3.8-Max at 2.4T. Nothing at the small sizes. The sources disagree on naming. — [Wikipedia: Qwen](https://en.wikipedia.org/wiki/Qwen); [promptquorum](https://www.promptquorum.com/local-llms/best-local-llms-for-coding)
- Nemotron 3 Nano 4B was published 2026-03-17. It is a hybrid Mamba-Transformer (42 layers: 21 Mamba, 4 attention, 17 MLP), pruned and distilled from Nemotron Nano 9B v2 — [NVIDIA HF blog](https://huggingface.co/blog/nvidia/nemotron-3-nano-4b)
- Gemma 4 E4B-it / E2B-it exist, with the Gemma 4 tech report dated "Jul 2" — [google/gemma-4-E4B-it](https://huggingface.co/google/gemma-4-E4B-it)
- JetBrains Mellum2.1: 12B total / 2.5B active MoE (64 experts, 8 active), Apache 2.0, 131,072 context, released around 2026-10-07/08 — [tech-insider](https://tech-insider.org/jetbrains-mellum-2-1-12b-moe-coding-agents-2026/); [nocode.mba tracker](https://www.nocode.mba/ai-release-tracker/releases/jetbrains-mellum2-1)
- LFM2.5-1.2B-Instruct (Liquid AI) replaces the deprecated LFM2-1.2B-Tool for tool calling — [Liquid docs](https://docs.liquid.ai/lfm/models/lfm2-1.2b-tool)
- MiniCPM5-2B (Sept 2026) exists. Its tool calling goes through SGLang/vLLM parsers, and OpenBMB recommends SGLang for tool calling (no llama.cpp guidance) — [buildfastwithai review](https://www.buildfastwithai.com/blogs/minicpm5-2b-review)

**Candidate details**

1. **Qwen3.5-4B (baseline, confirm)**
   - Apache-2.0. 4B LM plus vision encoder (HF metadata says 5B). Native 262,144 context, extensible to ~1.01M with YaRN — [Qwen/Qwen3.5-4B](https://huggingface.co/Qwen/Qwen3.5-4B)
   - Thinking is ON by default. Disable it with `enable_thinking: False` in the chat template kwargs. The `/think` and `/nothink` soft switches are NOT supported — [model card](https://huggingface.co/Qwen/Qwen3.5-4B)
   - Benchmarks (4B vs 9B): BFCL-V4 50.3 vs 66.1, TAU2-Bench 79.9 vs 79.1, LiveCodeBench v6 55.8 vs 65.6, IFEval 89.8 vs 91.5, MMLU-Pro 79.1 vs 82.5 — [model card](https://huggingface.co/Qwen/Qwen3.5-4B)
   - Recommended tool parser in vLLM/SGLang is `qwen3_coder` (XML-style tool calls) — [model card](https://huggingface.co/Qwen/Qwen3.5-4B)
   - Sampling: thinking/coding temp 0.6, top_p 0.95, top_k 20, presence 0. Non-thinking general temp 0.7, top_p 0.8, top_k 20, presence 1.5 — [model card](https://huggingface.co/Qwen/Qwen3.5-4B)
   - Independent tool-call eval (LM Studio, Q4_K_M, 3.4 GB): ranked #1 of 13 models at 97.5% pass rate, 48 tok/s on an AMD AI Max+ 395. Weakest area was multi-tool (in one run it fired two calls at once and failed) — [jdhodges.com, Mar 19 2026](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)

2. **Qwen3.5-2B (ultra-fast sub-tier)**
   - BFCL-V4 43.6, TAU2 48.8, IFEval 78.6 (thinking) / 61.2 (non-thinking), MMLU-Pro 66.5 / 55.3. Non-thinking by default. The card warns it "is more prone to entering thinking loops" in thinking mode. 262K native context — [Qwen/Qwen3.5-2B](https://huggingface.co/Qwen/Qwen3.5-2B)
   - Qwen3.5-0.8B: BFCL-V4 25.3, TAU2 11.6. Too weak for agent use — [same card](https://huggingface.co/Qwen/Qwen3.5-2B)

3. **NVIDIA Nemotron 3 Nano 4B**
   - License is the NVIDIA Nemotron Open Model License ("ready for commercial use"), not Apache. Context up to 262K. Official Q4_K_M GGUF is 2.84 GB — [nvidia/NVIDIA-Nemotron-3-Nano-4B-GGUF](https://huggingface.co/nvidia/NVIDIA-Nemotron-3-Nano-4B-GGUF)
   - Reasoning-off benchmarks (Q4_K_M, BF16 in parentheses): IFEval-Prompt 81.5 (82.8), IFBench-Prompt 46.9 (43.2), RULER-128k 91.2 (91.1). No BFCL, tau2 or LiveCodeBench published — [GGUF card](https://huggingface.co/nvidia/NVIDIA-Nemotron-3-Nano-4B-GGUF)
   - Reasoning is toggled via the system prompt. The final RL stage targeted multi-turn conversational tool calling — [GGUF card](https://huggingface.co/nvidia/NVIDIA-Nemotron-3-Nano-4B-GGUF); [Unsloth guide](https://unsloth.ai/docs/models/nemotron-3)
   - Unsloth suggests temp 0.6 / top_p 0.95 for tool calling (1.0/1.0 for chat). It estimates ~5 GB total memory and offers a UD-Q4_K_XL quant — [Unsloth Nemotron 3 guide](https://unsloth.ai/docs/models/nemotron-3)
   - Independent eval: 95.0% tool-call pass rate (#2, tied), 4.2 GB file, 44 tok/s. Author calls it "best value" for mostly single-tool workflows — [jdhodges.com](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)
   - Speed claims: Q4_K_M runs at 18 tok/s on a Jetson Orin Nano 8GB, and NVIDIA claims the lowest TTFT in its class at long inputs (RTX 4070, llama.cpp) — [NVIDIA blog](https://huggingface.co/blog/nvidia/nemotron-3-nano-4b)

4. **Gemma 4 E4B-it (and E2B-it)**
   - Apache 2.0. 4.5B effective params, 8B with embeddings (Per-Layer Embeddings). 128K context. Text, image and audio input. Native function calling. Thinking is enabled with `<|think|>` in the system prompt — [google/gemma-4-E4B-it](https://huggingface.co/google/gemma-4-E4B-it)
   - E4B vs E2B: LiveCodeBench v6 52.0 vs 44.0, Codeforces ELO 940 vs 633, Tau2 (avg of 3) 42.2 vs 24.5, MMLU-Pro 69.4 vs 60.0. No BFCL published — [model card](https://huggingface.co/google/gemma-4-E4B-it)
   - Third-party small tests disagree. One blog has E4B at 75% function calling but 0% on 5+ turn multi-turn, and E2B at 80% / 70%. Another developer had E4B pass only 2 of 6 agent scenarios, including not calling any tool when asked to write a file — [aiexplorer blog](https://aiexplorer-blog.vercel.app/post/gemma-4-e2b-benchmark); [dev.to test](https://dev.to/vystartasv/google-said-it-had-native-function-calling-i-tested-it-54i7)

5. **LFM2.5-1.2B-Instruct (Liquid AI)**
   - BFCLv3 49.1, measured with a custom Liquid handler. Artificial Analysis Tau2 is 10.8% (via aggregator, unverified). Tool calls use a Pythonic format inside `<|tool_call_start|>…<|tool_call_end|>`. The vLLM docs list both 128K and 32K context, which is unresolved — [vLLM recipe](https://docs.vllm.ai/projects/recipes/en/latest/LiquidAI/LFM2.5.html); [llmreference](https://www.llmreference.com/model/lfm-2.5-1.2b-instruct); [llmbase](https://llmbase.ai/models/liquidai/lfm2-5-1-2b-instruct/)

6. **JetBrains Mellum2.1-12B-A2.5B-Thinking (watch, not a strict fit)**
   - JetBrains-reported: SWE-bench Verified 47.0, LiveCodeBench v6 82.0 (Mellum2 Thinking: 69.4). Not independently verified, and one source has an apparent typo for the Mellum2 baseline — [tech-insider](https://tech-insider.org/jetbrains-mellum-2-1-12b-moe-coding-agents-2026/); [nocode.mba](https://www.nocode.mba/ai-release-tracker/releases/jetbrains-mellum2-1)
   - GGUF status is conflicting. One source says "coming soon". A later one says the official repo `JetBrains/Mellum2.1-12B-A2.5B-Thinking-GGUF` was created 2026-10-07 with a `llama-server -hf ...:Q4_K_M` example and files of 6.5–12.0 GiB — [toolhalla](https://toolhalla.ai/blog/can-you-run-mellum-2-1-locally); [itechguides](https://www.itechguides.com/?p=1045128)

**Older / superseded (do not add)**
- Phi-4-mini (3.8B, 2025): 57.5% tool-call pass rate, format compliance 6/8 or lower. Gemma 3 4B QAT: 55.0%. Both are well below Qwen3.5-4B and Nemotron 4B — [jdhodges.com](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)
- Qwen2.5-Coder-7B is still called the "FIM king" for autocomplete (88.4% HumanEval, ~5 GB Q4) in an Aug 2026 guide. It is a 2024 model, but relevant if nox-local wants true FIM autocomplete, which Qwen3.5 lacks — [InsiderLLM coding guide](https://insiderllm.com/guides/best-local-coding-models-2026/)
- Fine-tuned specialists (Hammer 2.1 7B: 20%, xLAM-2 8B: 15%) failed mostly on output format through an OpenAI-compatible server, not on understanding — [jdhodges.com](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)

### Inferences

**VRAM estimates.** Weights figures come from cited file sizes. KV figures are my estimates from architecture; verify with `llama-server` logs.

| Model | Quant / file | Est. VRAM @32K | Est. VRAM @64K | Notes |
|---|---|---|---|---|
| Qwen3.5-4B | UD-Q4_K_XL / Q4_K_M ≈ 2.9–3.4 GB | ~4–4.5 GB | ~5–5.5 GB | Hybrid Gated-DeltaNet: only ~1/4 of layers keep a KV cache, so KV is small. `-ctk q8_0 -ctv q8_0` shrinks it further. 64K fits easily. Skip the mmproj (vision) file to save VRAM. |
| Qwen3.5-2B | Q4_K_M ≈ 1.3–1.5 GB (est.) | ~2.5 GB | ~3 GB | Same hybrid architecture. |
| Nemotron 3 Nano 4B | Q4_K_M 2.84 GB (official) | ~3.5–4 GB | ~4–4.5 GB | Only 4 attention layers, so KV is tiny. Mamba state is constant-size. Best long-context-per-GB of the group. |
| Gemma 4 E4B | Q4_K_M ≈ 4.5–5 GB (est., 8B incl. PLE) | ~5.5–6.5 GB | ~6.5–7.5 GB | PLE embeddings can stay on CPU in some runtimes. Fits at 32K, tight at 64K. 128K max context. |
| LFM2.5-1.2B | Q4_K_M ≈ 0.8 GB (est.) | ~1.5 GB | ~2 GB | Short-conv hybrid, so KV is small. Context limit is unclear (32K vs 128K). |
| Mellum2.1 12B-A2.5B | Q4_K_M ≥ 6.5 GiB | >8 GB | >8 GB | Will NOT fit fully. It needs `--n-cpu-moe` expert offload to RAM, which disqualifies it from a VRAM-only fast tier. It may suit a separate "smart-but-cheap" tier. |

**Speed.** Token generation on this card is roughly bandwidth-bound. 144 GB/s against ~3 GB of weights gives a ~48 tok/s ceiling, and the measured 37 tok/s for Qwen3.5-4B is consistent with that. So:
- Nemotron 3 Nano 4B Q4_K_M (2.84 GB): probably similar or slightly faster, *if* the Vulkan SSM kernels are efficient (unverified).
- Qwen3.5-2B: roughly 55–70 tok/s (estimate).
- LFM2.5-1.2B: roughly 80–100+ tok/s (estimate).
- Gemma 4 E4B: likely slower than Qwen3.5-4B, because more bytes are read per token.

**Ranking for the fast tier:**
1. Qwen3.5-4B in non-thinking mode, kept as default.
2. Nemotron 3 Nano 4B as the A/B challenger, for its long context, tiny KV and fast prefill.
3. Qwen3.5-2B for commit messages and summaries.
4. Gemma 4 E4B only if image input is needed.
5. LFM2.5-1.2B for routing and classification only.

For coding quality at 4B, Qwen3.5-4B's LCB 55.8 beats Gemma 4 E4B's 52.0. Nemotron 4B has no published coding score.

### Gaps
- No official BFCL/tau2/LiveCodeBench for Nemotron 3 Nano 4B. The NVIDIA blog's table is an image, so it could not be extracted.
- No BFCL for Gemma 4 E2B/E4B, and no LiveCodeBench for Qwen3.5-2B.
- No Aider polyglot or SWE-bench Verified figures for any 1–5B 2026 model. Leaderboards don't list this size class.
- MiniCPM5-2B, Granite 4.x small, SmolLM3-3B and Ministral 3 3B could not be evaluated within budget. No 2026 benchmark data for them was found in the searches made.
- Exact GGUF sizes for the Qwen3.5-2B, Gemma 4 E4B and LFM2.5 quants were not fetched. The figures above are estimates.

---

## Q2. Which reliably emit well-formed tool calls through llama.cpp `llama-server --jinja` (MCP-compatible)?

### Takeaway
Qwen3.5 works but has real llama.cpp edge cases. The worst is tool calls emitted *inside the thinking block*, which is still open (#20837) and was reported on Qwen3.5-9B. The practical fix for nox-local is to run the fast tier with thinking disabled, use Unsloth GGUFs (fixed templates) and a recent llama.cpp build, and set max_tokens above the reasoning budget plus the largest tool call. Nemotron 4B tool calling through llama.cpp is undocumented by NVIDIA. LFM2.5 uses a Pythonic tool format that depends on llama.cpp having a matching parser (unverified).

### Cited Findings
- llama.cpp issue #20260 ("peg-native chat format parser fails when model outputs text before `<tool_call>`", Qwen3.5-35B-A3B Unsloth GGUF, thinking + tools): opened 2026-03-09, **CLOSED 2026-09-12** — [ggml-org/llama.cpp#20260](https://github.com/ggml-org/llama.cpp/issues/20260) (status checked via `gh`)
- llama.cpp issue #20837 ("Qwen3.5 9B often prints tool calls in XML and stops when thinking is enabled — tool calls inside thinking block"): opened 2026-03-21, **still OPEN** as of 2026-11-07 — [ggml-org/llama.cpp#20837](https://github.com/ggml-org/llama.cpp/issues/20837)
- Community reports that Qwen3.5 tool calls broke around llama.cpp build b8227 (plain chat unaffected). Stock Qwen3.5 Jinja templates had an `|items` filter bug on tool arguments, and Unsloth GGUFs ship template fixes. These are community claims — [r/LocalLLaMA mirror, Mar 2026](https://reddit.sentinel-team.org/posts/1roylbe/snapshots/2026-03-13T23%3A00%3A09.503181Z); [r/LocalLLaMA mirror, Apr 2026](https://reddit.sentinel-team.org/posts/1sis1vn/snapshots/2026-04-17T23%3A20%3A42.645991Z)
- Newer llama.cpp builds derive the tool-call parser from the template automatically. Older builds may need `--chat-template-kwargs '{"tool_call_format":"json"}'` (for a custom Qwen 3.6 template). A truncated tool call shows up as "JSON parsing failed", so the output limit must exceed reasoning budget plus the largest tool call — [Moore2877/Qwen-Agentic-Chat-Template-llamacpp](https://huggingface.co/Moore2877/Qwen-Agentic-Chat-Template-llamacpp)
- Community-fixed Qwen templates exist as an alternative — [clinraus/Qwen-Fixed-Chat-Templates](https://huggingface.co/clinraus/Qwen-Fixed-Chat-Templates)
- One user found LM Studio 0.4.9 parsed Qwen3.5 XML tool calls more reliably than raw llama.cpp at the time (Apr 2026) — [r/LocalLLaMA mirror](https://reddit.sentinel-team.org/posts/1sis1vn/snapshots/2026-04-17T23%3A20%3A42.645991Z)
- The NVIDIA Nemotron 3 Nano 4B GGUF card shows only a basic `llama-server -hf nvidia/NVIDIA-Nemotron-3-Nano-4B-GGUF:Q4_K_M -c 0 -ngl 999`. It does not mention `--jinja` or a tool parser — [GGUF card](https://huggingface.co/nvidia/NVIDIA-Nemotron-3-Nano-4B-GGUF)
- In the LM Studio (llama.cpp-based) OpenAI-API test, the low scorers were attributed to chat-template incompatibility with custom tool formats, not model quality. Qwen3.5-4B and Nemotron Nano 4B handled the format well (97.5% / 95%) — [jdhodges.com](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)
- Qwen3.5-2B thinking mode is prone to loops — [Qwen3.5-2B card](https://huggingface.co/Qwen/Qwen3.5-2B)
- Gemma 4 E2B/E4B: when thinking is disabled, no empty thought block is produced (this matters for template parsing) — [Gemma 4 E4B card](https://huggingface.co/google/gemma-4-E4B-it)
- LFM2.5 emits Pythonic calls in `<|tool_call_start|>…<|tool_call_end|>`, and vLLM needs a dedicated parser — [vLLM LFM2.5 recipe](https://docs.vllm.ai/projects/recipes/en/latest/LiquidAI/LFM2.5.html)

### Inferences
- For nox-local's fast tier, the cleanest config is Qwen3.5-4B (or 2B), Unsloth UD GGUF, `--jinja`, `chat_template_kwargs: {"enable_thinking": false}` per request, and a llama.cpp build newer than 2026-09-12 so the #20260 fix is included. With thinking off, the open #20837 path (tool calls inside `<think>`) is avoided.
- nox-local should keep a tolerant fallback parser. If `tool_calls` is empty but content contains `<tool_call>`/`<function=` XML, parse it client-side. This covers #20837-style failures.
- Nemotron 4B and Gemma 4 E4B need a local smoke test (single call, parallel calls, multi-turn, malformed-args) before being trusted with MCP.

### Gaps
- Could not confirm whether current llama.cpp has native parsers for Nemotron-H (Nemotron 3 Nano) tool calls, Gemma 4's function-call tokens, or LFM2.5's Pythonic format. Their llama.cpp GitHub issues were not searched within budget.
- No source tested these models specifically through raw `llama-server --jinja` plus an MCP client. The jdhodges eval used LM Studio's API layer.

---

## Q3. Reported tokens/second on low-end GPUs / Vulkan

### Takeaway
There are almost no published RX 6500 XT numbers. The user's own measurement (37 tok/s tg, 685 tok/s pp512 for Qwen3.5-4B Q4_K_XL) is the best data point available, and it beats the community figures for older AMD cards on Vulkan.

### Cited Findings
- RX 580 (Polaris) + llama.cpp Vulkan (RADV): ~16 tok/s on Qwen3.5-4B Q4_K_M, while another user on the same card reports ~27 tok/s with bartowski Q6_K. Anecdotal — [r/LocalLLaMA mirror, Mar 2026](https://reddit.sentinel-team.org/posts/1rkky2n/snapshots/2026-03-04T15%3A10%3A50.627029Z)
- RX 9070 XT Vulkan: Qwen3 4B Q8_0 tg128 ≈ 102 tok/s — same thread / [llama.cpp Vulkan discussion #10879](https://github.com/ggml-org/llama.cpp/discussions/10879)
- RX 6600 + Vulkan: Gemma-3n-E4B run took ~18 s vs ~90 s on CPU (5x). No tok/s given — [VulkanIlm](https://gh.nn.ci/Talnz007/VulkanIlm)
- AMD AI Max+ 395 (LM Studio): Qwen3.5-4B Q4_K_M 48 tok/s, Nemotron Nano 4B 44 tok/s, Phi-4-mini 57 tok/s, Gemma 3 4B QAT 60 tok/s — [jdhodges.com](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)
- Nemotron 3 Nano 4B Q4_K_M: 18 tok/s on Jetson Orin Nano 8GB, up to 2x Nano 9B v2 — [NVIDIA blog](https://huggingface.co/blog/nvidia/nemotron-3-nano-4b)

### Inferences
- On the AI Max+ test, Nemotron 4B ran ~8% slower than Qwen3.5-4B (44 vs 48). Expect roughly ~33–37 tok/s on the RX 6500 XT. Its prefill and long-context TTFT should be better, which matters for agent turns with large context.
- Both 2B-class models should be roughly 1.5–2x faster in generation on this card. This is a bandwidth-scaling estimate, not a measurement.

### Gaps
- No RX 6500 XT / Navi 24 llama-bench results were found for any 2026 small model. nox-local should measure Nemotron 4B, Qwen3.5-2B and Gemma 4 E4B itself with `llama-bench -p 512 -n 128` at 32K depth.
- Vulkan backend performance for Mamba/SSM layers (Nemotron-H) and Gated DeltaNet (Qwen3.5) is unverified relative to CUDA.

---

## Q4. Community consensus on the best small model for agentic coding right now

### Takeaway
Community and guide consensus in 2026: "agentic coding" really starts at Devstral Small 2 (24B) or Qwen3.6-27B, and Qwen3.5-9B is the 8 GB pick. Inside 1–5B, Qwen3.5-4B is the default recommendation for tool calling, with Nemotron 3 Nano 4B the most-cited alternative. Small models are seen as good for bounded tasks (single tool calls, summaries, commit messages), not long multi-step agent loops.

### Cited Findings
- Qwen3.5-4B is called the best overall tool-calling pick and Nemotron Nano 4B the best value. The author found "no meaningful correlation between model size and tool calling quality" — [jdhodges.com](https://www.jdhodges.com/blog/local-llms-on-tool-calling-2026-pt1-local-lm/)
- An Aug 2026 guide picks Qwen3.5 9B for the 8 GB chat/coding tier (~6.6 GB Q4) and Qwen2.5-Coder 7B for FIM autocomplete. It recommends keeping 4–8K context for coding on 8 GB — [InsiderLLM, updated Aug 19 2026](https://insiderllm.com/guides/best-local-coding-models-2026/)
- Guides converge on Devstral Small 2 (24B, Mistral-reported 65.8% SWE-bench Verified) for 16 GB and Qwen3.6-27B for 24 GB. One guide lists Qwen3.5-9B as the best smaller candidate for bounded tasks, and notes Mellum2.1 (12B-A2.5B, Oct 8 2026) as a challenger — [kunalganglani](https://www.kunalganglani.com/blog/best-local-model-agentic-coding); [promptquorum](https://www.promptquorum.com/local-llms/best-local-llms-for-coding)
- Qwen3.5-4B is named the best coding model under 4 GB (LCB 55.8) — summarised from [bodegaone](https://www.bodegaone.ai/local-llms) / [promptquorum](https://www.promptquorum.com/local-llms/best-local-llms-for-coding) (aggregator; matches the official card)
- Small models are framed as best for bounded, high-volume, single-domain tasks. For 50+ tools, preselect a few tools first — [callsphere, May 2026](https://callsphere.ai/blog/llm-comparison-function-calling-tool-orchestration-small-models-may-2026.md)
- Gemma 4 small models look usable for simple single tool calls but weak at multi-step chaining (several small self-run tests) — [aiexplorer](https://aiexplorer-blog.vercel.app/post/gemma-4-e2b-benchmark); [dev.to](https://dev.to/vystartasv/google-said-it-had-native-function-calling-i-tested-it-54i7)

### Inferences
- Recommended fast-tier shortlist for nox-local, in priority order:
  1. **Qwen3.5-4B UD-Q4_K_XL, non-thinking.** Keep as default; confirmed as best-in-class.
  2. **Nemotron 3 Nano 4B Q4_K_M / UD-Q4_K_XL.** A/B challenger with the cheapest long context.
  3. **Qwen3.5-2B Q4_K_M/Q5_K_M, non-thinking.** Commit messages, file summaries, search-result triage.
  4. **Gemma 4 E4B Q4_K_M.** Only if image/screenshot input is wanted. Weaker tau2 (42.2 vs 79.9).
  5. **LFM2.5-1.2B.** Router/classifier experiments only.
  6. Optional, outside the fast tier: **Mellum2.1-12B-A2.5B** with expert offload, as a "cheap smart coder" tier once its GGUF is confirmed.
  7. Optional FIM: **Qwen2.5-Coder-1.5B/3B** for true fill-in-the-middle autocomplete (older 2024 model, but it has FIM tokens).
- Qwen3.5-9B (66.1 BFCL-V4, 65.6 LCB) remains the clear step up when quality matters. Keep it as the next tier and don't try to replace it with a 4B.

### Gaps
- Direct r/LocalLLaMA threads from Sep–Nov 2026 were not retrievable (only mirrors and aggregators surfaced). No community consensus post specifically about 1–5B agentic coding in that window was found.
- No Artificial Analysis or BFCL leaderboard snapshot from Nov 2026 was fetched for small models.
