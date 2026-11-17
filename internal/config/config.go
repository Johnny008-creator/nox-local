// Package config loads ~/.nox-local/config.toml (global) and
// .nox/config.toml in the working directory (project), project values winning.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/Johnny008-creator/nox-local/internal/hooks"
	"github.com/Johnny008-creator/nox-local/internal/mcpclient"
	"github.com/Johnny008-creator/nox-local/internal/paths"
)

type Config struct {
	Model  string `toml:"model"`  // model id to start with ("" = MEDIUM tier)
	Mode   string `toml:"mode"`   // manual | plan | auto | yolo
	Effort string `toml:"effort"` // low | medium | high
	// MCP servers; each one is connected only after you enable it with /mcp.
	MCP map[string]mcpclient.ServerConfig `toml:"mcp"`
	// Commands run on prompt / post_tool / stop.
	Hooks []hooks.Hook `toml:"hooks"`
	// Commit all changes after every turn that changed files (needs git).
	AutoCommit bool `toml:"auto_commit"`
}

const template = `# NØX-Local settings. Project overrides live in .nox/config.toml.

# Model to start with (nox-local models lists them). Empty = MEDIUM tier.
# model = "qwen3.5-4b"

# manual | plan | auto | yolo   (Shift+Tab cycles at runtime)
mode = "auto"

# low (answer directly, fastest) | medium | high (think first)
effort = "low"

# MCP servers. Nothing is connected until you switch a server on with /mcp:
# small models handle a few tools well, not dozens. A project's .mcp.json
# (Claude Code format) is read too.
# [mcp.filesystem]
# command = "npx"
# args = ["-y", "@modelcontextprotocol/server-filesystem", "."]
# [mcp.docs]
# url = "https://example.com/mcp"

# Commit after every turn that changed files.
# auto_commit = true

# Hooks. Events: prompt (failing blocks the prompt), post_tool (match =
# regexp on tool name), stop (after each turn).
# [[hooks]]
# event = "stop"
# command = "go build ./..."
# [[hooks]]
# event = "post_tool"
# match = "Edit|Write"
# command = "gofmt -l ."
`

func Default() Config {
	return Config{Mode: "auto", Effort: "low", MCP: map[string]mcpclient.ServerConfig{}}
}

// Load reads both files; missing files are fine. A commented template is
// written to ~/.nox-local/config.toml on first run.
func Load() Config {
	c := Default()
	global := paths.File("config.toml")
	if _, err := os.Stat(global); os.IsNotExist(err) {
		_ = os.WriteFile(global, []byte(template), 0o644)
	}
	for _, p := range []string{global, projectFile()} {
		if p == "" {
			continue
		}
		var layer Config
		if _, err := toml.DecodeFile(p, &layer); err != nil {
			continue
		}
		if layer.Model != "" {
			c.Model = layer.Model
		}
		if layer.Mode != "" {
			c.Mode = layer.Mode
		}
		if layer.Effort != "" {
			c.Effort = layer.Effort
		}
		for k, v := range layer.MCP {
			c.MCP[k] = v
		}
		c.Hooks = append(c.Hooks, layer.Hooks...)
		if layer.AutoCommit {
			c.AutoCommit = true
		}
	}
	return c
}

func projectFile() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(wd, ".nox", "config.toml")
}

// Path is the global config file.
func Path() string { return paths.File("config.toml") }
