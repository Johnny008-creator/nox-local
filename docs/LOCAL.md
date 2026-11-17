# nox-local

NØX for models on your own machine. `nox-local` starts llama.cpp's
`llama-server` with settings for small GPUs and drives the model with a lean
agent loop (short prompt, six tools, tight output limits, recovery of tool
calls written as text, a stop for repeated calls).

## Setup (Linux, AMD/Intel/NVIDIA via Vulkan)

```sh
sudo pacman -S --needed cmake ninja vulkan-headers spirv-headers shaderc   # Arch
git clone --depth 1 https://github.com/ggml-org/llama.cpp ~/.nox/llama.cpp
cmake -S ~/.nox/llama.cpp -B ~/.nox/llama.cpp/build -G Ninja \
  -DGGML_VULKAN=ON -DCMAKE_BUILD_TYPE=Release -DLLAMA_CURL=OFF
cmake --build ~/.nox/llama.cpp/build --target llama-server llama-bench
```

`nox-local` finds `llama-server` on PATH, in `~/.nox/llama.cpp/build/bin`,
or at `$NOX_LLAMA_SERVER`.

## Use

```sh
nox-local models            # catalog, sizes, what is downloaded
nox-local pull qwen3.5-4b   # download (resumes)
nox-local                   # TUI
nox-local -p "explain main.go"
```

Effort `low` (default) answers directly. `medium` lets the model think first, which is slower but better on hard tasks; `high` gives it a bigger thinking budget, and gpt-oss also reasons harder. Models are stored in `~/models` (`NOX_MODELS`).

NØX-Local keeps its own state in `~/.nox-local`:

- **Config:** `config.toml`
- **Sessions and history:** `sessions/` and `native/`
- **Server log:** `local/llama-server.log`

It shares nothing with the cloud NØX in `~/.nox`, apart from reusing a llama.cpp build in `~/.nox/llama.cpp` if one is there.

## In the TUI

- **`/model`** switches tier (FAST, MEDIUM, HIGH, ULTRA). The conversation carries over, and the new model loads on your next message.
- **`/mcp`** switches MCP servers on and off.
  - Servers come from `[mcp.<name>]` in the config and from a project `.mcp.json`.
  - A server is connected and offered to the model only while it is on, because small models handle a few tools well and dozens badly.
  - The choice is remembered.
- **Skills** are folders with a `SKILL.md`, found in `.nox/skills`, `~/.nox-local/skills` and `~/.claude/skills`.
  - They are listed compactly in the prompt, and the model reads one when a task matches.
  - `/skills` lets you pick one by hand.
- **`/compact`** asks the model for a summary and continues in a fresh context seeded with it.
  - Once the context is 70 % full, NØX tells you it is a good time for `/compact`; it never compacts on its own.
- **`/todos`** shows the task list the model keeps with `TodoWrite`. The sidebar shows it too.

Tools the model gets: `Read`, `Write`, `Edit`, `Glob`, `Grep`, `Bash`, `WebSearch` (DuckDuckGo, no key), `WebFetch`, `TodoWrite`, plus the tools of enabled MCP servers.

GPU layers and MoE expert offload are not pinned: llama-server's `--fit`
sizes them to the free VRAM at launch.
