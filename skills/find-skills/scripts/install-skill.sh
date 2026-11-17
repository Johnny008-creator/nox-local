#!/usr/bin/env sh
# Copies one skill folder from a GitHub repo into ~/.nox-local/skills/<name>.
# Usage: install-skill.sh OWNER/REPO PATH/TO/SKILL-FOLDER [--project DIR]
# It only downloads and copies files. Nothing from the skill is executed.
set -eu
repo="$1"; path="${2%/}"; dest_root="$HOME/.nox-local/skills"
if [ "${3:-}" = "--project" ]; then dest_root="${4:?project dir}/.nox/skills"; fi
case "$repo" in */*) ;; *) echo "repo must be OWNER/REPO"; exit 2 ;; esac
name=$(basename "$path"); [ "$path" = "." ] || [ -z "$path" ] && name=$(basename "$repo")
dest="$dest_root/$name"
if [ -e "$dest" ]; then echo "already installed: $dest (remove it first to reinstall)"; exit 1; fi
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
git clone --quiet --depth 1 --filter=blob:none --sparse "https://github.com/$repo.git" "$tmp/r"
if [ -n "$path" ] && [ "$path" != "." ]; then git -C "$tmp/r" sparse-checkout set --no-cone "/$path/"; else git -C "$tmp/r" sparse-checkout disable; fi
src="$tmp/r/$path"
[ -f "$src/SKILL.md" ] || { echo "no SKILL.md in $repo/$path"; exit 1; }
mkdir -p "$dest_root"
rm -rf "$src/.git"; cp -R "$src" "$dest"
commit=$(git -C "$tmp/r" rev-parse --short HEAD)
printf 'source: https://github.com/%s/tree/%s/%s\n' "$repo" "$commit" "$path" > "$dest/.source"
echo "installed $name -> $dest (from $repo@$commit)"
echo "files:"; (cd "$dest" && find . -type f ! -name .source | sort | sed 's/^\.\//  /')
echo "--- SKILL.md (first 25 lines) ---"; head -25 "$dest/SKILL.md"
