#!/usr/bin/env bash
# Direct Bash++ imports against the unchanged Sprint 183 Python checkouts.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
fixtures="${here}/../tests/python-packages"
BASHY="${BASHY:-${here}/../../bashy/bin/bash}"
NANOCHAT_ROOT="${NANOCHAT_ROOT:-/Users/qiangli/projects/poc/nanochat}"
MINISWE_ROOT="${MINISWE_ROOT:-/Users/qiangli/projects/poc/mini-swe-agent}"
SH_ROOT="${SH_ROOT:-${here}/../../sh}"

nanochat_sha=dc54a1a3077cab11d68fac4c5d1cd5c51f5d8c7a
miniswe_sha=04d809ceab9df28f9adaed044884180159172930

fail() {
	printf 'python-package-gate: FAIL: %s\n' "$*" >&2
	exit 1
}

scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

checkout_sha() {
	git -C "$1" rev-parse HEAD 2>/dev/null || true
}

check_pin() {
	root="$1"
	want="$2"
	name="$3"
	got="$(checkout_sha "$root")"
	[ "$got" = "$want" ] || fail "$name checkout must be $want, got ${got:-not-a-git-checkout}"
}

[ -x "$BASHY" ] || fail "Bash++ executable is unavailable: $BASHY"
for fixture in nanochat.bpp nanochat-missing-attribute.bpp nanochat-stale-handle.go mini-swe-agent.bpp; do
	[ -f "$fixtures/$fixture" ] || fail "fixture is unavailable: $fixtures/$fixture"
done

ran=0
if ! git -C "$NANOCHAT_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	printf 'python-package-gate: SKIP nanochat: optional checkout absent: %s\n' "$NANOCHAT_ROOT"
else
	check_pin "$NANOCHAT_ROOT" "$nanochat_sha" nanochat
	before="$(git -C "$NANOCHAT_ROOT" status --porcelain=v1)"
	out="$(PYTHONPATH="$NANOCHAT_ROOT" "$BASHY" --bashpp "$fixtures/nanochat.bpp")" || fail "nanochat direct-import fixture did not run"
	case "$out" in
		$'true 42\nfalse\n'*'ValueError: fixture-boom'*) ;;
		*) fail "nanochat output was $(printf %q "$out")" ;;
	esac
	[ -f "$SH_ROOT/go.mod" ] || fail "sh module is unavailable for the stale-handle API probe: $SH_ROOT"
	(
		cd "$SH_ROOT"
		PYTHONPATH="$NANOCHAT_ROOT" go run "$fixtures/nanochat-stale-handle.go" "$NANOCHAT_ROOT"
	) || fail "nanochat stale-handle API fixture did not run"

	if PYTHONPATH="$NANOCHAT_ROOT" "$BASHY" --bashpp "$fixtures/nanochat-missing-attribute.bpp" >"$scratch/nanochat-missing.out" 2>"$scratch/nanochat-missing.err"; then
		fail "nanochat missing attribute unexpectedly succeeded"
	fi
	grep -q 'sprint183_missing_attribute' "$scratch/nanochat-missing.err" || fail "missing-attribute diagnostic did not name the attribute"
	after="$(git -C "$NANOCHAT_ROOT" status --porcelain=v1)"
	[ "$after" = "$before" ] || fail "nanochat fixture changed the checkout"
	printf 'python-package-gate: PASS nanochat %s\n' "$nanochat_sha"
	ran=$((ran + 1))
fi

if ! git -C "$MINISWE_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	printf 'python-package-gate: SKIP mini-SWE-agent: optional checkout absent: %s\n' "$MINISWE_ROOT"
else
	check_pin "$MINISWE_ROOT" "$miniswe_sha" mini-SWE-agent
	python="$MINISWE_ROOT/.venv/bin/python"
	[ -x "$python" ] || fail "mini-SWE-agent pinned environment is unavailable: $python"
	before="$(git -C "$MINISWE_ROOT" status --porcelain=v1)"
	out="$(cd "$MINISWE_ROOT" && PYTHONPATH="$MINISWE_ROOT/src" BASHPP_PYTHON="$python" "$BASHY" --bashpp "$fixtures/mini-swe-agent.bpp")" || fail "mini-SWE-agent direct-import fixture did not run"
	[ "$(printf '%s\n' "$out" | tail -n 1)" = DefaultAgent ] || fail "mini-SWE-agent output did not end with DefaultAgent: $(printf %q "$out")"
	after="$(git -C "$MINISWE_ROOT" status --porcelain=v1)"
	[ "$after" = "$before" ] || fail "mini-SWE-agent fixture changed the checkout"
	printf 'python-package-gate: PASS mini-SWE-agent %s\n' "$miniswe_sha"
	ran=$((ran + 1))
fi

printf 'python-package-gate: OK — %d/2 optional unchanged-package probes ran\n' "$ran"
