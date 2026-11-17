# Running local LLMs on RX 6500 XT 8 GB + 16 GB DDR4 with llama.cpp Vulkan for an agentic coding CLI (state: Nov 2026)

**Verified hardware (from coordinator):** Ryzen 5 5600GT (Cezanne APU, 6C/12T, AM4, Vega 7 iGPU), so **DDR4 dual-channel** (speed unknown, typically 3200 MT/s, about 51.2 GB/s theoretical and about 40–45 GB/s in practice) and **PCIe 3.0 only, so the RX 6500 XT runs at PCIe 3.0 x4** (about 3.9 GB/s theoretical, about 3.2–3.5 GB/s in practice). VRAM confirmed at 8 GB. The idle desktop uses about 6 GB RAM. Total hard limit is 24 GB (8 GB VRAM + 16 GB RAM).
**User baseline:** Qwen3.5-4B Q4_K_XL at 37 tok/s generation and 685 tok/s pp512 on this card.

Note: unless a line carries a source, it is my own arithmetic or inference and is labelled as such under "Inferences".

---

## 1. Measured speeds on RX 6500 XT and similar RDNA2 cards (Vulkan vs ROCm)

### Takeaway
Token generation on this card is memory-bandwidth-bound at about 100–106 GB/s effective, which is about 70–75% of the 144 GB/s peak. That means roughly **tok/s ≈ 100 / (model GB in VRAM)**: about 35–40 tok/s for 4B Q4, about 17–20 for 8–9B Q4, and about 11–13 for a 14B IQ3/Q3 that just fits. ROCm on gfx1034 is unsupported and not worth the effort. Use Vulkan (RADV).

