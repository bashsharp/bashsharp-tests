#!/usr/bin/env bash
# Builds the tools/tour Go harness (`tour`) with the pinned Go toolchain and
# prints the binary path. Sprint 155 / Story S155.9 / Story-ID 43af37063b09.
#
# Resolution mirrors the way tools/go-by-example/launch.go is built by its
# gate: `GOTOOLCHAIN=<pinned version> go env GOROOT` names the SDK
# (docs/tour/toolchain.tsv), and `<GOROOT>/bin/go build` with GOTOOLCHAIN=local
# compiles the package. The binary lives under the repository's .cache so a
# clean checkout carries only sources. Every tools/tour/*.sh wrapper sources
# this file and execs the binary.
set -euo pipefail
if [ "$(uname -s | tr '[:upper:]' '[:lower:]')" = "windows_nt" ]; then
  TOUR_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -W)"
else
  TOUR_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
fi
TOUR_HOST_GOOS="$(uname -s | tr '[:upper:]' '[:lower:]')"
TOUR_HOST_GOARCH="$(uname -m)"
TOUR_EXE_SUFFIX=""
[ "${TOUR_HOST_GOOS}" = "windows_nt" ] && TOUR_EXE_SUFFIX=".exe"
TOUR_BIN="${TOUR_ROOT_DIR}/.cache/tour/bin/tour${TOUR_EXE_SUFFIX}"

tour_pinned_version() {
  local goos goarch
  goos="${TOUR_HOST_GOOS}"
  goarch="${TOUR_HOST_GOARCH}"
  awk -F '\t' -v goos="${goos}" -v goarch="${goarch}" \
    '$1 !~ /^#/ && NF && $1 == goos && $2 == goarch { print $3; found = 1; exit } END { if (!found) exit 1 }' \
    "${TOUR_ROOT_DIR}/docs/tour/toolchain.tsv"
}

tour_bootstrap_go() {
  if command -v go >/dev/null 2>&1; then
    command -v go
    return
  fi
  command -v bashy >/dev/null 2>&1 || return 1
  local root
  root="$(bashy go env GOROOT 2>/dev/null)" || return 1
  root="$(printf '%s' "${root}" | tr '\\' '/')"
  if [ -f "${root}/bin/go.exe" ]; then
    printf '%s\n' "${root}/bin/go.exe"
  else
    printf '%s\n' "${root}/bin/go"
  fi
}

tour_build() {
  local version bootstrap goroot gobin
  version="$(tour_pinned_version)" || { echo "FATAL: no pinned Go toolchain for ${TOUR_HOST_GOOS}/${TOUR_HOST_GOARCH}" >&2; return 2; }
  bootstrap="$(tour_bootstrap_go)" || { echo "FATAL: no Go resolver is available for the pinned ${version} toolchain" >&2; return 2; }
  export TOUR_BOOTSTRAP_GO="${bootstrap}"
  goroot="$(GOTOOLCHAIN="${version}" "${bootstrap}" env GOROOT 2>/dev/null || true)"
  goroot="$(printf '%s' "${goroot}" | tr '\\' '/')"
  [ -n "${goroot}" ] || { echo "FATAL: cannot resolve GOROOT for ${version}" >&2; return 2; }
  gobin="${goroot}/bin/go${TOUR_EXE_SUFFIX}"
  [ -f "${gobin}" ] || { echo "FATAL: pinned Go binary missing at ${gobin}" >&2; return 2; }
  mkdir -p "$(dirname "${TOUR_BIN}")"
  ( cd "${TOUR_ROOT_DIR}/tools/tour" && GOTOOLCHAIN=local GOWORK=off GOFLAGS= "${gobin}" build -o "${TOUR_BIN}" . ) \
    || { echo "FATAL: cannot build tools/tour with ${gobin}" >&2; return 2; }
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  tour_build
  echo "${TOUR_BIN}"
fi
