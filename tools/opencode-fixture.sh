#!/usr/bin/env bash
# Shared source context for the existing Sprint 184 OpenCode API fixture.
set -euo pipefail
fixture=${1:?usage: opencode-fixture.sh FIXTURE}
root=${OPENCODE_ROOT:?set OPENCODE_ROOT to the pinned OpenCode checkout}
binary=${BASHY_BIN:?set BASHY_BIN to the Bash++ executable}
pin=e03db9bc6908f75c9334d8aa997deeaac81c0298
[[ $(git -C "$root" rev-parse HEAD) = "$pin" ]] || { echo 'OpenCode fixture: wrong checkout revision' >&2; exit 1; }
before=$(git -C "$root" status --porcelain=v1)
[[ -z "$before" ]] || { echo 'OpenCode fixture: checkout is dirty' >&2; exit 1; }
# Package self-exports and extensionless imports require the real package
# directory and stdin, exactly as in the standalone Sprint 184 gate.
out=$(cd "$root/packages/opencode" && BASHPP_TYPESCRIPT_RUNTIME=bun "$binary" --bashpp <"$fixture")
[[ "$out" = '/tmp/x.json|/tmp/x.jsonc' ]] || { printf 'OpenCode fixture: unexpected output %q\n' "$out" >&2; exit 1; }
[[ $(git -C "$root" status --porcelain=v1) = "$before" ]] || { echo 'OpenCode fixture: checkout changed' >&2; exit 1; }
printf '%s\n' "$out"
