# Changelog

> This is the history of the whole NØX-Local project. Entries about the
> desktop app, Studio and the voice tools describe a separate private
> repository; this one contains the terminal agent only.

## Unreleased

- **The terminal UI is now NØX's own.** The TUI was ported from the cloud NØX onto the local backend: the gradient wordmark and header, the sidebar (models, context meter, plan, changed files, session), the diff cards inside approvals, the full-screen `/diff` viewer with split and unified modes, syntax-highlighted code, toasts, the command palette, vim keys, mouse selection, `@file` completion, session search, worktrees, goals and loops. The local backend is unchanged: same llama.cpp tiers, lean agent loop, MCP and skills. Providers without an equivalent were dropped from the UI: no API keys, `/login`, budgets, provider switching, racing two cloud models, Discord mirroring or dictation. `/compact` and `/handoff` now work by asking the local model for a summary and seeding the next session with it.
- **Fixes from testing:**
  - `/plan` + `/go` works: the plan-first turn is remembered on its own, so `/go` no longer answers "no plan waiting". The status line shows "plan first" and "plan ready · /go runs it".
  - `/btw` now sees the tool rows (what the agent read, ran and changed), not just the words, so it can answer "what are you doing?" while a turn runs.
  - Artifacts the backend reports (`canvas`, `code`) get a proper label, and a canvas opens as its JavaScript instead of a plain text dump.
  - PDF export no longer crashes on emoji. fpdf panicked on runes past U+FFFF; they are now dropped and common symbols get plain stand-ins. Exports and every tool also recover from panics and return an error, so a turn can no longer hang on "running…".
  - Stop works during commands. Each command runs in its own process group, and Stop or a timeout kills the group, servers it started included.
  - Bash has `background: true` for servers and watchers: it returns after 5 s with the URL, log and PID, and the agent tells the user what to open. Background commands stop when NØX exits.
  - Music: ace-synth decodes in tiles of 256 frames. A one-minute song in one 1024-frame tile made the driver reset the GPU (vk::DeviceLostError); 60 s now takes 31 s.
  - Voice: sherpa-onnx is pinned to v1.13.8, because 2.0 dropped the espeak-ng that Piper voices need. Read aloud says "Ø" as "O".
  - The thinking indicator animates while the model thinks.
- **Live visuals in the chat:** a new Artifact kind, `canvas`, is JavaScript defining `draw(ctx, w, h, t)` that runs live in the conversation, in a sandboxed frame. The `nox` helpers draw bar, line and pie charts, text, palettes and seeded randomness in the theme's colours, all offline. Canvas, HTML and SVG artifacts show inline with Replay, full screen and panel buttons. Runtime errors appear under the visual, off-screen animations pause, and saved canvases are standalone HTML pages. A new built-in skill, `visualize`, comes with four tested examples.
- **Desktop fixes:** the "Running…" line no longer flickers during tool calls; it stays in place instead of being re-inserted, and a finished tool card no longer replays its entrance animation. The Chat / Code / Studio switch is an aligned three-column control with icons.
- **Studio (voice):** a Voice mode that runs on the CPU with sherpa-onnx v2.0.1 (`scripts/install-studio.sh voice` downloads it and checks SHA-256). Read aloud uses the Czech voice Jirka (Piper) at a chosen speed. Transcribe uses Parakeet TDT 0.6B v3 on any recording ffmpeg reads, in 60 s chunks; the text can be copied or sent to the chat. Neither stops the chat model.
- **Studio (music):** a Music mode with ACE-Step 1.5 through acestep.cpp (`scripts/install-studio.sh music`). Style caption, optional lyrics or instrumental, length 30 s to 3 min, Czech or English vocals. The gallery plays tracks and shows their lyrics. A 30 s song took 27 s on the RX 6500 XT. The 1.7B lyrics model writes poor Czech, so the UI suggests writing your own.
- **Built-in skills** (`skills/`, installed to `~/.nox-local/skills` by `scripts/install.sh`):
  - `find-skills` searches GitHub in a fixed order: anthropics/skills, then code search for SKILL.md, then curated lists. It shows the candidates, checks a skill for risky scripts and instructions, and installs it with `scripts/install-skill.sh`, which only copies files and records the source.
  - `create-skill` writes a new SKILL.md in the format small models follow well.
