#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
if [ -z "${SH_ROOT:-}" ] && [ -z "${DHNT_ROOT:-}" ]; then
  echo "FAIL: set SH_ROOT or DHNT_ROOT" >&2
  exit 2
fi
cd "$ROOT/tools/grammar-oracle"
GO=("${GRAMMAR_ORACLE_GO:-go}")
export GOWORK=off GOTOOLCHAIN="${GRAMMAR_ORACLE_TOOLCHAIN:-go1.27.2}"
# Materialize the frozen derived inputs (derived.tsv) into a temp root; no tracked bytes.
DERIVED="$(mktemp -d)"
trap 'rm -rf "$DERIVED"' EXIT
"${GO[@]}" run ./mutate/cmd -root "$ROOT" -out "$DERIVED"
"${GO[@]}" run ./driver/cmd -root "$ROOT" -derived-root "$DERIVED" -toolchain "$GOTOOLCHAIN" "$@"
