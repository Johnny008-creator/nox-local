#!/usr/bin/env sh
# Installs Obscura (https://github.com/h4ckf0r0day/obscura, Apache-2.0), the
# headless browser NØX-Local uses to read JavaScript pages and as an MCP
# connector, into ~/.nox-local/bin. Checks the release's SHA-256.
set -eu
dest="$HOME/.nox-local/bin"
case "$(uname -s)-$(uname -m)" in
  Linux-x86_64) asset=obscura-x86_64-linux.tar.gz ;;
  Linux-aarch64) asset=obscura-aarch64-linux.tar.gz ;;
  Darwin-arm64) asset=obscura-aarch64-macos.tar.gz ;;
  Darwin-x86_64) asset=obscura-x86_64-macos.tar.gz ;;
  *) echo "no Obscura build for $(uname -s) $(uname -m)"; exit 1 ;;
esac
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
api=https://api.github.com/repos/h4ckf0r0day/obscura/releases/latest
meta=$(curl -fsSL "$api")
url=$(printf '%s' "$meta" | python3 -c "import json,sys; a=[x for x in json.load(sys.stdin)['assets'] if x['name']=='$asset'][0]; print(a['browser_download_url'])")
want=$(printf '%s' "$meta" | python3 -c "import json,sys; a=[x for x in json.load(sys.stdin)['assets'] if x['name']=='$asset'][0]; print((a.get('digest') or '').removeprefix('sha256:'))")
curl -fsSL -o "$tmp/$asset" "$url"
got=$(sha256sum "$tmp/$asset" | cut -d' ' -f1)
if [ -n "$want" ] && [ "$want" != "$got" ]; then
  echo "checksum mismatch: expected $want, got $got"; exit 1
fi
mkdir -p "$dest"
tar xzf "$tmp/$asset" -C "$dest"
echo "installed $("$dest/obscura" --version) to $dest"
