# Benchmarks

These were measured on 2026-11-07 on an RX 6500 XT 8 GB (PCIe 3.0 x4), a Ryzen 5 5600GT and 16 GB of DDR4. The software was llama.cpp `cd959ba` (Vulkan, RADV / Mesa 26.2.5).

Each model was launched through `llama-server` exactly the way `nox-local` launches it: `--fit` placement, KV cache q8_0, flash attention, and the context window from the catalog.

- **Prefill:** a ~3.5K-token prompt.
- **Generation:** 400 tokens.
- **Tools:** 3 attempts, each a `read_file` call that must parse.

| Tier | Model | Context | Prefill tok/s | Generation tok/s | Tools |
|---|---|---|---|---|---|
| FAST | Qwen3.5-4B UD-Q4_K_XL + MTP | 64K | 553 | **55.3** (MTP, 81 % accepted) | 3/3 |
| FAST | without MTP | 64K | 664 | 36.4 | 3/3 |
| MEDIUM | Mellum2.1-12B-A2.5B MXFP4 | 32K | 491 | **46.3** | 3/3 |
| MEDIUM | with MTP | 32K | 308 | 40.1 (63 % accepted, slower) | 3/3 |
| HIGH | gpt-oss-20b MXFP4 | 64K | 207 | **18.2** | 3/3 |
| ULTRA | Qwen3.6-35B-A3B UD-IQ3_XXS | 48K | 180 | **20.9** | 3/3 |

All four tiers also answered a real `nox-local -p` task correctly: count the profiles in `catalog.go` using tools. Wall time for that task was 14 s on FAST, 27 s on MEDIUM, 42 s on HIGH and 46 s on ULTRA.

## Speculative decoding

`--spec-type ngram-mod` drafts tokens from text that is already in the context. It needs no draft model and no VRAM.

Agents mostly rewrite code they have just read, so it is on for every profile; FAST combines it with MTP.

The edit task asks the model to reproduce a ~3.5K-character Go file with one variable renamed (900 tokens out, temperature 0).

| Tier | Edit, no spec | Edit, ngram-mod | Plain generation with ngram-mod |
|---|---|---|---|
| FAST (MTP → MTP + ngram) | 60.3 | **218.7** | 48.4 → 51.0 |
| MEDIUM | 49.4 | **185.7** | 46.3 → 53.0 |
| HIGH | 18.3 | **38.2** | 18.5 → 18.7 |
| ULTRA | 22.9 | **41.6** | 20.9 → 21.9 |

Things that did not help on HIGH:

- **CPU thread count:** 4–8 threads all gave 18.5 tok/s, and 12 threads dropped to 16.8.
- **Smaller context:** dropping from 64K to 32K gave +0.3 tok/s.

Generation of experts in RAM is bound by DDR4 bandwidth, not by the CPU.

## Settings that mattered

- **`RADV_PERFTEST=nogttspill`**
  - Without it, Mesa spills GPU buffers into system RAM over PCIe x4.
  - On FAST at 64K that cost 3× in generation (36 → 11 tok/s) and 8× in prefill (664 → 78).
  - `nox-local` sets it for `llama-server`.
- **`--load-mode none` for MoE models**
  - With mmap, the experts that `--fit` keeps in RAM get evicted from the page cache.
  - gpt-oss-20b fell from 18 to 2.9 tok/s, and ULTRA prefill from 180 to 41.
- **Do not raise `-ub` / `-b` on MoE tiers**
  - With nogttspill, VRAM is a hard limit shared with the desktop.
  - `-ub 2048` on ULTRA exhausted it and crashed the compositor (amdgpu "Not enough memory for command submission").
  - Leave placement to `--fit`, which keeps a 1 GiB margin.
- **MTP**
  - Clear win on Qwen3.5-4B.
  - A loss on Mellum, so it is off there.
