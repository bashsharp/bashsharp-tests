#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SHELL_BIN="${BASH_SHELL_BIN-$ROOT/../bashy/bin/bash}"
AGENTOS_BIN="${BASHY_AGENTOS_BIN-$ROOT/../bashy/bin/bashy}"
[[ -n "$SHELL_BIN" && -x "$SHELL_BIN" ]] || { echo "literal-bytes: BASH_SHELL_BIN must name an executable shell: $SHELL_BIN" >&2; exit 2; }
[[ -n "$AGENTOS_BIN" && -x "$AGENTOS_BIN" ]] || { echo "literal-bytes: BASHY_AGENTOS_BIN must name an executable AgentOS binary: $AGENTOS_BIN" >&2; exit 2; }
for adapter in "$ROOT/tests/literal-bytes/ycode_adapter_test.go" "$ROOT/tests/literal-bytes/yoke_mcp_test.go"; do
  [[ -s "$adapter" ]] || { echo "literal-bytes: required adapter is absent or empty: $adapter" >&2; exit 2; }
done
tmp="$(mktemp -d)"
failed=0
cleanup() { if (( failed == 0 )); then rm -rf "$tmp"; else echo "literal-bytes: raw failure artifacts retained at $tmp" >&2; fi; }
trap cleanup EXIT
shim_home="$tmp/agent-home"
mkdir -p "$shim_home"
HOME="$shim_home" "$AGENTOS_BIN" install-agent gemini >/dev/null
shim="$shim_home/.bashy/shims/bash"
[[ -x "$shim" ]] || { echo "literal-bytes: install-agent did not create the bash shim" >&2; exit 2; }
passed=0 failed=0
case_count=0
declare -A transport_counts=([bashy-c]=0 [install-agent-shim]=0 [ycode-provider]=0 [yoke-mcp]=0)
while IFS=$'\t' read -r name expected; do
  [[ -z "$name" && -z "${expected:-}" ]] && continue
  [[ "$name" == \#* ]] && continue
  [[ -n "$name" ]] || { echo "literal-bytes: malformed case row with empty name" >&2; failed=$((failed + 1)); continue; }
  [[ -n "$expected" ]] || { echo "literal-bytes: empty expected bytes for case '$name'" >&2; failed=$((failed + 1)); continue; }
  expected="${expected// /}"
  case_count=$((case_count + 1))
  script="$ROOT/tests/literal-bytes/cases/$name.bsh"
  [[ -s "$script" ]] || { echo "literal-bytes: case '$name' is absent or empty: $script" >&2; failed=$((failed + 1)); continue; }
  actual="$tmp/$name.bin"
  reply="$(cat "$script"; printf .)"
  reply="${reply%.}"
  for transport in bashy-c install-agent-shim; do
    actual="$tmp/$name.$transport.bin"
    if [[ "$transport" == bashy-c ]]; then runner="$SHELL_BIN"; else runner="$shim"; fi
    BYTE_PACK_OUT="$actual" "$runner" -c "$reply" >"$tmp/$name.$transport.stdout" 2>"$tmp/$name.$transport.stderr" || {
      rc=$?; echo "FAIL $transport/$name (exit $rc)" >&2; cat "$tmp/$name.$transport.stdout" "$tmp/$name.$transport.stderr" >&2; failed=$((failed + 1)); continue;
    }
    got="$(od -An -tx1 -v "$actual" | tr -d ' \n')"
    if [[ "$got" != "$expected" ]]; then
      echo "FAIL $transport/$name (expected hex $expected, got $got)" >&2; failed=$((failed + 1))
    else
      echo "PASS $transport/$name"; passed=$((passed + 1)); transport_counts[$transport]=$((transport_counts[$transport] + 1))
    fi
  done
done < "$ROOT/tests/literal-bytes/cases.tsv"
[[ "$case_count" -eq 15 ]] || { echo "literal-bytes: expected exactly 15 cases, found $case_count" >&2; failed=$((failed + 1)); }
[[ "$passed" -eq $((case_count * 2)) ]] || { echo "literal-bytes: direct/shim measured $passed checks, expected $((case_count * 2))" >&2; failed=$((failed + 1)); }

# Exercise the production ycode provider adapter by overlaying the focused test
# into its package. No files in the ycode checkout are written or modified.
YCODE_DIR="${YCODE_DIR:-$ROOT/../ycode}"
test_source="$ROOT/tests/literal-bytes/ycode_adapter_test.go"
test_dest="$YCODE_DIR/internal/harness/provider/literal_bytes_s381_test.go"
overlay="$tmp/ycode-overlay.json"
ruby -rjson -e 'puts JSON.generate(Replace: {ARGV[0] => ARGV[1]})' "$test_dest" "$test_source" > "$overlay"
if ! (cd "$YCODE_DIR" && LITERAL_BYTES_ROOT="$ROOT/tests/literal-bytes" BASH_SHELL_BIN="$SHELL_BIN" go test -v -overlay="$overlay" ./internal/harness/provider -run '^TestSprint381LiteralBytes$' -count=1) >"$tmp/ycode.log" 2>&1; then
  cat "$tmp/ycode.log" >&2
  echo "FAIL ycode provider JSON tool adapter" >&2
  failed=$((failed + 1))
else
  cat "$tmp/ycode.log"
  transport_counts[ycode-provider]=$(rg -c '^    --- PASS: TestSprint381LiteralBytes/' "$tmp/ycode.log" || true)
fi
YOKE_DIR="${YOKE_DIR:-$ROOT/../yoke}"
yoke_test_source="$ROOT/tests/literal-bytes/yoke_mcp_test.go"
yoke_test_dest="$YOKE_DIR/mcp/literal_bytes_s381_test.go"
yoke_overlay="$tmp/yoke-overlay.json"
ruby -rjson -e 'puts JSON.generate(Replace: {ARGV[0] => ARGV[1]})' "$yoke_test_dest" "$yoke_test_source" > "$yoke_overlay"
if ! (cd "$YOKE_DIR" && LITERAL_BYTES_ROOT="$ROOT/tests/literal-bytes" BASH_SHELL_BIN="$SHELL_BIN" go test -v -overlay="$yoke_overlay" ./mcp -run '^TestSprint381LiteralBytesRunTool$' -count=1) >"$tmp/yoke.log" 2>&1; then
  cat "$tmp/yoke.log" >&2
  echo "FAIL yoke MCP run_tool adapter" >&2
  failed=$((failed + 1))
else
  cat "$tmp/yoke.log"
  transport_counts[yoke-mcp]=$(rg -c '^    --- PASS: TestSprint381LiteralBytesRunTool/' "$tmp/yoke.log" || true)
fi
for transport in bashy-c install-agent-shim ycode-provider yoke-mcp; do
  count="${transport_counts[$transport]}"
  [[ "$count" -eq "$case_count" ]] || { echo "literal-bytes: $transport measured $count/$case_count cases" >&2; failed=$((failed + 1)); }
done
echo "literal-bytes: cases=$case_count transports=4 measured=$((transport_counts[bashy-c] + transport_counts[install-agent-shim] + transport_counts[ycode-provider] + transport_counts[yoke-mcp])) expected=$((case_count * 4)) failed=$failed"
(( failed == 0 ))
