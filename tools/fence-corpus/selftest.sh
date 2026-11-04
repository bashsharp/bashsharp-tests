#!/bin/sh
set -eu
here=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
# Build once; use the same production entrypoints without repeated resolution.
bin=$(sh "$here/build.sh")
export FENCE_CORPUS=1
GBE_ROOT=$(CDPATH= cd -- "$here/../.." && pwd); export GBE_ROOT
BASHY_BIN="$here/testdata/runner.sh"; export BASHY_BIN
FENCE_TIMEOUT=2; export FENCE_TIMEOUT
work=$(mktemp -d)
trap 'chmod -R u+w "$work"; rm -rf "$work"' EXIT HUP INT TERM
cp -R "$here/testdata/pass" "$work/pass"
pass() {
  name=$1; shift
  if ! "$@" >"$work/log" 2>&1; then cat "$work/log"; echo "FAIL selftest $name"; exit 1; fi
  grep -q PASS "$work/log" || { cat "$work/log"; exit 1; }
  check_cleanup
  echo "PASS selftest $name"
}
fail() {
  name=$1; pattern=$2; shift 2
  if "$@" >"$work/log" 2>&1; then cat "$work/log"; echo "FAIL selftest $name accepted corruption"; exit 1; fi
  grep -q "$pattern" "$work/log" || { cat "$work/log"; echo "FAIL selftest $name wrong failure"; exit 1; }
  check_cleanup
  echo "PASS selftest $name"
}
check_cleanup() {
  run_area=$(sed -n 's/^Run area: //p' "$work/log" | head -n 1)
  if [ -n "$run_area" ]; then
    [ ! -e "$run_area" ] || { echo 'FAIL leaked run area'; exit 1; }
  fi
}
reset() { rm -rf "$work/case"; cp -R "$here/testdata/pass" "$work/case"; }
pass inventory "$bin" fence-validate "$work/pass"
pass bytes "$bin" fence-check --corpus "$work/pass"
pass isolation "$bin" fence-check --corpus "$work/pass"
reset
# A declared fence is provisioned once; its first-install stderr is run output.
sed 's/	-	0	-/	python	0	-/' "$work/case/cases.tsv" > "$work/cases"
cp "$work/cases" "$work/case/cases.tsv"
cat > "$work/case/chapter/hello.bsh" <<'FIXTURE'
[ -f "$UV_PYTHON_INSTALL_DIR/ready" ] || exit 94
[ "$BASHY_OFFLINE:$UV_OFFLINE:$UV_PYTHON_DOWNLOADS" = 1:1:never ] || exit 92
printf 'hello\n'
FIXTURE
printf 'second\trun\tchapter/hello.bsh\tpython\t0\t-\n' >> "$work/case/cases.tsv"
pass provision-once env FENCE_TOOLCHAIN_DIR="$work/tools" "$bin" fence-check --corpus "$work/case"
# Inspect mode bits, rather than test -w (root can write read-only files).
[ "$(ls -l "$work/tools/python/ready" | awk '{print substr($1,3,1) substr($1,6,1) substr($1,9,1)}')" = w-- ]
provision_log=$(sed -n 's/^Provision log: //p' "$work/log")
[ "$(grep -c '^fixture first install$' "$provision_log")" -eq 1 ]
grep -q '^Summary: 2 cases, 2 PASS, 0 failures$' "$work/log"
pass provision-reuse env FENCE_TOOLCHAIN_DIR="$work/tools" "$bin" fence-check --corpus "$work/case"
provision_log=$(sed -n 's/^Provision log: //p' "$work/log")
if grep -q 'fixture first install' "$provision_log"; then echo 'FAIL repeated install'; exit 1; fi
chmod u+w "$work/tools/python"
: > "$work/tools/python/fail"
fail provision-fatal 'fixture provisioning failed' env FENCE_TOOLCHAIN_DIR="$work/tools" "$bin" fence-check --corpus "$work/case"
if grep -q '^PASS case' "$work/log"; then echo 'FAIL executed after provisioning failure'; exit 1; fi
[ "$(ls -l "$work/tools/python/ready" | awk '{print substr($1,3,1)}')" = w ]
[ ! -e "$work/tools/.fence-lock" ]
# Default provisioning failures also release their run-scoped scratch/tools.
sed 's/python/unknown-language/g' "$work/case/cases.tsv" > "$work/cases"
cp "$work/cases" "$work/case/cases.tsv"
fail provision-default-cleanup 'unsupported fence need' "$bin" fence-check --corpus "$work/case"
# A busy caller directory belongs to another run: do not thaw it or remove its lease.
mkdir "$work/tools/.fence-lock"
chmod u-w "$work/tools/python/ready"
fail provision-busy 'toolchain directory busy' env FENCE_TOOLCHAIN_DIR="$work/tools" "$bin" fence-check --corpus "$work/pass"
[ -d "$work/tools/.fence-lock" ]
[ "$(ls -l "$work/tools/python/ready" | awk '{print substr($1,3,1)}')" = - ]
rmdir "$work/tools/.fence-lock"
chmod u+w "$work/tools/python/ready"
reset
cat > "$work/case/chapter/hello.bsh" <<'FIXTURE'
[ "${FENCE_LEAK-unset}:${CLAUDE_TEST-unset}:${CODEX_TEST-unset}:${BASHY_AGENT_TEST-unset}" = unset:unset:unset:unset ] || exit 91
printf 'hello\n'
FIXTURE
pass environment-allowlist env FENCE_LEAK=leak CLAUDE_TEST=leak CODEX_TEST=leak BASHY_AGENT_TEST=leak "$bin" fence-check --corpus "$work/case"
reset
cat > "$work/case/chapter/hello.bsh" <<'FIXTURE'
i=0
while [ "$i" -lt 30 ]; do
 printf 'actual-out-%s\n' "$i"
 printf 'actual-err-%s\n' "$i" >&2
 i=$((i + 1))
