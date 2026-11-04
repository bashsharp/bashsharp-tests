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
# Use OS names matching the manifest (independent of architecture).
case $(uname -s) in
  Linux) platform=linux ;;
  Darwin) platform=darwin ;;
  MINGW*|MSYS*|CYGWIN*|Windows_NT) platform=windows ;;
  *) platform=$(uname -s | tr '[:upper:]' '[:lower:]') ;;
esac
awk -F '\t' -v platform="$platform" -v list="$list" '
  function fatal(message) { print "FATAL: " message; failed = 1 }
  NR == 1 {
    if ($0 != "corpus\tplatforms\tstatus") fatal("invalid corpora.tsv header")
    next
  }
  {
    if (NF != 3 || $1 !~ /^[a-z0-9-]+$/ ||
        $2 !~ /^[a-z]+(,[a-z]+)*$/ || ($3 != "pending" && $3 != "required")) {
      fatal("invalid corpora.tsv row " NR)
      next
    }
    if ($1 in listed) fatal("duplicate corpus " $1)
    listed[$1] = 1
    if ($3 == "pending") print "PENDING " $1
    if ($3 == "required" && index("," $2 ",", "," platform ",")) required[$1] = 1
  }
  END {
    if (NR == 0) fatal("empty corpora.tsv")
    while ((getline path < list) > 0) {
      name = path
      sub(/^.*\//, "", name)
      found[name] = 1
      if (!(name in listed)) fatal("unlisted corpus " name)
    }
    close(list)
    for (name in required)
      if (!(name in found)) fatal("missing required corpus " name " on " platform)
    exit failed
  }
' "$here/corpora.tsv"
while IFS= read -r corpus; do
  "$here/validate-inventory.sh" "$corpus"
  "$here/check.sh" --corpus "$corpus"
done < "$list"
echo 'PASS fence-corpus aggregate'
