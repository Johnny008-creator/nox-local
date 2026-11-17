# Security

NØX-Local runs a model and the commands it asks for on your machine. It is not a sandbox: the agent can read and write files and run shell commands in the directory you start it in.

## What the design assumes

- **You are the last line of defence.** `manual` and `plan` modes ask before changes; `auto` asks for risky commands and protected files; `yolo` asks nothing.
- **The project's own settings are untrusted.** A project's `.mcp.json` and `.nox/config.toml` can start processes, so they stay off until you run `/trust`.
- **Network access is two tools.** `WebSearch` and `WebFetch` are read-only and run without a prompt. `/web off` removes them for every later turn.
- **No telemetry.** Nothing is sent anywhere except the searches and fetches the model asks for.

## Reporting

If you find a way to break out of these assumptions (a tool that writes outside the project without asking, a trusting path, a command injection in the approval display), please do not open a public issue. Email the address on the GitHub profile or use GitHub's private vulnerability reporting on this repository.
