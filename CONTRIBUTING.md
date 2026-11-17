# Contributing

Thanks for taking a look. NØX-Local is one person's tool that got useful enough to share, so the best contributions are focused ones.

## The one rule

**The agent loop must stay usable by a 4B model.** That is the whole point of the project.

If a change adds a tool, a prompt section, a round trip or a required capability, say what it costs a small model. Changes that assume a frontier model will be declined even if they are good code.

## Before you open a PR

```sh
go vet ./...
go test ./...
```

Both must pass. If you touched the TUI, run it: `go run ./cmd/nox-local`.

## What is most useful

- **A model profile** for a GPU you have. Numbers matter: file, context, generation tok/s, prompt tok/s, and the flags you used. Saluki in `internal/local/catalog.go` is a good example of an argument that is not obvious.
- **A bug report with the server log.** `~/.nox-local/local/llama-server.log` explains most failures.
- **A fix for a template quirk.** Models differ in how they mark tool calls and thinking. Parsing belongs in the loop, not in a prompt hack.
- **Translation or clarity** in the README. It is the first thing people see.

## What is not useful

- Formatting-only PRs.
- New dependencies. The binary is small and self-contained on purpose.
- Features that only work with a 70B model.

## Style

- Go standard library first, `gofmt`, short functions.
- Comments explain *why*, not *what*.
- Tests next to the code they test.

## Security

Do not open a public issue for a vulnerability. See [SECURITY.md](SECURITY.md).
