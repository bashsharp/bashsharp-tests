#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/../../.." && pwd)"
corpus="$root/python-by-example"
manifest="$corpus/cases.tsv"
bashy_bin="${BASHY:-bashy}"
limit="${PYTHON_CORPUS_SAMPLE_LIMIT:-6}"

test -s "$corpus/inventory.tsv"
test -s "$corpus/pin.tsv"
test -s "$manifest"
[ "$(sed -n '1p' "$manifest")" = $'id\tmode\tfile\tneeds\trc\txfail' ]

count=0
while IFS=$'\t' read -r id mode file needs want_rc xfail extra; do
  [ "$id" != id ] || continue
  [ -z "${extra:-}" ] && [ "$mode" = run ] && [ "$needs" = python ] && [ "$xfail" = - ]
  [ -f "$corpus/$file" ] && [ -f "$corpus/${file%.bsh}.expected" ]
  [ "$(sed -n '1p' "$corpus/${file%.bsh}.expected")" = "# rc=$want_rc" ]
  if [ "$count" -lt "$limit" ]; then
    actual="$(mktemp -d)"
    trap 'rm -rf "$actual"' RETURN
    rc=0
    "$bashy_bin" --bashpp "$corpus/$file" >"$actual/stdout" 2>"$actual/stderr" || rc=$?
    [ "$rc" = "$want_rc" ] || { echo "FAIL $id rc=$rc want=$want_rc" >&2; exit 1; }
    tail -n +2 "$corpus/${file%.bsh}.expected" >"$actual/want"
    cmp -s "$actual/want" "$actual/stdout" || { diff -u "$actual/want" "$actual/stdout" >&2; exit 1; }
    if [ -f "$corpus/${file%.bsh}.stderr" ]; then
      cmp -s "$corpus/${file%.bsh}.stderr" "$actual/stderr" || { diff -u "$corpus/${file%.bsh}.stderr" "$actual/stderr" >&2; exit 1; }
    fi
    rm -rf "$actual"
    trap - RETURN
  fi
  count=$((count + 1))
done <"$manifest"

printf 'python fence corpus: manifest %s cases; bounded sample %s PASS\n' "$count" "$limit"