done
exit 7
FIXTURE
fail diagnostics 'rc actual=7, want 0' "$bin" fence-check --corpus "$work/case"
for pattern in 'stdout bytes differ' 'stderr bytes differ' 'actual-out-0' 'actual-err-0' 'excerpt limited'; do
 grep -q "$pattern" "$work/log"
done
if grep -q 'actual-out-20\|actual-err-20' "$work/log"; then echo 'FAIL unbounded diagnostic'; exit 1; fi
reset
ln -s "$work/pass/chapter/hello.bsh" "$work/case/chapter/escape"
fail symlink-isolation symlinks "$bin" fence-check --corpus "$work/case"
reset
printf '# rc=0\nwrong\n' > "$work/case/chapter/hello.expected"
fail mismatch 'stdout bytes differ' "$bin" fence-check --corpus "$work/case"
reset
# Preserve all bytes: removing only the final newline must be detected.
printf '# rc=0\nhello' > "$work/case/chapter/hello.expected"
fail trailing-newline 'stdout bytes differ' "$bin" fence-check --corpus "$work/case"
reset
sed 's/1/2/g' "$work/case/counts.tsv" > "$work/counts"
cp "$work/counts" "$work/case/counts.tsv"
fail ratchet-drop ratchet "$bin" fence-check --corpus "$work/case"
fail update-cannot-lower ratchet "$bin" fence-check --corpus "$work/case" --update-counts
cmp "$work/counts" "$work/case/counts.tsv"
reset
printf 'fixture:hello\tcase\thello\tduplicate\n' >> "$work/case/inventory.tsv"
fail inventory-duplicate duplicate "$bin" fence-validate "$work/case"
reset
sed 's/hello/absent/g' "$work/case/inventory.tsv" > "$work/inventory"
cp "$work/inventory" "$work/case/inventory.tsv"
fail inventory-unmapped 'unknown case_id' "$bin" fence-validate "$work/case"
reset
rm "$work/case/chapter/hello.expected"
fail missing-expected hello.expected "$bin" fence-check --corpus "$work/case"
reset
printf 'hello\trun\tchapter/hello.bsh\t-\t0\tbug\n' > "$work/row"
head -n 1 "$work/case/cases.tsv" > "$work/cases"
cat "$work/row" >> "$work/cases"
cp "$work/cases" "$work/case/cases.tsv"
fail xfail xfail "$bin" fence-check --corpus "$work/case"
reset
printf 'while :; do :; done\n' > "$work/case/chapter/hello.bsh"
fail timeout deadline "$bin" fence-check --corpus "$work/case"
reset
printf "printf 'oops\\\\n' >&2\nexit 7\n" > "$work/case/chapter/hello.bsh"
fail exit-status want "$bin" fence-check --corpus "$work/case"
reset
sed 's/1/0/g' "$work/case/counts.tsv" > "$work/counts"
cp "$work/counts" "$work/case/counts.tsv"
pass update-raises "$bin" fence-check --corpus "$work/case" --update-counts
pass raised-floor "$bin" fence-check --corpus "$work/case"
fail missing-runner 'missing executable runner' env BASHY_BIN="$work/absent" "$bin" fence-check --corpus "$work/pass"
fail missing-sdk 'cannot resolve' env GBE_BOOTSTRAP_GO="$work/absent" "$bin" fence-check --corpus "$work/pass"
reset
sed 's/run/compiled/g' "$work/case/cases.tsv" > "$work/cases"
cp "$work/cases" "$work/case/cases.tsv"
sed 's/run/compiled/g' "$work/case/counts.tsv" > "$work/counts"
cp "$work/counts" "$work/case/counts.tsv"
printf '# rc=0\n' > "$work/case/chapter/hello.expected"
printf 'compiled\n' > "$work/case/chapter/hello.stderr"
fail missing-runtime FENCE_SH_MODULE "$bin" fence-check --corpus "$work/case"
pass compiled env FENCE_TIMEOUT=20 FENCE_SH_MODULE="$here/testdata/runtime" "$bin" fence-check --corpus "$work/case"
# Build runs both checking and native compilation, but neither source nor
# artifact may run: both intentionally hang.
reset
sed 's/run/build/g' "$work/case/cases.tsv" > "$work/cases"
cp "$work/cases" "$work/case/cases.tsv"
sed 's/run/build/g' "$work/case/counts.tsv" > "$work/counts"
cp "$work/counts" "$work/case/counts.tsv"
printf '# rc=0\n' > "$work/case/chapter/hello.expected"
printf 'while :; do :; done\n' > "$work/case/chapter/hello.bsh"
: > "$work/case/chapter/build-hang"
pass build-never-executes env FENCE_TIMEOUT=20 FENCE_SH_MODULE="$here/testdata/runtime" "$bin" fence-check --corpus "$work/case" --mode build --update-counts
grep -q '^build' "$work/case/counts.tsv"
: > "$work/case/chapter/check-fail"
fail build-check-fails 'static check failed' env FENCE_SH_MODULE="$here/testdata/runtime" "$bin" fence-check --corpus "$work/case"
rm "$work/case/chapter/check-fail"
: > "$work/case/chapter/transpile-fail"
fail build-transpile-fails 'transpile failed' env FENCE_SH_MODULE="$here/testdata/runtime" "$bin" fence-check --corpus "$work/case"
rm "$work/case/chapter/transpile-fail" "$work/case/chapter/build-hang"
: > "$work/case/chapter/build-fail"
fail build-compiler-fails 'build failed' env FENCE_TIMEOUT=20 FENCE_SH_MODULE="$here/testdata/runtime" "$bin" fence-check --corpus "$work/case"
printf '# rc=0\nunexpected\n' > "$work/case/chapter/hello.expected"
fail build-output-contract 'build expected' "$bin" fence-check --corpus "$work/case"