- **Code without GitHub:** a Code task can open any local folder (in place, on its current branch; a folder without Git gets `git init` and a snapshot commit) or create an empty project in `~/Projects`. Push and pull requests use the folder's `origin` remote when there is one. GitHub cloning still works as before.
- **Studio (images):** a third desktop section that generates images locally with stable-diffusion.cpp. FLUX.2 Klein 4B or Z-Image Turbo, with progress, cancel and a gallery with prompt, seed and model. The chat model is unloaded first, and chat and Studio never use the GPU at the same time. `scripts/install-studio.sh` builds a pinned sd-cli.
- The desktop's default model is now Qwen3.5 9B (then 4B) instead of Mellum.
- **Qwen3.8 27B prompt processing fixed.** `--fit` used to put whole layers on the CPU, including the recurrent DeltaNet blocks, which the CPU ran on a single core: a 2.4k prompt did not finish in 4 minutes. Every layer now stays on the GPU and only the FFN weights of layers 24–64 live in RAM. Prompt processing is 70 t/s and generation 3 tok/s. `--fit` and MTP are off.
- **Qwen2.5-Coder 14B:** runs with 16k context, a 256 MiB `--fit` margin and no mmap. 36 of 49 layers now fit on the GPU, generation went from 3.7 to 5.3 tok/s and prompt processing from 110 to 151 t/s. The 9 GB file does not fit 8 GB of VRAM, so the old 22 tok/s note was dropped.
- **Qwen3.5 4B and 9B run without speculative decoding.** MTP only won on short prompts. At 19k tokens of context the 4B generated 11 tok/s with MTP + ngram and 32 tok/s without, and prompt processing went from 263 to 513 t/s. The measurements are in `internal/local/catalog.go`.
- **Thinking budget:** thinking requests carry `thinking_budget_tokens` (2048 at medium effort, 6144 at high). llama-server closes the think block when the budget runs out. Before this, `max_tokens` covered thinking and answer together, and Qwen3.5-4B once thought for 11k tokens (16 minutes) on one reply.
- llama-server starts with `--no-reasoning-preserve`, so templates that support it (Qwen3.8) no longer keep every past turn's thinking in the prompt.
- **Qwen3.8 27B runs:** weights loaded without mmap, a tighter `--fit` margin, an MTP draft head and low reasoning effort at medium.
- **Qwen mark:** a hexagon-knot SVG in a pill beside the desktop logo, in each theme's accent colour. The TUI badge reads `⢾⡷ using Qwen`.

- **Code desktop workspace:** GitHub account/device login, paginated repository picker and a separate clone/task branch per conversation. Reuses the local coding agent with a 60-step workflow.
- **Code review panel:** working-tree changes, text/binary previews, branch diff, task commits, selected-file commits, non-force task-branch pushes and a GitHub pull-request form link. Code checkouts survive conversation deletion.
- Serialize desktop Git mutations with agent turn starts; save final conversation state before marking a turn idle. Cancel pending GitHub login on app shutdown.

- **Deep research mode:** the Research toggle in the desktop and `/research` in the TUI. The model plans in TodoWrite, searches and reads in rounds, and writes a sourced markdown report. The step limit is 60.
- **DOCX and PDF export** (`internal/docexport`):
  - goldmark parses the markdown
  - DOCX gets real heading styles, lists, tables and code
  - PDF gets selectable text, embedded Noto or Liberation fonts and page numbers
  - the Artifact tool's `export` writes the files into the working folder, and the desktop artifact panel has "Save as MD / DOCX / PDF"
