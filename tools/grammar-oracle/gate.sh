#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
if [ -z "${SH_ROOT:-}" ] && [ -z "${DHNT_ROOT:-}" ]; then
  echo "FAIL: set SH_ROOT or DHNT_ROOT" >&2
  exit 2
fi
cd "$ROOT/tools/grammar-oracle"
GOWORK=off GOTOOLCHAIN="${GRAMMAR_ORACLE_TOOLCHAIN:-go1.27.2}" "${GRAMMAR_ORACLE_GO:-go}" run ./driver/cmd -root "$ROOT" -toolchain "${GRAMMAR_ORACLE_TOOLCHAIN:-go1.27.2}" "$@"
