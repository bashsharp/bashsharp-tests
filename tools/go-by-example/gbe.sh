#!/usr/bin/env bash
# Sprint: #155; Story: S155.10; Story-ID: 67bdd9fae2b3
#
# Usage: gbe.sh <subcommand> [args...]
#
# Entry point for the Go program in tools/go-by-example/gbe: builds it with the
# pinned toolchain (build.sh) and execs the subcommand with GBE_ROOT bound to
# this checkout. gate.sh, validate.sh, refresh.sh and tamper-tests.sh keep
# their names and CLIs and route through here; the validators that had no
# wrapper (validate-candidate, validate-evidence, validate-bounded-evidence,
# summarize, tamper-retained-evidence, bounded-evidence-selftests) are reached
# as subcommands.
set -euo pipefail
if [ "$(uname -s | tr '[:upper:]' '[:lower:]')" = "windows_nt" ]; then
  ROOT="$(cd "$(dirname "$0")/../.." && pwd -W)"
else
  ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
fi
if ! command -v go >/dev/null 2>&1 && command -v bashy >/dev/null 2>&1; then
  _gbe_go_root="$(bashy go env GOROOT 2>/dev/null)"
  _gbe_go_root="$(printf '%s' "${_gbe_go_root}" | tr '\\' '/')"
  export GBE_BOOTSTRAP_GO="${_gbe_go_root}/bin/go.exe"
fi
if [ "$(uname -s | tr '[:upper:]' '[:lower:]')" = "windows_nt" ]; then
  BIN="$(bashy "${ROOT}/tools/go-by-example/build.sh")" || exit 2
else
  BIN="$("${ROOT}/tools/go-by-example/build.sh")" || exit 2
fi
GBE_ROOT="${ROOT}" exec "${BIN}" "$@"
