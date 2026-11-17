#!/usr/bin/env sh
# Builds NØX-Local (TUI) and installs it to ~/.local/bin, plus the built-in
# skills to ~/.nox-local/skills.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
dest="${NOX_BIN_DIR:-$HOME/.local/bin}"
mkdir -p "$dest"
ver=$(git -C "$root" describe --tags --always --dirty 2>/dev/null || echo dev)
(cd "$root" && go build -trimpath -ldflags "-s -w -X main.commit=$ver" -o "$dest/nox-local" ./cmd/nox-local)
echo "installed $dest/nox-local ($ver)"

# Built-in skills go to ~/.nox-local/skills. Only the ones shipped here are
# replaced; skills the user installed or wrote are left alone.
skills="$HOME/.nox-local/skills"
mkdir -p "$skills"
for s in "$root"/skills/*/; do
  name=$(basename "$s")
  rm -rf "${skills:?}/$name"
  cp -R "$s" "$skills/$name"
done
echo "installed built-in skills to $skills: $(ls "$root/skills" | tr '\n' ' ')"

case ":$PATH:" in
  *":$dest:"*) ;;
  *) echo "note: $dest is not on PATH" ;;
esac
