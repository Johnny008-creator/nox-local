<div align="center">

# N Ø X · Local

**A terminal coding agent that runs only on your own machine.**

It uses the same TUI as the cloud NØX, with an agent loop built for 4B to 35B models running in llama.cpp.

![version](https://img.shields.io/badge/version-1.0.0--rc1-14b8a6)
![go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![llama.cpp](https://img.shields.io/badge/llama.cpp-Vulkan-8b5cf6)
![platform](https://img.shields.io/badge/platform-Linux-f59e0b)
![license](https://img.shields.io/badge/license-MIT-22c55e)

</div>

<sub>What the TUI looks like (MEDIUM tier, sidebar on):</sub>

```text
 N  Ø  X                                                                                 ╭──────────────────────────────╮
 ━━━━━━━━━━━━━━━━━━━                                                                     │ M O D E L S                  │
  local  qwen3.5-9b · llama.cpp · v1.0.0-rc1                                             │ ○ qwen3.5-4b                 │
 ~/code/notes-cli                                                                        │ ● qwen3.5-9b          active │
                                                                                         ╰──────────────────────────────╯
 ▎ add a --json flag to the export command                                               ╭──────────────────────────────╮
 ∴ thought for 2.1s                                                                      │ C O N T E X T                │
 ◆ Read cmd/export.go ✓ 1.2s                                                             │ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
   ⎿ 87 lines                                                                            │ 9.4k / 32.8k · 28%           │
 ◆ Edit cmd/export.go ✓ 3.4s                                                             ╰──────────────────────────────╯
 ╭────────────────────────────────────────────────────────────────────────────────────╮  ╭──────────────────────────────╮
 │ export.go  +2 −1                                                                   │  │ P L A N  2/3                 │
 │   fs.StringVar(&out, "o", "", "output file")                                       │  │ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
 │ + fs.BoolVar(&asJSON, "json", false, "write JSON…")                                │  │ ✓ read export.go             │
 ╰────────────────────────────────────────────────────────────────────────────────────╯  │ ✓ add the --json flag        │
 ◆ Run go test ./... ✓ 0.2s                                                              │ ● run the tests              │
   ⎿ ok    notes-cli/cmd  0.21s                                                          ╰──────────────────────────────╯
 Added --json to export: it writes the notes as a JSON array instead of Markdown. Tests  ╭──────────────────────────────╮
 pass.                                                                                   │ C H A N G E D                │
                                                                                         │ cmd/                         │
                                                                                         │ └ export.go          +2 −1   │
                                                                                         ╰──────────────────────────────╯
                                                                                         ╭──────────────────────────────╮
                                                                                         │ S E S S I O N                │
                                                                                         │ 1 turn · 1m 12s              │
                                                                                         ╰──────────────────────────────╯
 ╭──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
 │ ❯ Ask NØX anything · / for commands                                                      local · qwen3.5-9b · medium │
 ╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
  ⏵⏵ auto mode on (shift+tab to cycle)                                                                       9.4k tokens
```

## Why

Cloud agents send your code to someone else's server and stop working when your quota runs out. NØX-Local runs on your own machine, offline and for free. Small models need a different agent loop than frontier models, though, and most of NØX-Local is that loop:

- The system prompt is short and there are only nine built-in tools, so a 4B model still picks the right one.
- Tool output is trimmed, files are read 300 lines at a time, and old tool results are dropped first when the context fills.
- If the model writes a tool call as plain text instead of making it, the call is recovered and run.
- A loop guard stops the model after three identical calls in a row.
- MCP servers are opt-in, so the model only sees the tools you switched on.

## Model tiers

Pick a tier with `/model` at any time. The conversation carries over.

The speeds below were measured on the reference machine: RX 6500 XT 8 GB on PCIe 3.0 x4, Ryzen 5 5600GT, 16 GB DDR4.

| Tier | Model | File | Writing | Editing code¹ | Tool calls |
|---|---|---|---|---|---|
| **FAST** | Qwen3.5-4B UD-Q4_K_XL | 3.0 GB | 51 tok/s | 219 tok/s | 3/3 |
| **MEDIUM** | Mellum2.1-12B-A2.5B MXFP4 | 7.3 GB | 53 tok/s | 186 tok/s | 3/3 |
| **HIGH** | gpt-oss-20b MXFP4 | 12.1 GB | 19 tok/s | 38 tok/s | 3/3 |
| **ULTRA** | Qwen3.6-35B-A3B UD-IQ3_XXS | 13.2 GB | 22 tok/s | 42 tok/s | 3/3 |

¹ When the model rewrites code it has just read, n-gram speculative decoding drafts tokens from the context. FAST also uses its MTP head. Both are on by default and cost no VRAM.

HIGH and ULTRA are mixture-of-experts models. Their experts sit in system RAM, so together with the KV cache and llama-server's working memory each tier uses well under 18 GB. That leaves room for a browser and an editor.

The full numbers and the settings that mattered are in [docs/BENCHMARKS.md](docs/BENCHMARKS.md). The research behind the picks is in [docs/research](docs/research/reports).

## Features

| | |
|---|---|
| **Tools** | `Read` `Write` `Edit` `Glob` `Grep` `Bash` `WebSearch` `WebFetch` `TodoWrite`, plus the tools of enabled MCP servers |
| **MCP** | Servers come from `config.toml` or a project `.mcp.json`. `/mcp` switches them on or off, and only servers that are on get connected. |
| **Skills** | `SKILL.md` folders from `.nox/skills`, `~/.nox-local/skills` and `~/.claude/skills`, listed compactly in the prompt. Built in: **find-skills** (search GitHub, review, install), **create-skill** (write a new one) and **visualize** (charts, animations, diagrams) |
| **Task list** | The model plans multi-step work with `TodoWrite`. You see the list in the sidebar and with `/todos`. |
| **Context** | A gauge in the sidebar. At 70 % you get a hint, and `/compact` summarises the conversation and continues in a fresh context. |
| **Safety** | Four modes: `manual`, `plan`, `auto` and `yolo`. In `auto`, protected files (`.env`, keys) and risky commands still ask first. |
| **Undo** | Git checkpoints before every turn. `/rewind` restores files. |
| **Sessions** | `/resume`, `-c` to continue, and `/export` to Markdown |
| **Project memory** | `NOX.md` instructions, created with `/init` and loaded into every prompt |
| **Automation** | Hooks on prompt, after tools and at the end of a turn, optional auto-commit, and `/worktree` |
| **Headless** | `nox-local -p "…"` prints the answer, and `--output json` streams events. |

## Quick start

### 1. Build llama.cpp with Vulkan

```sh
sudo pacman -S --needed cmake ninja vulkan-headers spirv-headers shaderc   # Arch
git clone --depth 1 https://github.com/ggml-org/llama.cpp ~/.nox-local/llama.cpp
cmake -S ~/.nox-local/llama.cpp -B ~/.nox-local/llama.cpp/build -G Ninja \
  -DGGML_VULKAN=ON -DCMAKE_BUILD_TYPE=Release -DLLAMA_CURL=OFF
cmake --build ~/.nox-local/llama.cpp/build --target llama-server llama-bench
```

NØX-Local looks for `llama-server` in this order: `$NOX_LLAMA_SERVER`, then `PATH`, then `~/.nox-local/llama.cpp`, then `~/.nox/llama.cpp`.

### 2. Install NØX-Local

Download `nox-local` from [Releases](../../releases), or build it from source (Go 1.27+):

```sh
git clone https://github.com/Johnny008-creator/N0X-Local && cd N0X-Local
sh scripts/install.sh            # → ~/.local/bin/nox-local
```

### 3. Get a model and start

```sh
nox-local models                 # catalog, sizes, what is downloaded
nox-local pull mellum2.1-12b     # into ~/models, resumes if interrupted
nox-local                        # TUI
```

## Using it

| Command | |
|---|---|
| `/model` | switch tier, which loads on your next message |
| `/effort` | `low` answers directly, `medium` thinks first, `high` thinks longer |
| `/mode` | `manual`, `plan`, `auto` or `yolo` (also **Shift+Tab**) |
| `/mcp` | switch MCP servers on and off |
| `/skills` | browse skills and use one |
| `/todos` | show the agent's task list |
| `/compact` | summarise and continue in a fresh context |
| `/rewind` | undo file changes from earlier turns |
| `/resume` · `/clear` · `/export` | sessions |
| `/doctor` | check llama-server, models and MCP |
| `/help` | everything else |

**Keys:**

- **Ctrl+O** picks a model and **Ctrl+T** cycles effort.
- **@file** attaches a project file.
- **Esc** interrupts the current turn; prompts you queued still run.
- **Ctrl+C** stops everything.

## Configuration

Settings live in `~/.nox-local/config.toml`, which is created with comments on the first run. A project can override them in `.nox/config.toml`.

```toml
model  = "mellum2.1-12b"   # tier to start with
mode   = "auto"
effort = "low"

[mcp.filesystem]           # off until you switch it on with /mcp
command = "npx"
args    = ["-y", "@modelcontextprotocol/server-filesystem", "."]

[[hooks]]
event   = "stop"
command = "go build ./..."
```

NØX-Local keeps all its state in `~/.nox-local` and shares nothing with the cloud NØX.

## Notes for AMD / Vulkan

- NØX-Local starts `llama-server` with `RADV_PERFTEST=nogttspill`. Without it, Mesa spills buffers into system RAM over PCIe, and a PCIe x4 card then runs 3 to 8 times slower.
- With nogttspill, VRAM is a hard limit shared with your desktop. Placement is left to `--fit`, which keeps a 1 GiB margin.
- Don't raise `-ub` or `-b` by hand on the MoE tiers.
- MoE models load without mmap. Experts kept in a memory-mapped file get evicted from the page cache and slow generation down badly.

## Status

`1.0.0-rc1` is feature-complete and measured on the reference machine. It still needs real-world use before 1.0.0. Bugs and rough edges go in [Issues](../../issues).

**Next:**

- More measured profiles for other GPUs.

## License

MIT © 2026 Jan Raszka. See [LICENSE](LICENSE).
