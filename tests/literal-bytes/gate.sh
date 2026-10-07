#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BASHY_BIN="${BASHY_BIN:-$ROOT/../bashy/bin/bash}"
[[ -x "$BASHY_BIN" ]] || { echo "literal-bytes: required executable missing: $BASHY_BIN" >&2; exit 2; }
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
shim_home="$tmp/agent-home"
mkdir -p "$shim_home"
HOME="$shim_home" "$BASHY_BIN" install-agent gemini >/dev/null
shim="$shim_home/.bashy/shims/bash"
[[ -x "$shim" ]] || { echo "literal-bytes: install-agent did not create the bash shim" >&2; exit 2; }
passed=0 failed=0
adapter_cases=0
while IFS=$'\t' read -r name expected; do
  [[ -z "$name" || "$name" == \#* ]] && continue
  expected="${expected// /}"
  script="$ROOT/tests/literal-bytes/cases/$name.bsh"
  actual="$tmp/$name.bin"
  reply="$(cat "$script"; printf .)"
  reply="${reply%.}"
  for transport in bashy-c install-agent-shim; do
    actual="$tmp/$name.$transport.bin"
    if [[ "$transport" == bashy-c ]]; then runner="$BASHY_BIN"; else runner="$shim"; fi
    BYTE_PACK_OUT="$actual" "$runner" -c "$reply" >"$tmp/$name.$transport.stdout" 2>"$tmp/$name.$transport.stderr" || {
      rc=$?; echo "FAIL $transport/$name (exit $rc)" >&2; cat "$tmp/$name.$transport.stderr" >&2; failed=$((failed + 1)); continue;
    }
    got="$(od -An -tx1 -v "$actual" | tr -d ' \n')"
    if [[ "$got" != "$expected" ]]; then
      echo "FAIL $transport/$name (expected hex $expected, got $got)" >&2; failed=$((failed + 1))
    else
      echo "PASS $transport/$name"; passed=$((passed + 1))
    fi
  done
done < "$ROOT/tests/literal-bytes/cases.tsv"

# Exercise the production ycode provider adapter by overlaying the focused test
# into its package. No files in the ycode checkout are written or modified.
YCODE_DIR="${YCODE_DIR:-$ROOT/../ycode}"
test_source="$ROOT/tests/literal-bytes/ycode_adapter_test.go"
test_dest="$YCODE_DIR/internal/harness/provider/literal_bytes_s381_test.go"
overlay="$tmp/ycode-overlay.json"
printf '{"Replace":{"%s":"%s"}}\n' "$test_dest" "$test_source" > "$overlay"
if ! (cd "$YCODE_DIR" && LITERAL_BYTES_ROOT="$ROOT/tests/literal-bytes" BASHY_BIN="$BASHY_BIN" go test -overlay="$overlay" ./internal/harness/provider -run '^TestSprint381LiteralBytes$' -count=1); then
  echo "FAIL ycode provider JSON tool adapter" >&2
  failed=$((failed + 1))
else
  echo "PASS ycode provider JSON tool adapter (15 cases)"
  adapter_cases=$((adapter_cases + 15))
fi
YOKE_DIR="${YOKE_DIR:-$ROOT/../yoke}"
yoke_test_source="$ROOT/tests/literal-bytes/yoke_mcp_test.go"
yoke_test_dest="$YOKE_DIR/mcp/literal_bytes_s381_test.go"
yoke_overlay="$tmp/yoke-overlay.json"
printf '{"Replace":{"%s":"%s"}}\n' "$yoke_test_dest" "$yoke_test_source" > "$yoke_overlay"
if ! (cd "$YOKE_DIR" && LITERAL_BYTES_ROOT="$ROOT/tests/literal-bytes" BASHY_BIN="$BASHY_BIN" go test -overlay="$yoke_overlay" ./mcp -run '^TestSprint381LiteralBytesRunTool$' -count=1); then
  echo "FAIL yoke MCP run_tool adapter" >&2
  failed=$((failed + 1))
else
  echo "PASS yoke MCP run_tool adapter (15 cases)"
  adapter_cases=$((adapter_cases + 15))
fi
echo "literal-bytes: $passed direct/shim checks + $adapter_cases adapter checks, $failed failed"
(( failed == 0 ))
