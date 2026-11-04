#!/bin/sh
set -eu
here=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
[ "$#" -le 1 ] || { echo "usage: aggregate.sh [ROOT]" >&2; exit 1; }
root=$(CDPATH= cd -- "${1:-$here/../..}" && pwd)
for script in validate-inventory.sh check.sh; do
  [ -x "$here/$script" ] || { echo "FATAL: missing fence runner $script" >&2; exit 1; }
done
# Corpora may be at repository root or under tests/examples. Discover by name,
# not cases.tsv, so a deleted manifest cannot silently remove a corpus.
# The tools and docs namesakes hold machinery/provenance, not executable corpora.
list=$(mktemp)
trap 'rm -f "$list"' EXIT HUP INT TERM
find "$root" \( -name .git -o -name .cache -o -path "$root/tools" -o -path "$root/docs" -o -path "$root/.agents" \) -prune -o \
  -type d \( -name '*-by-example' -o -name examples-fenced -o -name tour-fenced \) -print > "$list"
while IFS= read -r corpus; do
  "$here/validate-inventory.sh" "$corpus"
  "$here/check.sh" --corpus "$corpus"
done < "$list"
echo 'PASS fence-corpus aggregate'
