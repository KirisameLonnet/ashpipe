#!/usr/bin/env bash
set -euo pipefail

start_dir="${1:-$PWD}"

find_root() {
  local dir="$1"
  while [[ "$dir" != "/" ]]; do
    if [[ -f "$dir/.ashpipe/config.yaml" ]]; then
      printf '%s\n' "$dir"
      return 0
    fi
    dir="$(dirname "$dir")"
  done
  return 1
}

root="$(find_root "$start_dir" || true)"
if [[ -z "$root" ]]; then
  echo "ashpipe: no .ashpipe/config.yaml found from $start_dir upward" >&2
  exit 2
fi

ashpipe_bin=""
if command -v ashpipe >/dev/null 2>&1; then
  ashpipe_bin="$(command -v ashpipe)"
elif [[ -x "$root/ashpipe" ]]; then
  ashpipe_bin="$root/ashpipe"
elif [[ -x "$PWD/ashpipe" ]]; then
  ashpipe_bin="$PWD/ashpipe"
fi

if [[ -z "$ashpipe_bin" ]]; then
  cat >&2 <<'EOF'
ashpipe: binary not found.

Install ashpipe into PATH or build it in this repository with:
  go build -o ashpipe .

Codex should not use SSH/MCP file tools as a fallback for portal file edits.
EOF
  exit 127
fi

cd "$root"

echo "ashpipe: workspace root: $root"
echo "ashpipe: binary: $ashpipe_bin"
echo "ashpipe: mounting portals"
"$ashpipe_bin" mount

echo
echo "ashpipe: status"
"$ashpipe_bin" status

cat <<'EOF'

ashpipe: ready for native Codex file operations.
Safety: do not manually rm -rf portal symlinks or mount directories; unmount first or use ashpipe remove.
EOF
