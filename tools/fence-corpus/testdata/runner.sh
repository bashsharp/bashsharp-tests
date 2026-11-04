#!/bin/sh
# Deliberately a protocol stub: selftests exercise the gate, not Bash# semantics.
if [ "$1" = transpile ]; then
  [ "$#" = 5 ] && [ "$2" = --bashsharp ] && [ "$4" = -o ] || exit 98
  printf '%s\n' 'package main' 'func main() { print("compiled\n") }' > "$5"
  exit 0
fi
[ "$1" = --bashsharp ] || exit 98
[ -d "$HOME" ] && [ -d "$TMPDIR" ] || exit 97
[ ! -e touched ] || exit 96
: > touched
. "./$2"
