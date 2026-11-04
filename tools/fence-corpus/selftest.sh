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
trap 'rm -rf "$work"' EXIT HUP INT TERM
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
echo 'PASS fence-corpus selftest: all positive and negative assertions verified'
