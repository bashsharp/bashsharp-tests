#!/usr/bin/env bash
set -euo pipefail

[ "$#" -eq 1 ] || { echo "usage: $0 CASE.bsh" >&2; exit 2; }
case_file="$1"
[ -f "$case_file" ] || { echo "oracle: missing case: $case_file" >&2; exit 2; }

scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT
awk '/^~~~py as py$/ { inside=1; next } /^~~~$/ && inside { exit } inside { print }' "$case_file" >"$scratch/oracle.py"
printf '\nmain()\n' >>"$scratch/oracle.py"

cat >"$scratch/resolve.bsh" <<'EOF'
~~~py as py
import sys
def executable() -> str: return sys.executable
~~~
value := py.executable()
printf '%s\n' "$value"
EOF
python_bin="$("${BASHY:-bashy}" --bashpp "$scratch/resolve.bsh")"
case "$python_bin" in
  '$HOME/'*) python_bin="${HOME}${python_bin#\$HOME}" ;;
esac
[ -x "$python_bin" ] || { echo "oracle: fence provision did not resolve an executable CPython" >&2; exit 1; }
exec "$python_bin" "$scratch/oracle.py"
