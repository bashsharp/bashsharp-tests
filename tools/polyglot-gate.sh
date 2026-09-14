#!/usr/bin/env bash
# Product-level gate for Bash++ naked Python source fences.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
BASHY="${BASHY:-${here}/../../bashy/bin/bash}"
[ -x "$BASHY" ] || { echo "polyglot-gate: bashy oracle is not executable: $BASHY" >&2; exit 1; }
command -v python3 >/dev/null || { echo "polyglot-gate: python3 is required" >&2; exit 1; }

scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

cat >"$scratch/direct.bpp" <<'BPP'
~~~python
def add(a: int, b: int) -> int:
    print("worker-out")
    return a + b
def answer() -> int:
    return 42
~~~
x := add(20, 22)
y := answer()
echo "$x:$y"
BPP
got="$("$BASHY" --bashpp "$scratch/direct.bpp")"
[ "$got" = $'worker-out\n42:42' ] || { printf 'polyglot-gate: direct output = %q\n' "$got" >&2; exit 1; }

cat >"$scratch/qualified.bpp" <<'BPP'
~~~python as py
def loose(value):
    return value + "!"
def fail():
    raise ValueError("boom")
~~~
value, callErr := py.loose(ok)
failed, failErr := py.fail()
echo "$value:${callErr:+unexpected}:${failErr:+caught}"
env | grep '^py=' || true
BPP
got="$("$BASHY" --bashpp "$scratch/qualified.bpp")"
[ "$got" = 'ok!::caught' ] || { printf 'polyglot-gate: qualified output = %q\n' "$got" >&2; exit 1; }

# The runtime is discovered only when a foreign block is prepared. A plain
# Bash++ script succeeds with an empty PATH; a Python fence fails explicitly.
PATH=/nonexistent "$BASHY" --bashpp -c 'echo lazy' >"$scratch/lazy.out"
[ "$(<"$scratch/lazy.out")" = lazy ] || exit 1
if PATH=/nonexistent "$BASHY" --bashpp "$scratch/direct.bpp" >"$scratch/missing.out" 2>"$scratch/missing.err"; then
	echo "polyglot-gate: missing Python unexpectedly succeeded" >&2
	exit 1
fi
grep -q 'Python runtime unavailable' "$scratch/missing.err"

# Other language modes retain the ordinary shell interpretation of the Class-E
# opening line. Parse-only avoids trying to execute that ordinary command.
printf '%s\n' '~~~python' | "$BASHY" --no-bashpp -n
printf '%s\n' '~~~python' | "$BASHY" --posix -n

echo "polyglot-gate: OK — direct, qualified, dynamic, hidden-state, lazy-runtime and mode-isolation cases passed"
