#!/usr/bin/env bash
# Execute a registered Python fixture in the same pinned package context as
# python-package-gate.sh. Missing prerequisites are errors, never empty imports.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
fixture="${1:?fixture path required}"
shift
BASHY="${BASHY:-${here}/../../bashy/bin/bash}"
fail() { printf 'python-package-fixture: %s\n' "$*" >&2; exit 1; }
[ -x "$BASHY" ] || fail "candidate is not executable: $BASHY"
BASHY="$(cd "$(dirname "$BASHY")" && pwd)/$(basename "$BASHY")"
fixture="$(cd "$(dirname "$fixture")" && pwd)/$(basename "$fixture")"
fixture_dir="$(cd "$here/../tests/python-packages" && pwd)"
case "$fixture" in
  "$fixture_dir/nanochat.bpp"|"$fixture_dir/nanochat-missing-attribute.bpp")
    root="${NANOCHAT_ROOT:-${HOME}/projects/poc/nanochat}"
    pin=dc54a1a3077cab11d68fac4c5d1cd5c51f5d8c7a
    ;;
  "$fixture_dir/mini-swe-agent.bpp")
    root="${MINISWE_ROOT:-${HOME}/projects/poc/mini-swe-agent}"
    pin=04d809ceab9df28f9adaed044884180159172930
    ;;
  *) fail "unregistered fixture: $fixture" ;;
esac
actual="$(git -C "$root" rev-parse HEAD 2>/dev/null)" || fail "package checkout unavailable: $root"
[ "$actual" = "$pin" ] || fail "package checkout must be $pin, got $actual"
root="$(cd "$root" && pwd)"
if [ "$pin" = dc54a1a3077cab11d68fac4c5d1cd5c51f5d8c7a ]; then
  export PYTHONPATH="$root"
else
  export PYTHONPATH="$root/src" BASHPP_PYTHON="$root/.venv/bin/python"
  [ -x "$BASHPP_PYTHON" ] || fail "pinned Python environment unavailable: $BASHPP_PYTHON"
  cd "$root"
fi
exec "$BASHY" "$@" "$fixture"
