#!/usr/bin/env bash
# Product-level gate for Bash++ Python, TypeScript, Rust, C, C++, and Go fences.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
BASHY="${BASHY:-${here}/../../bashy/bin/bash}"
[ -x "$BASHY" ] || { echo "polyglot-gate: bashy oracle is not executable: $BASHY" >&2; exit 1; }
command -v python3 >/dev/null || { echo "polyglot-gate: python3 is required" >&2; exit 1; }
command -v node >/dev/null || { echo "polyglot-gate: node is required" >&2; exit 1; }
command -v rustc >/dev/null || { echo "polyglot-gate: rustc is required" >&2; exit 1; }
command -v clang >/dev/null || { echo "polyglot-gate: clang is required" >&2; exit 1; }
command -v clang++ >/dev/null || { echo "polyglot-gate: clang++ is required" >&2; exit 1; }
command -v go >/dev/null || { echo "polyglot-gate: go is required" >&2; exit 1; }

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

cat >"$scratch/rust.bpp" <<'BPP'
~~~rust
pub fn add(a: i64, b: i64) -> i64 { println!("rust"); a + b }
~~~
x := add(20, 22)
echo "$x"
BPP
got="$("$BASHY" --bashpp "$scratch/rust.bpp")"
[ "$got" = $'rust\n42' ] || { printf 'polyglot-gate: Rust direct output = %q\n' "$got" >&2; exit 1; }

cat >"$scratch/rs-alias.bpp" <<'BPP'
~~~rs as native
pub fn greet(name: &str) -> String { format!("hello {name}") }
pub fn fail() -> Result<i64, String> {
    Err("negative".into())
}
~~~
message := native.greet(world)
echo "$message"
BPP
got="$("$BASHY" --bashpp "$scratch/rs-alias.bpp")"
[ "$got" = 'hello world' ] || { printf 'polyglot-gate: Rust qualified output = %q\n' "$got" >&2; exit 1; }
cat >"$scratch/rust-error.bpp" <<'BPP'
~~~rust
pub fn fail() -> Result<i64, String> { Err("negative".into()) }
~~~
failed := fail()
BPP
if "$BASHY" --bashpp "$scratch/rust-error.bpp" >"$scratch/rust-error.out" 2>"$scratch/rust-error.err"; then
	echo "polyglot-gate: Rust Result error unexpectedly succeeded" >&2
	exit 1
fi
grep -q 'negative' "$scratch/rust-error.err"

cat >"$scratch/c.bpp" <<'BPP'
~~~c
#include <stdint.h>
#include <stdio.h>
int64_t add(int64_t a, int64_t b) { puts("c"); return a + b; }
~~~
x := add(20, 22)
echo "$x"
BPP
got="$("$BASHY" --bashpp "$scratch/c.bpp")"
[ "$got" = $'c\n42' ] || { printf 'polyglot-gate: C output = %q\n' "$got" >&2; exit 1; }

cat >"$scratch/cpp.bpp" <<'BPP'
~~~cxx as native
#include <stdexcept>
#include <string>
std::string greet(const std::string& name) { return "hello "+name; }
void fail() { throw std::runtime_error("cpp boom"); }
~~~
message := native.greet(world)
echo "$message"
BPP
got="$("$BASHY" --bashpp "$scratch/cpp.bpp")"
[ "$got" = 'hello world' ] || { printf 'polyglot-gate: C++ output = %q\n' "$got" >&2; exit 1; }
cat >"$scratch/cpp-error.bpp" <<'BPP'
~~~cpp
#include <stdexcept>
void fail() { throw std::runtime_error("cpp boom"); }
~~~
fail()
BPP
if "$BASHY" --bashpp "$scratch/cpp-error.bpp" >"$scratch/cpp-error.out" 2>"$scratch/cpp-error.err"; then
	echo "polyglot-gate: C++ exception unexpectedly succeeded" >&2
	exit 1
fi
grep -q 'cpp boom' "$scratch/cpp-error.err"

mkdir -p "$scratch/goproj/version"
cat >"$scratch/goproj/go.mod" <<'MOD'
module example.local/polyglotgate

go 1.25
MOD
cat >"$scratch/goproj/version/version.go" <<'GO'
package version
const Value = "module"
GO
cat >"$scratch/goproj/go.bpp" <<'BPP'
~~~go as native
import "example.local/polyglotgate/version"
import "fmt"
func Add(a int64, b int64) int64 { fmt.Println(version.Value); return a+b }
func Blob(value []byte) []byte { return append(value, '!') }
func Checked(value int64) (int64, error) { if value < 0 { return 0, fmt.Errorf("negative") }; return value, nil }
~~~
x := native.Add(20, 22)
value := native.Blob(ok)
echo "$x:$value"
BPP
before="$(find "$scratch/goproj" -type f -print | sort | xargs shasum -a 256)"
got="$("$BASHY" --bashpp "$scratch/goproj/go.bpp")"
[ "$got" = $'module\n42:ok!' ] || { printf 'polyglot-gate: Go output = %q\n' "$got" >&2; exit 1; }
after="$(find "$scratch/goproj" -type f -print | sort | xargs shasum -a 256)"
[ "$before" = "$after" ] || { echo "polyglot-gate: Go overlay changed the checkout" >&2; exit 1; }
cat >"$scratch/goproj/go-error.bpp" <<'BPP'
~~~go
import "fmt"
func Checked(value int64) (int64, error) { return 0, fmt.Errorf("negative") }
~~~
failed := Checked(-1)
BPP
if "$BASHY" --bashpp "$scratch/goproj/go-error.bpp" >"$scratch/go-error.out" 2>"$scratch/go-error.err"; then
	echo "polyglot-gate: Go trailing error unexpectedly succeeded" >&2
	exit 1
fi
grep -q 'negative' "$scratch/go-error.err"

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
printf '%s\n' '~~~rust' | "$BASHY" --no-bashpp -n
printf '%s\n' '~~~rust' | "$BASHY" --posix -n
printf '%s\n' '~~~c' | "$BASHY" --no-bashpp -n
printf '%s\n' '~~~c' | "$BASHY" --posix -n
printf '%s\n' '~~~cpp' | "$BASHY" --no-bashpp -n
printf '%s\n' '~~~cpp' | "$BASHY" --posix -n
printf '%s\n' '~~~go' | "$BASHY" --no-bashpp -n
printf '%s\n' '~~~go' | "$BASHY" --posix -n

echo "polyglot-gate: OK — Python/TypeScript/Rust/C/C++/Go direct, qualified, mixed-runtime, errors, lazy-runtime and mode-isolation cases passed"
