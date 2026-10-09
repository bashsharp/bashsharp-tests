#!/usr/bin/env bash
# A failed corpus traversal must never be accepted as an empty symlink set.
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
out=$(
  find() { return 23; }
  BASHPP_SHARP_ROOT="$ROOT"
  source "$ROOT/tools/bashsharp/validate.sh" 2>&1
)
rc=$?
if [ "$rc" -eq 0 ]; then
  echo 'FAIL: matrix validation accepted a failed find' >&2
  exit 1
fi
case "$out" in
  *'cannot inspect corpus symlinks'*) echo 'PASS: failed find is rejected' ;;
  *) printf 'FAIL: wrong refusal: %s\n' "$out" >&2; exit 1 ;;
esac
