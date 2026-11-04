#!/bin/sh
# Deliberately a protocol stub: selftests exercise the gate, not Bash# semantics.
if [ "$1" = check ]; then
  [ "$#" = 3 ] && [ "$2" = --bashsharp ] || exit 98
  [ ! -e check-fail ] || exit 42
  : > checked
  exit 0
fi
if [ "$1" = transpile ]; then
  [ ! -e transpile-fail ] || exit 43
  if [ -e build-hang ]; then
    [ -e checked ] || exit 44
    printf '%s\n' 'package main' 'func main() { for {} }' > "$5"
    exit 0
  fi
  if [ -e build-fail ]; then
    printf '%s\n' 'not valid Go' > "$5"
    exit 0
  fi
  [ "$#" = 5 ] && [ "$2" = --bashsharp ] && [ "$4" = -o ] || exit 98
  printf '%s\n' 'package main' 'func main() { print("compiled\n") }' > "$5"
  exit 0
fi
[ "$1" = --bashsharp ] || exit 98
case "$2" in
  provision-*.bsh)
    [ ! -e "$UV_PYTHON_INSTALL_DIR/fail" ] || { echo 'fixture provisioning failed' >&2; exit 42; }
    if [ ! -e "$UV_PYTHON_INSTALL_DIR/ready" ]; then
      printf 'fixture first install\n' >&2
      printf 'ready\n' > "$UV_PYTHON_INSTALL_DIR/ready"
    fi
    exit 0 ;;
esac
[ -d "$HOME" ] && [ -d "$TMPDIR" ] || exit 97
[ ! -e touched ] || exit 96
: > touched
. "./$2"
