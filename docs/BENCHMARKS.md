# Benchmarks

Measured on the reference machine: RX 6500 XT 8 GB (PCIe 3.0 x4), Ryzen 5 5600GT, 16 GB DDR4. llama.cpp `cd959ba`, Vulkan through RADV (Mesa 26.2.5), flash attention on, KV cache q8_0.

The numbers below come from `llama-bench -p 512 -n 128 -fa 1 -ctk q8_0 -ctv q8_0`, which is how the launch settings can be compared without a chat template in the way. Saluki 27B does not fit in 8 GB on its own, so its numbers are from `llama-server --fit` with a 16K context.

| Model | File | Context | pp512 t/s | tg128 t/s |
|---|---|---|---|---|
| Qwen3.5 4B | 3.0 GB | 64K | 591 | **32.5** |
| Qwen3.5 9B | 6.1 GB | 32K | 378 | 8.8 |
| Ornith 1.5 9B IQ4_XS | 5.2 GB | 32K | 425 | **22.0** |
| Nemotron 3.5 Bolt E4B | 5.1 GB | 32K | **655** | 8.5 |
| Underdog Saluki 27B IQ2 | 7.9 GB | 16K | ~5 | ~3.2 |

`pp512` is prompt processing, `tg128` is generation; both are what an agent feels: prompt speed on the first turn and after every tool result, generation speed for the answer.

## What the numbers say

- **4B is in a class of its own.** 32 tok/s generation with the whole model in VRAM. It is the only model here that feels interactive.
- **Ornith 1.5 9B is the sweet spot.** A Qwen3.5 fine-tune quantized to IQ4_XS: 22 tok/s, 425 t/s prefill, 5.2 GB. Nearly the 9B's capability at twice the speed.
- **The 9B is not twice the 4B.** It is 2.5× slower to write and 1.6× slower to read. Worth it for Czech and longer plans, painful for tool loops.
- **Nemotron Bolt reads fast and writes slowly.** 655 t/s prefill is the best here, 8.5 t/s generation the worst of the small models. Good for "read this 20K file and answer briefly", bad for writing code.
- **Saluki 27B does not fit.** Everything about it is slow on an 8 GB card, and a 2K-token agent prompt takes minutes to process. It is the smartest model in the catalog and the least usable in a loop.

## Context depth

Generation speed does not stay put as the context fills. Measured on Qwen3.5-4B with llama-server, 64K context:

| Prompt | Generation | Prefill |
|---|---|---|
| 2.4K | 43 tok/s | — |
| 10K | 43 tok/s | — |
| 19K | 12 tok/s | 263 t/s |

Past roughly 10K tokens the speed drops, which is why `/compact` exists and why the profiles keep the context at 32K rather than the model's full 64K or 256K.

## Speculative decoding

`--spec-type ngram-mod` drafts tokens from text already in the context. No draft model, no VRAM. Agents mostly rewrite code they have just read, so it is a clear win on edits:

| | Edit, no spec | Edit, ngram-mod |
|---|---|---|
| Qwen3.5 4B | 60.3 | **218.7** |
| Mellum 12B (legacy) | 49.4 | **185.7** |

It also **halves prompt processing** at long context on Qwen3.5, so the Qwen and Ornith profiles launch with `--spec-type none`. Nemotron was measured both ways in a server run with a short prompt: ngram changed generation not at all (8.1 vs 7.8) and cut prefill from 74 to 36 t/s, so it runs without speculation too.

MTP (the multi-token prediction heads in the Qwen3.5-4B/9B files) only wins on short prompts. Past ~10K tokens it collapses: 11 tok/s with MTP against 32 without at 19K, and prompt processing halves. An agent spends most of a session past 10K, so MTP is not used.

## Settings that mattered

- **`RADV_PERFTEST=nogttspill`** — without it, Mesa spills GPU buffers into system RAM over PCIe x4. On the 4B at 64K that cost 3× in generation (36 → 11 tok/s) and 8× in prefill (664 → 78). NØX-Local sets it for `llama-server`.
- **`--load-mode none` for models that do not fit** — with mmap, the layers `--fit` keeps in RAM get evicted from the page cache. On the 27B that turned a working 3 tok/s into a server that never answered.
- **Do not raise `-ub` / `-b`** — with nogttspill, VRAM is a hard limit. `-ub 2048` on a 27B exhausted it and crashed the compositor. Placement is left to `--fit`, which keeps a 1 GiB margin.
- **32K context, not the model maximum** — KV cache is VRAM. The 4B trains at 64K and is launched with 64K because it fits; the 9B trains at 256K and is launched with 32K because 64K would push layers onto the CPU.
