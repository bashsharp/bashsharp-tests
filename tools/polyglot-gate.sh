#!/usr/bin/env bash
# Product-level gate for Bash++ naked Python and TypeScript source fences.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
BASHY="${BASHY:-${here}/../../bashy/bin/bash}"
[ -x "$BASHY" ] || { echo "polyglot-gate: bashy oracle is not executable: $BASHY" >&2; exit 1; }
command -v python3 >/dev/null || { echo "polyglot-gate: python3 is required" >&2; exit 1; }
command -v node >/dev/null || { echo "polyglot-gate: node is required" >&2; exit 1; }

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

# `py` is an alias spelling of `python` (as `ts` is of `typescript`); the
# `~~~py as py` + `py.main()` shape is the documented launcher form.
cat >"$scratch/py-alias.bpp" <<'BPP'
~~~py as py
def main() -> str:
    return "launched"
~~~
value := py.main()
echo "$value"
BPP
got="$("$BASHY" --bashpp "$scratch/py-alias.bpp")"
[ "$got" = 'launched' ] || { printf 'polyglot-gate: py alias output = %q\n' "$got" >&2; exit 1; }

cat >"$scratch/typescript.bpp" <<'BPP'
~~~python as py
def twice(value: int) -> int:
    return value * 2
~~~
~~~typescript as ts
interface Pair { left: number; right: number }
type Numeric = number
export function add(a: Numeric, b: Numeric): number { console.log("typescript"); return a + b }
function answer(): number { return 42 }
~~~
x := ts.add(20, 22)
y := ts.answer()
z := py.twice(3)
echo "$x:$y:$z"
BPP
got="$("$BASHY" --bashpp "$scratch/typescript.bpp")"
[ "$got" = $'typescript\n42:42:6' ] || { printf 'polyglot-gate: TypeScript output = %q\n' "$got" >&2; exit 1; }

# A relative import may name its .ts file explicitly — the spelling Node's
# native type stripping requires — whatever the project's tsconfig says; the
# fence's checking program must not refuse it (TS5097). The project is the
# source file's directory (an ESM package.json + one .ts module); the same
# default compiler as the row above is used, and the runtime stays Node.
mkdir -p "$scratch/tsproj/src"
printf '%s\n' '{"type":"module"}' >"$scratch/tsproj/package.json"
printf '%s\n' 'export function compact(value: number): string { return value >= 1000 ? (value / 1000) + "k" : String(value) }' >"$scratch/tsproj/src/format.ts"
cat >"$scratch/tsproj/program.bpp" <<'BPP'
~~~ts as ts
import { compact } from "./src/format.ts"
export function launch(n: number): string { return compact(n) }
~~~
value := ts.launch(1500)
echo "$value"
BPP
got="$("$BASHY" --bashpp "$scratch/tsproj/program.bpp")"
[ "$got" = '1.5k' ] || { printf 'polyglot-gate: explicit .ts import output = %q\n' "$got" >&2; exit 1; }

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
printf '%s\n' '~~~typescript' | "$BASHY" --no-bashpp -n
printf '%s\n' '~~~typescript' | "$BASHY" --posix -n

echo "polyglot-gate: OK — Python/TypeScript direct, qualified, mixed-runtime, explicit-.ts-import, hidden-state, lazy-runtime and mode-isolation cases passed"
