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
  echo "PASS selftest $name"
}
fail() {
  name=$1; pattern=$2; shift 2
  if "$@" >"$work/log" 2>&1; then cat "$work/log"; echo "FAIL selftest $name accepted corruption"; exit 1; fi
  grep -q "$pattern" "$work/log" || { cat "$work/log"; echo "FAIL selftest $name wrong failure"; exit 1; }
  echo "PASS selftest $name"
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
[ "$(ls -l "$work/tools/python/ready" | awk '{print substr($1,3,1) substr($1,6,1) substr($1,9,1)}')" = --- ]
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