# Each licence accepts its documented volatile change, rejects observable
# corruption, and cannot hide a changed process exit status.
for comparator in wallclock interleave_order pointer_address map_order duration panic_trace; do
  reset
  printf '%s\n' "$comparator" > "$work/case/chapter/hello.compare"
  case "$comparator" in
    wallclock)
      expected='at 2026-11-03T01:02:03Z'
      actual='at 2026-11-04T02:03:04Z' ;;
    pointer_address)
      expected='pointer: 0x1234'
      actual='pointer: 0xabcd' ;;
    duration)
      expected='=== RUN   TestExample
--- PASS: TestExample (0.01s)
PASS'
      actual='=== RUN   TestExample
--- PASS: TestExample (0.02s)
PASS' ;;
    map_order)
      expected='sum: 9
index: 1
a -> apple
b -> banana
key: a
key: b
0 103
1 111'
      actual='sum: 9
index: 1
b -> banana
a -> apple
key: b
key: a
0 103
1 111' ;;
    interleave_order)
      expected='direct : 0
direct : 1
direct : 2
going
goroutine : 0
goroutine : 1
goroutine : 2
done'
      actual='direct : 0
direct : 1
direct : 2
goroutine : 0
going
goroutine : 1
goroutine : 2
done' ;;
    panic_trace)
      expected='panic: boom

goroutine 1 [running]:
main.main()
  /oracle/main.go:12'
      actual='panic: boom

