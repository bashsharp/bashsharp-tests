#!/usr/bin/env bash
set -euo pipefail
script_dir=$(cd -- "$(dirname "$0")" && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
cand=$tmp/candidates/probe
mkdir -p "$cand/sh" "$cand/bashsharp/bin"
git -C "$cand/sh" init -q
git -C "$cand/sh" config user.name Test
git -C "$cand/sh" config user.email test@example.invalid
printf 'first\n' > "$cand/sh/source"
git -C "$cand/sh" add source
git -C "$cand/sh" commit -qm first
tool=$cand/bashsharp/bin/bashsharp
printf 'first binary\n' > "$tool"
sh_sha=$(git -C "$cand/sh" rev-parse HEAD)
tool_sha=$(sha256sum "$tool" | awk '{print $1}')
printf 'sh %s\nbashsharp %s\n' "$sh_sha" "$tool_sha" > "$cand/candidate.txt"
bash "$script_dir/candidate-identity.sh" "$cand" "$tool" "$cand/sh" >/dev/null
if BASHPP_PREVIOUS_SHA256=${tool_sha:0:16} bash "$script_dir/candidate-identity.sh" "$cand" "$tool" "$cand/sh" >/dev/null 2>&1; then
  echo 'candidate identity accepted an unchanged prior interpreter' >&2; exit 1
fi
printf 'changed binary\n' > "$tool"
if bash "$script_dir/candidate-identity.sh" "$cand" "$tool" "$cand/sh" >/dev/null 2>&1; then
  echo 'candidate identity accepted binary drift' >&2; exit 1
fi
printf 'first binary\n' > "$tool"
printf 'second\n' > "$cand/sh/source"
git -C "$cand/sh" commit -qam second
if bash "$script_dir/candidate-identity.sh" "$cand" "$tool" "$cand/sh" >/dev/null 2>&1; then
  echo 'candidate identity accepted checkout drift' >&2; exit 1
fi
echo 'candidate identity guards passed'
