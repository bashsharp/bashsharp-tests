#!/usr/bin/env bash
# Sprint 184 product gate for project-aware TypeScript fences. It is strictly
# read-only with respect to every fixture and never invokes a package manager.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
fixtures="${here}/../tests/typescript-workspaces"
DHNT_ROOT="${DHNT_ROOT:-/Users/qiangli/projects/poc/dhnt}"
SH_ROOT="${SH_ROOT:-${DHNT_ROOT}/sh}"
BASHY_ROOT="${BASHY_ROOT:-${DHNT_ROOT}/bashy}"
OPENCODE_ROOT="${OPENCODE_ROOT:-${DHNT_ROOT}/ycode/priorart/opencode}"
BASHPP_BUN="${BASHPP_BUN:-/Users/qiangli/.bun/bin/bun}"
NODE="${BASHPP_NODE:-$(command -v node || true)}"
COMPILER="${BASHPP_TYPESCRIPT_MODULE:-${OPENCODE_ROOT}/node_modules/typescript}"
opencode_sha=e03db9bc6908f75c9334d8aa997deeaac81c0298

fail() { printf 'sprint184-typescript-gate: FAIL: %s\n' "$*" >&2; exit 1; }
[ -f "${SH_ROOT}/go.mod" ] || fail "canonical sh sibling is unavailable: ${SH_ROOT}"
[ -f "${BASHY_ROOT}/go.mod" ] || fail "bashy sibling is unavailable: ${BASHY_ROOT}"
[ -x "$NODE" ] || fail "Node is unavailable: ${NODE:-unset}"
[ -x "$BASHPP_BUN" ] || fail "Bun is unavailable: $BASHPP_BUN"
[ -f "${COMPILER}/package.json" ] || fail "official TypeScript module is unavailable: $COMPILER"
[ "$(git -C "$OPENCODE_ROOT" rev-parse HEAD 2>/dev/null || true)" = "$opencode_sha" ] || fail "OpenCode must be pinned at $opencode_sha"

scratch="$(mktemp -d)"
cleanup() { rm -rf "$scratch"; }
trap cleanup EXIT

# Compile the canonical engine and both shell front doors which consume it.
# Direct Go builds avoid bashy's optional web-asset targets and package managers.
(cd "$SH_ROOT" && GOPROXY=off BASHPP_TYPESCRIPT_MODULE="$COMPILER" go test ./polyglot ./interp ./lower -run 'TypeScript|Environment|PythonFence')
(cd "$BASHY_ROOT" && GOPROXY=off go build -o "$scratch/bash" ./cmd/bash && GOPROXY=off go build -o "$scratch/bashy" ./cmd/bashy)
BASH="$scratch/bash"
BASHY="$scratch/bashy"
[ -x "$BASH" ] && [ -x "$BASHY" ] || fail "bashy binaries were not built"

before_fixtures="$(git -C "${here}/.." status --porcelain=v1 -- tests/typescript-workspaces)"
before_opencode="$(git -C "$OPENCODE_ROOT" status --porcelain=v1)"
[ -z "$before_opencode" ] || fail "OpenCode checkout must be clean before the gate"

run_interpreted() {
	layout="$1" runtime="$2" expected="$3"
	program="${fixtures}/${layout}/program.bpp"
	if [ "$runtime" = bun ]; then
		got="$(BASHPP_TYPESCRIPT_MODULE="$COMPILER" BASHPP_TYPESCRIPT_RUNTIME=bun BASHPP_BUN="$BASHPP_BUN" "$BASH" --bashpp "$program")"
	else
		got="$(BASHPP_TYPESCRIPT_MODULE="$COMPILER" BASHPP_NODE="$NODE" "$BASH" --bashpp "$program")"
	fi
	[ "$got" = "$expected" ] || fail "$layout/$runtime interpreted output was $(printf %q "$got")"
}

run_native() {
	layout="$1" runtime="$2" expected="$3"
	outdir="${scratch}/${layout}-${runtime}"
	mkdir -p "$outdir"
	cp -R "${fixtures}/${layout}/." "$outdir/"
	program="$outdir/program.bpp"
	printf 'module sprint184fixture\n\ngo 1.25\n\nrequire mvdan.cc/sh/v3 v3.0.0\nreplace mvdan.cc/sh/v3 => %s\n' "$SH_ROOT" >"$outdir/go.mod"
	if [ "$runtime" = bun ]; then
		(cd "$outdir" && GOPROXY=off GOFLAGS=-mod=mod BASHPP_TYPESCRIPT_MODULE="$COMPILER" BASHPP_TYPESCRIPT_RUNTIME=bun BASHPP_BUN="$BASHPP_BUN" "$BASHY" transpile --bashpp "$program" -o "$outdir/generated.go")
	else
		(cd "$outdir" && GOPROXY=off GOFLAGS=-mod=mod BASHPP_TYPESCRIPT_MODULE="$COMPILER" BASHPP_NODE="$NODE" "$BASHY" transpile --bashpp "$program" -o "$outdir/generated.go")
	fi
	(cd "$outdir" && GOWORK=off GOPROXY=off go build -mod=mod -o program generated.go)
	got="$("$outdir/program")"
	[ "$got" = "$expected" ] || fail "$layout/$runtime native output was $(printf %q "$got")"
	[ ! -e "$outdir/lifecycle-ran" ] || fail "$layout/$runtime native lifecycle script ran"
}

for layout in npm pnpm; do
	run_interpreted "$layout" node "$layout:node:ready"
	run_native "$layout" node "$layout:node:ready"
done
# A Bun lock selects a manager, never a runtime: Node remains usable, while the
# explicit Bun selection exercises the required Bun-managed path.
run_interpreted bun node 'bun:node:ready'
run_interpreted bun bun 'bun:bun:ready'
run_native bun bun 'bun:bun:ready'

# Use stdin from the package directory so unchanged package self-exports and
# extensionless internal imports resolve in their real source context.
got="$(cd "${OPENCODE_ROOT}/packages/opencode" && BASHPP_TYPESCRIPT_MODULE="$COMPILER" BASHPP_TYPESCRIPT_RUNTIME=bun BASHPP_BUN="$BASHPP_BUN" "$BASH" --bashpp <"${fixtures}/opencode.bpp")"
[ "$got" = '/tmp/x.json|/tmp/x.jsonc' ] || fail "OpenCode API output was $(printf %q "$got")"
version="$(cd "$OPENCODE_ROOT" && "$BASHPP_BUN" packages/opencode/src/index.ts --version)"
[ "$version" = local ] || fail "OpenCode CLI version was $(printf %q "$version")"

for layout in npm pnpm bun; do
	[ ! -e "${fixtures}/${layout}/lifecycle-ran" ] || fail "$layout lifecycle script ran"
done
after_fixtures="$(git -C "${here}/.." status --porcelain=v1 -- tests/typescript-workspaces)"
after_opencode="$(git -C "$OPENCODE_ROOT" status --porcelain=v1)"
[ "$after_fixtures" = "$before_fixtures" ] || fail "workspace fixtures changed"
[ "$after_opencode" = "$before_opencode" ] || fail "OpenCode checkout changed"

echo 'sprint184-typescript-gate: OK — npm/pnpm Node, Bun manager/runtime independence, interpreted/native parity, async calls, OpenCode API/CLI, and unchanged fixtures passed'