- **Obscura:** WebFetch renders pages through it when it is installed (`scripts/install-obscura.sh`, SHA-256 checked), and the desktop offers it as a browser connector. `/doctor` reports it.
- **Desktop:** the conversation is saved every few seconds during long turns.
- **Desktop:**
  - each chat has its own folder, with a Files panel, ZIP download, open folder and "use another folder"
  - Connectors: an MCP catalog, import from Claude Code, custom servers by command or URL, and removal
  - a Usage dialog (click the context gauge)
  - quiet animations
- **TUI:** `/usage`. Token counts now read 1.0k, 37.8k, 120k and 1.4M.
- When a turn ends with an empty reply (Mellum answering only while thinking), the model is asked once for its answer.
- **Artifacts:** the model saves documents, pages, code and SVG with the `Artifact` tool. Each one is versioned under `~/.nox-local/artifacts/`, and `/artifacts` in the TUI opens it.
- **Token counter:** it now includes tool-call arguments and uses the server's exact count after each request.
- **NØX-Local Desktop 0.1.0 (preview):**
  - a Wails app in the Frost style with conversations, streaming, tool steps, approvals, MCP switches and an artifact panel
  - the install script adds a launcher entry
  - two light themes, switched in the sidebar: Paper (default) and Frost. Literata, Karla and Nunito Sans are bundled (OFL), so the app works offline.

## 1.0.0-rc1 (2026-11-07)

This is the first release candidate. NØX-Local splits off from the cloud NØX as its own project, for local models only.

### Models

- There are four tiers, all measured on an RX 6500 XT 8 GB, Ryzen 5 5600GT and 16 GB DDR4:

  | Tier | Model | Generation |
  |---|---|---|
  | FAST | Qwen3.5-4B | 51 tok/s with MTP |
  | MEDIUM | Mellum2.1-12B-A2.5B | 53 tok/s |
  | HIGH | gpt-oss-20b | 19 tok/s |
  | ULTRA | Qwen3.6-35B-A3B | 22 tok/s |

  Tool calling passed 3/3 on every tier. See `docs/BENCHMARKS.md`.
- `llama-server` runs with `RADV_PERFTEST=nogttspill`. Without it, Mesa spilled buffers into system RAM, which made generation 3 times slower and prefill 8 times slower.
- MoE tiers load with `--load-mode none`, because mmapped experts in RAM dropped gpt-oss-20b to 2.9 tok/s.
- Every profile uses n-gram speculative decoding, and FAST adds MTP. Code edits run 2 to 4 times faster.
- Mellum always runs with thinking on, so its reasoning no longer leaks into the answer.

### Agent

- The loop is lean: a short prompt, tool output trimmed, files read 300 lines at a time, tool calls written as text recovered, and a guard against repeated calls.
- There are nine built-in tools: `Read`, `Write`, `Edit`, `Glob`, `Grep`, `Bash`, `WebSearch` (DuckDuckGo, no key), `WebFetch` and `TodoWrite`.
- MCP servers are opt-in. `/mcp` switches them on and off, only servers that are on get connected, and the choice is remembered.
- Skills from `.nox/skills`, `~/.nox-local/skills` and `~/.claude/skills` are listed compactly in the prompt.
- Effort has three levels: `low`, `medium` and `high`. `high` raises the thinking budget and gpt-oss's `reasoning_effort`.

### TUI

- `/model` lists the downloaded tiers with their measured speed. Switching keeps the conversation.
- `/compact` summarises the conversation and continues in a fresh context. At 70 % context you get a hint instead of an automatic compaction.
- `/todos` and a sidebar card show the agent's task list. The sidebar also has MODEL and MCP cards.
- `/doctor` checks llama-server, the models and MCP.

### Removed (cloud only)

- Provider switching and `@mentions`
- `/login` and the API key store
- The models.dev catalog
- Twin view and ultracode
- Cost and plan-limit tracking
- Image attachments
- The xhigh and max effort levels

### Other

- State lives in `~/.nox-local` and is separate from the cloud NØX. An existing llama.cpp build in `~/.nox/llama.cpp` is still used.
