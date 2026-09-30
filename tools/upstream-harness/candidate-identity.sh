#!/usr/bin/env bash
# Verify a named leaf's interpreter against the rebuild receipt. A checkout
# moved after rebuild must never be reported as a verdict on the new source.
set -euo pipefail

cand=${1:?candidate directory required}
tool=${2:?interpreter binary required}
shrt=${3:?shell-runtime checkout required}
record=$cand/candidate.txt
test -r "$record" || { echo "candidate identity: missing rebuild receipt: $record" >&2; exit 1; }
actual_sh=$(git -C "$shrt" rev-parse HEAD)
record_sh=$(awk '$1 == "sh" { print $2 }' "$record")
test -n "$record_sh" && test "$actual_sh" = "$record_sh" || {
  echo "candidate identity: shell-runtime checkout differs from rebuild receipt" >&2; exit 1;
}
actual_sha=$(sha256sum "$tool" | awk '{ print $1 }')
case $tool in
  */bashsharp/bin/bashsharp) label=bashsharp ;;
  */bashy/bin/bashy.real) label=bashy.real ;;
  *) echo "candidate identity: unexpected interpreter path: $tool" >&2; exit 1 ;;
esac
record_sha=$(awk -v label="$label" '$1 == label && length($2) == 64 { print $2 }' "$record")
test -n "$record_sha" && test "$actual_sha" = "$record_sha" || {
  echo "candidate identity: interpreter binary differs from rebuild receipt" >&2; exit 1;
}
if test -n "${BASHPP_PREVIOUS_SHA256:-}"; then
  case $BASHPP_PREVIOUS_SHA256 in
    *[!0-9a-f]* | '') echo "candidate identity: invalid prior interpreter SHA-256" >&2; exit 2 ;;
  esac
  test ${#BASHPP_PREVIOUS_SHA256} -ge 16 && test ${#BASHPP_PREVIOUS_SHA256} -le 64 || {
    echo "candidate identity: prior interpreter digest must have 16–64 hex characters" >&2; exit 2;
  }
  test "${actual_sha:0:${#BASHPP_PREVIOUS_SHA256}}" != "$BASHPP_PREVIOUS_SHA256" || {
    echo "candidate identity: interpreter hash unchanged from prior candidate" >&2; exit 1;
  }
fi
printf 'candidate identity: sh=%s bashpp=%s\n' "$actual_sh" "$actual_sha"
