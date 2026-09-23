#!/usr/bin/env bash
# Sprint: #155; Story: S155.10; Story-ID: 67bdd9fae2b3
#
# Build tools/go-by-example/gbe -- the one Go program behind every wrapper in
# this directory -- with the pinned Go toolchain, the same way the gate builds
# launch.go: the release named in docs/go-by-example/toolchain.tsv for this
# host is resolved through `GOTOOLCHAIN=<version> go env GOROOT` and its own
# bin/go does the build with GOTOOLCHAIN=local. Prints the binary path. The
# build is skipped while the sources and the pinned version are unchanged.
set -euo pipefail
if [ "$(uname -s | tr '[:upper:]' '[:lower:]')" = "windows_nt" ]; then
  ROOT="$(cd "$(dirname "$0")/../.." && pwd -W)"
else
  ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
fi
SRC="${ROOT}/tools/go-by-example/gbe"
OUT_DIR="${ROOT}/.cache/go-by-example/bin"
OUT="${OUT_DIR}/gbe"

die() { echo "FATAL: $*" >&2; exit 2; }

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
if [ "${os}" = "windows_nt" ]; then
  os=windows
  OUT="${OUT}.exe"
fi
case "${arch}" in
  x86_64) arch=amd64 ;;
  aarch64) arch=arm64 ;;
esac
version="$(awk -F '\t' -v os="${os}" -v arch="${arch}" '$1 !~ /^#/ && $1 == os && $2 == arch { print $3; exit }' "${ROOT}/docs/go-by-example/toolchain.tsv")"
[ -n "${version}" ] || die "no authenticated Go toolchain pin for ${os}/${arch}"
if command -v go >/dev/null 2>&1; then
  bootstrap_go="$(command -v go)"
elif command -v bashy >/dev/null 2>&1; then
  bootstrap_root="$(bashy go env GOROOT 2>/dev/null)" || die "bashy could not resolve its managed Go"
  bootstrap_root="$(printf '%s' "${bootstrap_root}" | tr '\\' '/')"
  bootstrap_go="${bootstrap_root}/bin/go.exe"
else
  die "no Go resolver is available for the pinned ${version} toolchain"
fi
GOROOT_PINNED="$(GOTOOLCHAIN="${version}" "${bootstrap_go}" env GOROOT)" || die "cannot resolve pinned Go toolchain ${version}"
GOROOT_PINNED="$(printf '%s' "${GOROOT_PINNED}" | tr '\\' '/')"
# The resolved bin/go must carry the reviewed digest; a same-version
# distribution build on PATH is not the pinned SDK, so the toolchain module
# GOTOOLCHAIN itself would select is tried next (digest still required).
pinned_sha="$(awk -F '\t' -v os="${os}" -v arch="${arch}" '$1 !~ /^#/ && $1 == os && $2 == arch { print $5; exit }' "${ROOT}/docs/go-by-example/toolchain.tsv")"
go_name=go
[ "${os}" = windows ] && go_name=go.exe
sha_of() {
  if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" 2>/dev/null | awk '{ print $1 }'
  else bashy sha256sum "$1" 2>/dev/null | awk '{ print $1 }'; fi
}
if [ "$(sha_of "${GOROOT_PINNED}/bin/${go_name}")" != "${pinned_sha}" ]; then
  module="$(GOTOOLCHAIN=local "${bootstrap_go}" env GOMODCACHE)/golang.org/toolchain@v0.0.1-${version}.${os}-${arch}"
  module="$(printf '%s' "${module}" | tr '\\' '/')"
  [ "$(sha_of "${module}/bin/${go_name}")" = "${pinned_sha}" ] || die "no Go toolchain with the reviewed ${version} digest ${pinned_sha}: ${GOROOT_PINNED}/bin/${go_name}"
  GOROOT_PINNED="${module}"
fi
GO="${GOROOT_PINNED}/bin/${go_name}"
[ -f "${GO}" ] || die "pinned Go toolchain has no bin/${go_name}: ${GOROOT_PINNED}"

if command -v shasum >/dev/null 2>&1; then
  stamp="$( (cd "${SRC}" && printf '%s\n' "${version}" && cat go.mod ./*.go) | shasum -a 256 | awk '{ print $1 }')"
else
  stamp="$( (cd "${SRC}" && printf '%s\n' "${version}" && cat go.mod ./*.go) | bashy sha256sum | awk '{ print $1 }')"
fi
if [ -f "${OUT}" ] && [ "$(cat "${OUT}.stamp" 2>/dev/null || true)" = "${stamp}" ]; then
  printf '%s\n' "${OUT}"
  exit 0
fi
mkdir -p "${OUT_DIR}"
if [ "${os}" = windows ]; then build_output="${OUT%.exe}.tmp.$$.exe"; else build_output="${OUT}.tmp.$$"; fi
( cd "${SRC}" && env GOTOOLCHAIN=local GOFLAGS= GOWORK=off "${GO}" build -trimpath -o "${build_output}" . ) >&2 \
  || die "cannot build tools/go-by-example/gbe with ${version}"
mv "${build_output}" "${OUT}"
printf '%s\n' "${stamp}" > "${OUT}.stamp"
printf '%s\n' "${OUT}"