### Cited Findings
- llama.cpp Vulkan scoreboard, Llama 2 7B Q4_0 (3.56 GiB), no FA: **RX 6500 XT pp512 255 t/s, tg128 27.8 t/s**. RX 6600 gives 758 / 50.6, RX 6600 XT gives 575 / 53.9, RX 6650 XT gives 1030 / 62.1. With FA, RX 6600 gives 805 / 53.2 and RX 6650 XT gives 1089 / 64.5. No RX 6400 row exists. The scoreboard does not list driver, VRAM variant or date. The 6500 XT row has a malformed commit hash ("g9fdfcd"), so it is probably an older build. — [llama.cpp discussion #10879](https://github.com/ggml-org/llama.cpp/discussions/10879)
- The same thread recommends `RADV_PERFTEST=nogttspill` for RADV users because it "can fix a bunch of performance issues". — [#10879](https://github.com/ggml-org/llama.cpp/discussions/10879)
- Ollama on an RX 6500 XT 8 GB (attached to a Raspberry Pi 5, backend unclear): llama3.2:3b 39.8 tok/s, llama3.1:8b 22.4 tok/s, llama2:13b 2.0 tok/s (spills out of VRAM). — [geerlingguy/ollama-benchmark issue #1 (mirror)](https://cf-workers-proxy-9e9.pages.dev/geerlingguy/ollama-benchmark/issues/1)
- An anecdotal report says RDNA2 (6600 XT) Vulkan is about 10% faster than ROCm. — [Debian AI list, Jan 2025](https://lists.debian.org/debian-ai/2025/01/msg00175.html)
- RX 6500 XT is gfx1034. Community guides give `HSA_OVERRIDE_GFX_VERSION=10.3.0` (masquerade as gfx1030). The override is not officially supported, and the guides describe the card as "barely useful" (they assume the 4 GB variant). — [insiderllm](https://insiderllm.com/guides/rocm-not-detecting-gpu-amd-fix/), [localaimaster](https://localaimaster.com/blog/rocm-unsupported-gpu-hsa-override)
- With ROCm 6.4.3+ and the override, some gfx1031/gfx1032 builds segfault on the first prompt (gfx1034 not tested). — [runaihome](https://runaihome.com/blog/amd-gpu-not-detected-rocm-hsa-override-fix-2026/)
- A user with RX 6500 XT + 16 GB on Kubuntu could not install ROCm (amdgpu-dkms failure). Ollama Vulkan ran on the GPU but output **garbage** (random text, Korean). That was an Ollama build, so it may have been a driver or Ollama bug. — [Level1Techs forum](https://forum.level1techs.com/t/rocm-on-kbuntu-25-10-with-rx6500xt/249754)
- On an RX 9070 XT (RDNA4), Vulkan vs HIP depends on the model: Qwen3.5-4B Q8 tg is about 76 t/s on HIP vs 11–25 t/s on Vulkan (an anomaly, with a drop after 81 tokens), while Qwen3-4B Q8 is 102 t/s on Vulkan vs 85 on HIP. Qwen3.5 (Gated DeltaNet hybrid) Vulkan kernels looked immature in early 2026 (build b8196). — [r/LocalLLaMA snapshot, Mar 2026](https://reddit.sentinel-team.org/posts/1rkky2n/snapshots/2026-03-04T15%3A10%3A50.627029Z)

### Inferences
- Effective bandwidth: 3.82 GB × 27.8 t/s ≈ 106 GB/s (scoreboard). The user's own baseline (37 t/s on a 4B Q4_K_XL of about 2.6–2.9 GB) gives about 100 GB/s. **Rule of thumb: tg ≈ 100 GB/s ÷ resident weight size (GB) at short context.** At long context, KV reads add to this, so expect 15–30% lower tg at 32k.
- Estimates for fully GPU-resident dense models: **4B Q4 ≈ 35–40 tok/s** (measured 37), **8–9B Q4_K_M (5–5.5 GB) ≈ 17–20 tok/s**, **14B at IQ3_XXS/Q3_K_S (about 6–6.5 GB) ≈ 13–15 tok/s** but with very little room left for KV. 14B Q4_K_M (about 9 GB) does not fit and would go partial-offload (see section 2).
- pp512: the user's 685 t/s on 4B is much higher than the old 255 t/s scoreboard figure, which points to newer builds being faster. Expect roughly 300–350 t/s pp for 9B. For agentic use with 10–20k-token system prompts plus tool schemas, the **first turn costs about 15–60 s of prefill**, so prompt caching (same prefix every turn, `--cache-reuse`, a single slot) matters a lot.
- The 5600GT has a Vega 7 iGPU that Vulkan also enumerates. Pin llama.cpp to the dGPU with `GGML_VK_VISIBLE_DEVICES=<index>` (check `llama-server --list-devices`). If the monitor is plugged into the motherboard (iGPU), Hyprland uses no dGPU VRAM, so all 8 GB are free for the model. But the iGPU's UMA carve-out (BIOS, often 512 MB–2 GB) comes out of the 16 GB RAM, so set it to the minimum. If the monitor is on the 6500 XT, budget about 0.3–0.8 GB VRAM for the compositor and browser (my estimate, not measured).
- Note that the RX 6500 XT has **no hardware video encoder (AV1/H.264 encode)**, which is irrelevant here, but it also has no matrix cores. RDNA2 has no WMMA/coopmat, so Vulkan uses scalar/FP16 shaders. That is why pp is modest.

### Gaps
- No published llama.cpp Vulkan benchmarks exist for the **8 GB** RX 6500 XT or the RX 6400 with 2026 builds, and none cover 9B/14B/MoE on these cards. The user should run `llama-bench -m X -p 512 -n 128 -fa 0,1 -ngl 99` for each tier.
- I found no benchmark of ROCm on gfx1034 with llama.cpp. ROCm also mostly does not install on Arch kernels (rocm-hip-sdk in Arch repos), and gfx1034 is not in the official target list. Recommendation: skip it.
- Whether Qwen3.5/3.6 hybrid-attention Vulkan kernels have caught up by Nov 2026 is unverified. The user's 37 t/s suggests it is fine on this card.

---

## 2. PCIe 3.0 x4 and DDR4 bandwidth: partial offload and MoE expert offload

### Takeaway
For **token generation**, `--n-cpu-moe` / `-ot exps=CPU` keeps expert weights in RAM and computes them **on the CPU**, so PCIe barely matters and **DDR4 bandwidth is the limit**. With about 3B active parameters, expect roughly **12–22 tok/s on DDR4-3200**. For **prompt processing**, llama.cpp by default copies CPU-resident weights over PCIe to the GPU for large batches. On PCIe 3.0 x4 (about 3.5 GB/s) that becomes the bottleneck, so use large `-ub` or test `--no-op-offload`.

### Cited Findings
- 3060 Ti 8 GB + **32 GB DDR4-3200**, Qwen3.6-35B-A3B Q6, `--n-cpu-moe 38`, `-ub 1024`, q8_0 KV, 200K ctx: **pp ≈ 700–750 t/s on a 50K prompt, tg ≈ 33 t/s early, about 22 t/s near 200K**. That is on a PCIe 4.0 x16 / 3.0 x16 card, not x4. — [r/LocalLLaMA snapshot, Apr 2026](https://reddit.sentinel-team.org/posts/1srijdf/snapshots/2026-04-25T00%3A46%3A56.728262Z)
- RTX 4060 Laptop 8 GB + DDR5, Qwen3.6-35B-A3B Q4_K_M: 10–12 tok/s at `--n-cpu-moe 99` (all experts on CPU), **36.6 tok/s at `--n-cpu-moe 38`**. Full flags: `-ngl 99 --n-cpu-moe 38 -np 1 -fa on --cache-type-k q8_0 --cache-type-v turbo2 --no-mmap --mlock --ctx-checkpoints 1 --cache-ram 0 --jinja --reasoning on -b 2048 -ub 2048`. Advice from that thread: start with `--fit` (on by default in recent builds), then tune `--n-cpu-moe` by hand. One commenter found `-ub` mattered more than moving 1–2 more layers to the GPU. A GTX 1060 6 GB + 48 GB setup got about 12 tok/s (UD-IQ4_NL, 80k ctx). — [same thread](https://reddit.sentinel-team.org/posts/1srijdf/snapshots/2026-04-25T00%3A46%3A56.728262Z)
- RTX 3070 Ti 8 GB + DDR5, 35B-A3B IQ3_XXS: 8.7 tok/s with no flag, 27.5 at `-ncmoe 35`, 40.9 at `-ncmoe 25`, peak 43.8 at `-ncmoe 23`, then a drop to 19.8 at `-ncmoe 19` once VRAM overflowed. **Sweep the value; overshooting VRAM is catastrophic.** — [Medium (paywalled summary)](https://medium.com/coding-nexus/one-flag-tripled-my-tokens-per-second-on-an-8gb-gpu-238f55d1e3bd)
- GTX 1080 8 GB: about 24 tok/s on 35B-A3B with `--n-cpu-moe 30`. — [r/LocalLLaMA snapshot, May 2026](https://reddit.sentinel-team.org/posts/1tcc7h5/snapshots/2026-05-13T22%3A21%3A19.892601Z)
- CPU-only Qwen3-30B-A3B Q4_K_M: 16.8 tok/s on 8 threads (ARM board, LPDDR5), vs 57.6 on a 7900 XTX Vulkan. — [Radxa forum](https://forum.radxa.com/t/llama-cpp-benchmarks/27813/17)
- Expert offload in general: one user went from about 5 to 27 tok/s on a 35B-A3B by keeping attention and shared weights on the GPU and experts in RAM. — [r/LocalLLaMA snapshot](https://reddit.sentinel-team.org/posts/1rkb5hb/snapshots/2026-03-04T15%3A10%3A50.627029Z)

### Inferences (my arithmetic; treat as estimates)
- **DDR4 ceiling for MoE tg:** a Qwen3/3.6 A3B at Q4 (about 4.5–4.8 bpw) reads about 1.7 GB per token in total, of which roughly 1.3–1.5 GB is routed experts. If all experts sit on the CPU with about 40–45 GB/s usable DDR4-3200 bandwidth, the hard ceiling is about 28–32 tok/s, and realistic numbers are **12–20 tok/s** (Zen 3 with 6 cores also limits compute). Keeping some expert layers on the GPU (lower `--n-cpu-moe`) raises this. DDR5-6000 (about 96 GB/s, not this machine) would roughly double the CPU part. The DDR4 3060 Ti result (33 t/s) had about 4 GB more VRAM headroom for experts than we will have.
- **This box's constraints are tighter than any cited setup:** only about 7–7.5 GB usable VRAM, about 6–8 GB free RAM after the desktop (see section 3), and an **8 GB-RAM-minus-OS ceiling, so a 35B-A3B Q4_K_M (about 21 GB) does not fit at all**. The total model plus KV must stay under about 14 GB (7 GB VRAM + 7 GB RAM). That means 30–35B-A3B at about IQ3_XXS/Q2_K_XL (about 12–14 GB) or about 20B-A3.6B (gpt-oss-20b MXFP4, about 12–13 GB file).
- **PCIe 3.0 x4 and prompt processing:** for batches of 32 tokens or more, llama.cpp by default uploads CPU-resident expert weights to the GPU so the GPU can run big matmuls (op offload). At about 3.5 GB/s, moving about 7 GB of experts per micro-batch costs about 2 s. With `-ub 512` that caps pp at about 250 t/s, and with `-ub 2048` about 1000 t/s minus compute. So **use `-ub 1024–2048`** (VRAM permitting) and **A/B test `--no-op-offload`** (keeps those matmuls on the CPU; available in mainline since mid-2025 as far as I know, so verify with `llama-server --help`). Expected MoE pp on this box is roughly **100–300 t/s**, well below the 700 t/s of PCIe x16 setups. That makes long agent prompts slow on the first turn. Prompt caching across turns is essential.
- **Dense partial offload (for example 14B Q4_K_M with about 2 GB of layers on the CPU):** in tg the CPU layers run at DDR4 speed (about 40 GB/s vs about 100 on the GPU), so 25% of weights on the CPU roughly halves tg to about 6–8 tok/s. MoE expert offload is clearly better than dense partial offload on this machine.
- **Model load time over x4:** 7 GB VRAM ÷ about 3.3 GB/s ≈ 2–3 s of PCIe, plus disk read. That is acceptable for hot-swap (section 3). NVMe read speed dominates if the page cache was dropped.

### Gaps
- I found no measured MoE pp/tg on any **PCIe x4** card. The PCIe op-offload effect above is reasoning from llama.cpp behaviour, not a benchmark.
- The exact DDR4 speed is unknown (check with `sudo dmidecode -t memory | grep -i speed`). If it is DDR4-2666 or lower (with XMP off), subtract about 15–20%.

---

## 3. Memory budget, KV cache, mmap/mlock, and multi-model hot-swap

### Takeaway
Reserve about **8–9 GB of RAM** for Hyprland, browser, editor and MCP servers. That leaves about **7 GB RAM + about 7.3 GB VRAM ≈ 14 GB** for model plus KV plus compute buffers. Use `-fa on -ctk q8_0 -ctv q8_0` (not q4_0, which hurts tool calling), `-np 1`, and `--no-mmap` for anything with CPU-resident weights. Use **llama-swap** or **llama-server router mode** so only **one** model is loaded at a time.

### Cited Findings
- Aggressive KV quantisation hurts tool calling: the llama.cpp docs warn that "extreme" settings like `-ctk q4_0` can "substantially degrade the model's tool calling performance." — [llama.cpp docs/function-calling.md](https://raw.githubusercontent.com/ggml-org/llama.cpp/master/docs/function-calling.md)
- Two detailed 8 GB MoE setups both used q8_0 K/V and `--no-mmap`. One also used `--mlock`, `-np 1`, `--ctx-checkpoints 1` and `--cache-ram 0`. — [r/LocalLLaMA snapshot](https://reddit.sentinel-team.org/posts/1srijdf/snapshots/2026-04-25T00%3A46%3A56.728262Z)
- **Vulkan flash attention on AMD without coopmat** runs a scalar FA path. In Dec 2025 a "slight slowdown with FA on some hardware, particularly iGPUs with limited bandwidth" was called "expected" because of the scalar path and missing AMD tuning. The issue was left open. An FA regression in pp was also reported on a Renoir APU. The old claim (early 2025) that Vulkan FA falls back to the CPU on non-NVIDIA GPUs is outdated. — [buttondown llama.cpp weekly digest (Dec 2025)](https://buttondown.com/weekly-project-news/archive/weekly-github-report-for-llamacpp-december-01-7163/), [discussion #12629](https://github.com/ggml-org/llama.cpp/discussions/12629)
- On the scoreboard, FA **helped** RDNA2 RX 6600 (pp 758 → 805, tg 50.6 → 53.2) and RX 6650 XT. — [#10879](https://github.com/ggml-org/llama.cpp/discussions/10879)
- Oct 2026 (b11413): Vulkan sparse FA extended to quantised KV. Reported gains are +15–19% decode at 64k–128k with q8_0 KV on RDNA3/4 (secondary blog source, not RDNA2). — [rohitai.com](https://rohitai.com/blog/llama-cpp-b11413-vulkan-sparse-flashattention-quantized-kv)
- Qwen3.5 uses a 3:1 Gated DeltaNet : Gated Attention layout. Recurrent layers keep a fixed-size state, so **only 1 in 4 layers has a growing KV cache**. — [Raschka, hybrid attention](https://sebastianraschka.com/llm-architecture-gallery/hybrid-attention/). A third-party card lists the Qwen3.5-9B attention layers as 4 KV heads × head dim 256. — [friendli.ai model card](https://friendli.ai/models/abcdsystems/Qwen3.5-9B-Abliterated-GDN-Hybrid-EXL3-6bpw)
- **llama-server router mode:** start `llama-server` **without** `-m`. It auto-discovers GGUFs in the cache or in `--models-dir`, loads and unloads via `POST /models/load` and `/models/unload`, auto-loads on a request naming an unloaded model, `--models-max` caps concurrently loaded models (one source says the default is 4, with LRU eviction), and `--no-models-autoload` makes loading explicit. There is no single "unload all" endpoint, and the JSON shape is still evolving (secondary sources). — [glukhov.org](https://glukhov.org/llm-hosting/llama-cpp/unload-llama-cpp-router-models/), [pi.dev docs](https://pi.dev/docs/latest/llama-cpp)
- **llama-swap:** a single YAML file. Each `models.<name>.cmd` is a full `llama-server ... --port ${PORT}` command line. `ttl` (seconds, 0 = never unload) controls idle unload. `groups` with `exclusive: true` (default) unloads other groups, `swap`, `persistent` and `members` are also available. `-watch-config` gives hot reload. — [llama-swap wiki: Configuration](https://github.com/mostlygeek/llama-swap/wiki/Configuration), [betterstack guide](https://betterstack.com/community/guides/ai/llama-swap/)
- In llama-server, `--parallel` defaults to multiple slots and the context is divided among them. Use `--parallel 1` (`-np 1`) for a single-user agent. — [Future-AI-ROCM-support guide](https://github.com/fpresiado/Future-AI-ROCM-support)

### Inferences / budgeting (my arithmetic)
- **KV formula:** bytes/token = 2 (K and V) × n_attn_layers × n_kv_heads × head_dim × bytes/elem (f16 = 2, q8_0 ≈ 1.06, q4_0 ≈ 0.56). Configs below are from my recall of the HF config.json files, so verify against the GGUF metadata or llama.cpp's startup log line "KV self size":
  - Qwen3-8B (36 L, 8 KV heads, 128 dim): 144 KiB/token f16, so **32k ≈ 4.5 GiB f16 / 2.4 GiB q8_0**. A 9B Q4 dense model with full attention + 32k q8_0 KV ≈ 5.5 + 2.4 + about 0.5 compute ≈ 8.4 GB, which **does not fit in 8 GB**. Use 16k, or put KV on the CPU, or prefer hybrid-attention models.
  - Qwen3.5-9B (hybrid, about 8 attention layers × 4 × 256): about 32 KiB/token f16, so **32k ≈ 1 GiB f16 / 0.55 GiB q8_0**. Hybrid models (Qwen3.5/3.6) are a big win on 8 GB: 64k context is feasible.
  - Qwen3-30B-A3B (48 L, 4 KV heads, 128 dim): 96 KiB/token f16, so 32k ≈ 3 GiB f16 / 1.6 GiB q8_0.
  - gpt-oss-20b (24 L, 8 KV heads × 64 dim, half the layers sliding-window 128): roughly 24 KiB/token on full-attention layers, which is small.
- **RAM budget (16 GB):** idle desktop 6 GB (measured), plus browser with a few tabs 1–2 GB, editor/IDE 0.5–1.5 GB, MCP servers (each Node/Python process 50–200 MB, so 3–5 servers ≈ 0.5–1 GB) and the agent CLI itself 0.1–0.3 GB. **Reserve about 8.5–9 GB; leave about 1 GB slack for page cache, so about 6–7 GB is available for CPU-side weights and buffers.** Enable zram (Omarchy/Arch default via zram-generator if configured) as a safety valve, but never run model weights from swap.
- **VRAM budget (8 GB):** about 0.25–0.4 GB for the Vulkan driver/context, plus the compute buffer (grows with `-ub`; about 0.3–1 GB), plus KV, plus weights. Practically **about 7.0–7.3 GB for weights + KV** if the display is on the iGPU.
- **mmap vs --no-mmap:** with mmap, CPU-side weights live in the page cache, which the kernel can evict under memory pressure (browser spike), and the model is then re-read from disk mid-generation with a huge stall. With `--no-mmap` they go to anonymous memory that can be swapped (to zram) instead. **`--mlock` pins them, which guarantees speed but can OOM-kill other apps in 16 GB**. Recommendation: `--no-mmap` without `--mlock` for MoE tiers, with mlock only if RAM headroom is at least 2 GB. For fully-GPU tiers mmap does not matter much (weights are copied to VRAM; with `--no-mmap` the host copy is freed after upload, which is my understanding but should be verified).
- **Four-tier hot-swap:** with 24 GB total, **only one tier can be resident at a time**, because VRAM is the binding limit and every tier wants most of the 8 GB. Use llama-swap with all four models in one exclusive group, or router mode with `--models-max 1`. Swap cost is load time (about 3–10 s from NVMe with a warm page cache, longer when cold) plus a **full re-prefill of the agent's system prompt** (10–60 s). So the agent should not ping-pong between tiers per turn. Pick a tier per task or session.

### Gaps
- No RDNA2-specific measurement of FA + q8_0 KV on Vulkan in 2026 builds. Test `-fa on` vs `off` with `llama-bench -fa 0,1 -ctk q8_0 -ctv q8_0`. Note that quantised V cache **requires** FA in llama.cpp.
- I could not verify router-mode defaults (`--models-max` default, the exact endpoints) from the official README. Check `llama-server --help` on the installed build.

---

## 4. Tool calling in llama-server, and alternative runtimes

### Takeaway
Use **mainline llama.cpp `llama-server` with `--jinja`** (default-on in recent builds per one source), the model's own GGUF chat template (or a fixed template via `--chat-template-file`), `-np 1`, and q8_0 or f16 KV. Native parsers exist for Qwen/Hermes, Llama 3.x, Mistral Nemo, GPT-OSS (Harmony), Command R7B, Functionary and Firefunction; everything else falls back to a "Generic" JSON handler. ik_llama.cpp's Vulkan backend is unmaintained, so it is not an option for this AMD card.

### Cited Findings
- `--jinja` enables function calling for all models. Check the template at `/props` (`chat_template`, `chat_template_tool_use`). Override with `--chat-template-file`, or as a last resort `--chat-template chatml`. Native formats: Llama 3.1/3.3/3.2, Functionary v3.1/v3.2, Hermes 2/3, Qwen 2.5 / Qwen 2.5 Coder, Mistral Nemo, Firefunction v2, Command R7B, DeepSeek R1 (WIP, "reluctant to call tools"), GPT-OSS / Harmony. Others use `Chat format: Generic`, which "may consume more tokens and be less efficient". **Parallel tool calls are disabled by default; enable them per request with `"parallel_tool_calls": true`.** "Native support requires the right template for these GGUFs." — [llama.cpp function-calling.md](https://raw.githubusercontent.com/ggml-org/llama.cpp/master/docs/function-calling.md)
- An OpenCode write-up: Jinja templates are enabled by default in recent builds. Qwen3.5 9B and 4B sometimes **printed bash commands as text instead of calling the write tool**. Tool-calling ability "is not tied to size or recency". A context that is too small fails **silently** in OpenCode (errors only show in the server log), and a model emitting a tool call as plain text often indicates truncated context. — [jacar.es OpenCode + llama.cpp guide](https://jacar.es/?p=7245), [r/LocalLLaMA snapshot Feb 2026](https://reddit.sentinel-team.org/posts/1rbxfj8/snapshots/2026-02-25T19%3A22%3A50.330313Z)
- Reasoning flags seen in a working 2026 Qwen3.6 agent setup: `--jinja --reasoning on --reasoning-budget -1`, env `LLAMA_CHAT_TEMPLATE_KWARGS={"preserve_thinking":true}`, per-request `thinking_budget_tokens` (needs a server budget of -1), and `--reasoning-budget-message`. — [r/LocalLLaMA snapshot](https://reddit.sentinel-team.org/posts/1srijdf/snapshots/2026-04-25T00%3A46%3A56.728262Z)
- **ik_llama.cpp:** the author said in spring 2026 that he lacks the bandwidth to maintain the Vulkan backend and asked for a maintainer. Earlier, Vulkan pp was about 3x slower than CUDA and tg 20–25% slower, and ROCm would not compile for the author on AMD. Its strengths (CPU/MoE kernels, ubergarm IQ_K quants) are aimed at CUDA + CPU. — [ik_llama.cpp discussion #562](https://github.com/ikawrakow/ik_llama.cpp/discussions/562), [r/LocalLLaMA snapshot May 2026](https://reddit.sentinel-team.org/posts/1svy5rj/snapshots/2026-05-02T03%3A06%3A21.60641Z)
- Ollama on RX 6500 XT with Vulkan (`OLLAMA_VULKAN=1`) produced garbage output for one user. — [Level1Techs](https://forum.level1techs.com/t/rocm-on-kbuntu-25-10-with-rx6500xt/249754)

### Inferences
- **Recommendation: mainline llama.cpp Vulkan build plus llama-swap (or router mode).** On Arch: `llama.cpp-vulkan` (AUR, or the official package if present), or build with `cmake -B build -DGGML_VULKAN=ON && cmake --build build -j`. You need `vulkan-radeon` (RADV, Mesa) and `vulkan-headers`/`shaderc`/`glslang` for building. AMDVLK is discontinued by AMD (2025), so use RADV. (Packaging details are from my knowledge; verify AUR names.)
- Ollama, LM Studio and KoboldCpp all wrap llama.cpp. They add convenience but lag on Vulkan fixes, give less control over `-ot`/`--n-cpu-moe`/`-ub`, and their tool-call handling is their own layer. Ollama's Vulkan is newer and showed a garbage-output report on this exact card. LM Studio (AppImage) has a Vulkan llama.cpp runtime and exposes MoE CPU offload, so it is fine for exploration but not as an agent backend. KoboldCpp has Vulkan but targets roleplay and storytelling. For a custom CLI (NØX), a plain OpenAI-compatible `llama-server` endpoint is the best fit.
- Streaming tool calls: llama-server streams `tool_calls` deltas in the OpenAI format when `--jinja` is on (known from the server README and changelog; the function-calling doc I fetched does not cover it), so treat this as **unverified here**. NØX should handle both streamed deltas and a non-streamed fallback, and should **validate JSON args and retry** because small models produce malformed calls.

### Gaps
- I did not verify a list of 2026-specific tool-calling breakages (for example the Qwen3.5/3.6 XML-style `<tool_call>` parser, or GPT-OSS Harmony streaming issues). Check open issues labelled "tool calling" in ggml-org/llama.cpp for the installed build.
- The function-calling doc's template table "may be out of date" (minja was removed in favour of a new Jinja engine), so the native-parser list above may be incomplete for Qwen3.x and GLM.

---

## 5. Speculative decoding on Vulkan for code editing

### Takeaway
A **draft model is unlikely to help on this card**: VRAM is too scarce, and a measured Vulkan run regressed (0.95x). **Draftless n-gram speculation (`--spec-type ngram-mod` / lookup) is the right thing to try**, because agentic code editing repeats large spans of file content verbatim, and it costs no VRAM. Benchmark it yourself.

### Cited Findings
- RTX 5060 Ti, Qwen2.5-Coder with a draft model: 7B target **0.95x on Vulkan** (0.27x on CUDA); a 14B target on CUDA reached 1.4x overall, 1.57x on code, 1.85x on math, about 1.0x on chat/summarisation. CPU-only 3B: code 1.96x. — [InventiveHQ](https://inventivehq.com/blog/llama-cpp-speculative-decoding-consumer-gpu)
- llama.cpp `ngram-mod` is draftless, uses a hash pool shared across slots, and is aimed at repeating blocks of code or text. Example: `--spec-ngram-size-n 24 --draft-min 48 --draft-max 64`. Dense models may need lower draft settings. It can be combined with a draft model (draftless takes precedence). — [llama.cpp docs/speculative.md (Debian mirror)](https://sources.debian.org/src/llama.cpp/8064%2Bdfsg-1/docs/speculative.md), [mintlify mirror](https://www.mintlify.com/ggml-org/llama.cpp/inference/speculative-decoding)

### Inferences
- On a bandwidth-bound card, verifying K draft tokens costs about one forward pass, so accepted n-grams are close to free tokens. Expect gains mainly when the model rewrites or quotes existing code (search/replace blocks, whole-file rewrites) and none on fresh reasoning. With MoE plus CPU experts, verification batches touch more experts, so gains shrink (inference).
- A draft model (for example a 0.6–1.7B) would consume 0.5–1.2 GB VRAM plus its own KV, which is better spent on context. Skip it unless tier 2 has spare VRAM.

### Gaps
- No ngram-mod measurements on Vulkan or on AMD were found.

---

## 6. How coding agents configure local models, context and MCP

### Takeaway
All the major agents talk to llama-server through an **OpenAI-compatible provider** and need the context limit **declared explicitly**. For small models, keep the **system prompt plus tool schemas small** (few tools, terse descriptions), use **at least 16k and preferably 32k context**, and check server logs for silent truncation.

### Cited Findings
- OpenCode: declare an `@ai-sdk/openai-compatible` provider in `opencode.json` with `baseURL: http://127.0.0.1:8080/v1`, and **set `limit.context` and `limit.output`**, because local models are not on models.dev and OpenCode otherwise cannot tell when it is near the window. `small_model` can point to the same local model. — [jacar.es](https://jacar.es/?p=7245)
- `-c` directly sets the up-front KV reservation in llama.cpp. — [fast.io](https://fast.io/resources/llama-cpp-context-size.md)
- Too-small context gives no error in OpenCode, only in the server logs. ministral-3:8b was recommended as a fast, reliable small tool-caller and Qwen3 had full tool usage in tests (single user on an M1 16 GB, anecdotal). — [r/LocalLLaMA snapshot](https://reddit.sentinel-team.org/posts/1rbxfj8/snapshots/2026-02-25T19%3A22%3A50.330313Z), [jacar.es](https://jacar.es/?p=7245)

### Inferences
- **NØX design implications for small models:** (1) keep the system prompt plus tool JSON-schemas under about 3–5k tokens for tiers 1–2, because each MCP server can add thousands of tokens of schemas; load MCP tools lazily or per-task and only expose the 5–10 relevant ones; (2) **keep the prefix byte-stable across turns** (no timestamps at the top) so llama-server's prompt cache hits, since pp is the expensive part on this card; (3) prefer search/replace edit formats (short outputs, and ngram speculation helps) over whole-file rewrites; (4) cap tool-output injection (truncate grep and ls results) so small context windows are not overrun; (5) set an explicit `max_tokens` and handle `finish_reason: length`.

### Gaps
- I did not fetch per-agent docs for Cline, Continue, Aider, Goose, Qwen Code or Crush within this budget. Their specific context recommendations for small models remain unverified here. Most use the same OpenAI-compatible base URL plus model name plus context-length pattern.

---

## Appendix: suggested starting configs (synthesised from the above; these are untested starting points to tune with llama-bench)

Common env: `RADV_PERFTEST=nogttspill GGML_VK_VISIBLE_DEVICES=<dGPU index>`.
Common flags: `--jinja -np 1 -fa on -ctk q8_0 -ctv q8_0 --cache-reuse 256 --host 127.0.0.1`.

| Tier | Model class (selection is out of scope here) | Placement | Context | Expected tg (estimate) | Key flags |
|---|---|---|---|---|---|
| Fast | about 4B dense Q4_K_XL (about 2.7 GB) | all GPU | 32–64k | 35–40 (measured 37) | `-ngl 99 -c 32768 -ub 512` |
| Medium | about 8–9B dense Q4_K_M (about 5.3 GB); hybrid attention preferred | all GPU | 16k (full-attn) / 32–64k (hybrid) | 17–20 | `-ngl 99 -c 32768` |
| High | about 20B-A3.6B MoE (gpt-oss-20b MXFP4, about 12.8 GB) or 30B-A3B at Q3 | attention + some experts on GPU, rest of experts in RAM | 32k | 12–20 | `-ngl 99 --n-cpu-moe N` (sweep) or `--fit`, `--no-mmap`, `-ub 1024–2048`, try `--no-op-offload` |
| Ultra | 30–35B-A3B at IQ3_XXS/Q2_K_XL (about 12–14 GB), or 14B dense IQ3 all on GPU | as High, RAM nearly full | 16–32k | 8–15 | as High; close the browser; no mlock |

Hot-swap via llama-swap, all four models in one `exclusive` group, `ttl: 600`, or `llama-server --models-dir ~/models --models-max 1` (router mode).