goroutine 9 [running]:
main.main()
  /generated/main.go:99' ;;
  esac
  printf '# rc=0\n' > "$work/case/chapter/hello.expected"
  if [ "$comparator" = panic_trace ]; then
    printf '%s\n' "$expected" > "$work/case/chapter/hello.stderr"
    printf "printf '%%s\\n' '%s' >&2\n" "$actual" > "$work/case/chapter/hello.bsh"
  else
    printf '%s\n' "$expected" >> "$work/case/chapter/hello.expected"
    printf "printf '%%s\\n' '%s'\n" "$actual" > "$work/case/chapter/hello.bsh"
  fi
  pass "compare-$comparator-volatile" "$bin" fence-check --corpus "$work/case"
  cp "$work/case/chapter/hello.bsh" "$work/original"
  printf 'exit 7\n' >> "$work/case/chapter/hello.bsh"
  fail "compare-$comparator-rc" 'want 0' "$bin" fence-check --corpus "$work/case"
  # Prefixing a changed event also tests the panic body, before its stack.
  { printf "printf 'corruption\\n' >&2\n"; cat "$work/original"; } > "$work/case/chapter/hello.bsh"
  if [ "$comparator" != panic_trace ]; then
    { printf "printf 'corruption\\n'\n"; cat "$work/original"; } > "$work/case/chapter/hello.bsh"
  fi
  fail "compare-$comparator-corruption" 'FAIL case' "$bin" fence-check --corpus "$work/case"
done
reset
printf 'unknown\n' > "$work/case/chapter/hello.compare"
fail compare-unknown 'unknown comparator' "$bin" fence-check --corpus "$work/case"

# Interrupt while a supervised case is active; cleanup follows its bounded reap.
reset
printf 'while :; do :; done\n' > "$work/case/chapter/hello.bsh"
for interrupt in TERM HUP INT; do
  "$bin" fence-check --corpus "$work/case" > "$work/log" 2>&1 &
  pid=$!
  sleep 1
  kill -"$interrupt" "$pid"
  if wait "$pid"; then echo 'FAIL interrupt accepted'; exit 1; fi
  grep -q interrupted "$work/log"
  check_cleanup
  echo "PASS selftest interrupt-$interrupt-cleanup"
done
env FENCE_TOOLCHAIN_DIR="$work/tools" "$bin" fence-check --corpus "$work/case" > "$work/log" 2>&1 &
pid=$!
sleep 1
kill -TERM "$pid"
if wait "$pid"; then echo 'FAIL interrupt accepted'; exit 1; fi
grep -q interrupted "$work/log"
check_cleanup
[ -d "$work/tools" ]
[ ! -e "$work/tools/.fence-lock" ]
[ "$(ls -l "$work/tools/python/ready" | awk '{print substr($1,3,1)}')" = w ]
echo 'PASS selftest interrupt-retained-writable'
mkdir -p "$work/aggregate"
cp -R "$here/testdata/pass" "$work/aggregate/examples-fenced"
pass aggregate sh "$here/aggregate.sh" "$work/aggregate"
printf '# rc=0\nwrong\n' > "$work/aggregate/examples-fenced/chapter/hello.expected"
fail aggregate-fatal 'stdout bytes differ' sh "$here/aggregate.sh" "$work/aggregate"
rm "$work/aggregate/examples-fenced/cases.tsv"
fail aggregate-missing-manifest cases.tsv sh "$here/aggregate.sh" "$work/aggregate"
# Isolate manifest changes from the production pending rows and use empty roots
# so these checks never depend on a corpus toolchain or the host OS.
mkdir -p "$work/gate" "$work/empty"
cp "$here/aggregate.sh" "$work/gate/aggregate.sh"
for script in validate-inventory.sh check.sh; do
  ln -s "$here/$script" "$work/gate/$script"
done
printf 'corpus\tplatforms\tstatus\nexamples-fenced\tlinux,darwin,windows\trequired\n' > "$work/gate/corpora.tsv"
fail aggregate-missing-required 'missing required corpus examples-fenced' sh "$work/gate/aggregate.sh" "$work/empty"
printf 'corpus\tplatforms\tstatus\nexamples-fenced\tlinux,darwin,windows\tpending\n' > "$work/gate/corpora.tsv"
pass aggregate-pending sh "$work/gate/aggregate.sh" "$work/empty"
grep -qx 'PENDING examples-fenced' "$work/log" || { cat "$work/log"; echo 'FAIL selftest pending not reported'; exit 1; }
mkdir -p "$work/empty/tests/go-by-example"
pass aggregate-plain-go-ignored sh "$work/gate/aggregate.sh" "$work/empty"
mkdir "$work/empty/foo-by-example"
fail aggregate-unlisted 'unlisted corpus foo-by-example' sh "$work/gate/aggregate.sh" "$work/empty"
rmdir "$work/empty/foo-by-example"
mkdir -p "$work/empty/examples/go-by-example"
fail aggregate-go-exclusion-exact-path 'unlisted corpus go-by-example' sh "$work/gate/aggregate.sh" "$work/empty"
echo 'PASS fence-corpus selftest: all positive and negative assertions verified'
