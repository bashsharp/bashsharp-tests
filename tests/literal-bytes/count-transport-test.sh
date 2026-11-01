#!/usr/bin/env bash
# Sprint: #557; Story: #112; Story-ID: cebf026837c7
#
# Bounded regressions for the portable go-test case counting used by
# tests/literal-bytes/gate.sh. Each scenario feeds a tiny synthetic log
# (3 expected cases, never the full 15-case corpus and never a real `go
# test` run) through count_transport_cases and asserts the exact count on
# stdout, the exact diagnostic on stderr, and the exit status.
#
# A final guard asserts gate.sh no longer shells out to rg for counting, so
# the Hosted41334176466 class (rg missing, masked by ||true, measured
# 30/expected 60) cannot return. This test itself uses only grep -Fx, so it
# also passes on hosts without rg.
set -euo pipefail

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
lib="$root/tests/literal-bytes/count-transport.sh"
gate="$root/tests/literal-bytes/gate.sh"
[[ -s "$lib" ]] || { echo "missing counting library: $lib" >&2; exit 2; }
# shellcheck disable=SC1090
. "$lib"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# 0. The gate must not depend on rg for transport counting.
if grep -F 'rg -c' "$gate" >/dev/null 2>&1 || grep -F '$(rg' "$gate" >/dev/null 2>&1; then
  echo "FAIL no-rg-dependency (gate.sh still invokes rg for counting)" >&2
  exit 1
fi
echo "PASS no-rg-dependency"

printf 'alpha\t00\nbeta\tff\ngamma\t0a0b\n' > "$tmp/cases.tsv"

make_log() {
  # $1 = destination; remaining stdin is the log body.
  cat > "$1"
}

# run_case <name> <want_count> <want_rc> <want_stderr_line_or_empty> <log>
run_case() {
  local name="$1" want_count="$2" want_rc="$3" want_err="$4" log="$5"
  local work="$tmp/work-$name"
  mkdir -p "$work"
  local got rc=0
  set +e
  got=$(count_transport_cases "$log" "TestDemo" "$tmp/cases.tsv" "t" "$work" 2>"$tmp/$name.stderr")
  rc=$?
  set -e
  if [[ "$got" != "$want_count" ]]; then
    echo "FAIL $name (stdout count: got '$got', want '$want_count')" >&2
    exit 1
  fi
  if (( rc != want_rc )); then
    echo "FAIL $name (exit: got $rc, want $want_rc)" >&2
    exit 1
  fi
  if [[ -n "$want_err" ]]; then
    grep -Fx "$want_err" "$tmp/$name.stderr" >/dev/null || {
      echo "FAIL $name (missing exact stderr line: $want_err)" >&2
      cat "$tmp/$name.stderr" >&2
      exit 1
    }
  else
    [[ ! -s "$tmp/$name.stderr" ]] || {
      echo "FAIL $name (unexpected stderr output)" >&2
      cat "$tmp/$name.stderr" >&2
      exit 1
    }
  fi
  echo "PASS $name"
}

make_log "$tmp/all-pass.log" <<'EOF'
=== RUN   TestDemo
=== RUN   TestDemo/alpha
--- PASS: TestDemo (0.30s)
    --- PASS: TestDemo/alpha (0.10s)
    --- PASS: TestDemo/beta (0.10s)
    --- PASS: TestDemo/gamma (0.10s)
PASS
ok  	example/demo	0.311s
EOF
run_case "all-pass" "3" 0 "" "$tmp/all-pass.log"

make_log "$tmp/missing.log" <<'EOF'
=== RUN   TestDemo
    --- PASS: TestDemo/alpha (0.10s)
    --- PASS: TestDemo/beta (0.10s)
PASS
EOF
run_case "missing" "2" 1 "literal-bytes: t missing case 'gamma'" "$tmp/missing.log"

make_log "$tmp/duplicate.log" <<'EOF'
    --- PASS: TestDemo/alpha (0.10s)
    --- PASS: TestDemo/alpha (0.10s)
    --- PASS: TestDemo/beta (0.10s)
    --- PASS: TestDemo/gamma (0.10s)
EOF
run_case "duplicate" "2" 1 "literal-bytes: t duplicate pass 'alpha'" "$tmp/duplicate.log"

make_log "$tmp/unexpected.log" <<'EOF'
    --- PASS: TestDemo/alpha (0.10s)
    --- PASS: TestDemo/beta (0.10s)
    --- PASS: TestDemo/gamma (0.10s)
    --- PASS: TestDemo/delta (0.10s)
EOF
run_case "unexpected" "3" 1 "literal-bytes: t unexpected pass 'delta'" "$tmp/unexpected.log"

make_log "$tmp/failed.log" <<'EOF'
    --- PASS: TestDemo/alpha (0.10s)
    --- PASS: TestDemo/beta (0.10s)
    --- FAIL: TestDemo/gamma (0.05s)
FAIL
EOF
run_case "failed" "2" 1 "literal-bytes: t failed case 'gamma'" "$tmp/failed.log"

make_log "$tmp/skipped.log" <<'EOF'
    --- PASS: TestDemo/alpha (0.10s)
    --- PASS: TestDemo/beta (0.10s)
    --- SKIP: TestDemo/gamma (0.00s)
PASS
EOF
run_case "skipped" "2" 1 "literal-bytes: t skipped case 'gamma'" "$tmp/skipped.log"

# Unreadable log fails fast with a zero count rather than an empty string.
mkdir -p "$tmp/work-absent"
set +e
got=$(count_transport_cases "$tmp/does-not-exist.log" "TestDemo" "$tmp/cases.tsv" "t" "$tmp/work-absent" 2>"$tmp/absent.stderr")
rc=$?
set -e
[[ "$got" == "0" ]] || { echo "FAIL absent-log (stdout: got '$got', want '0')" >&2; exit 1; }
(( rc != 0 )) || { echo "FAIL absent-log (exit 0, want nonzero)" >&2; exit 1; }
grep -Fx "literal-bytes: t go-test log is absent or empty: $tmp/does-not-exist.log" "$tmp/absent.stderr" >/dev/null || {
  echo "FAIL absent-log (missing exact stderr line)" >&2; cat "$tmp/absent.stderr" >&2; exit 1; }
echo "PASS absent-log"

# A contradictory terminal result is never a measured pass.
cp "$tmp/all-pass.log" "$tmp/pass-fail.log"
printf '    --- FAIL: TestDemo/gamma (0.01s)\n' >> "$tmp/pass-fail.log"
run_case "pass-fail" "2" 1 "literal-bytes: t failed case 'gamma'" "$tmp/pass-fail.log"
cp "$tmp/all-pass.log" "$tmp/pass-skip.log"
printf '    --- SKIP: TestDemo/gamma (0.01s)\n' >> "$tmp/pass-skip.log"
run_case "pass-skip" "2" 1 "literal-bytes: t skipped case 'gamma'" "$tmp/pass-skip.log"

# Runtime tool errors must not become empty missing/unexpected lists and
# therefore apparent success. Use shell functions so no host tools change.
(
  comm() { return 7; }
  run_case "comm-error" "0" 2 "literal-bytes: t counting command failed" "$tmp/all-pass.log"
)
(
  awk() { return 7; }
  # In particular, failure on the left of awk | sort must survive sort's 0.
  run_case "pipeline-error" "0" 2 "literal-bytes: t counting command failed" "$tmp/all-pass.log"
)
(
  wc() { return 7; }
  run_case "wc-error" "0" 2 "literal-bytes: t counting command failed" "$tmp/all-pass.log"
)
(
  command() {
    if [[ "$#" -eq 2 && "$1" == -v && "$2" == wc ]]; then return 1; fi
    builtin command "$@"
  }
  run_case "missing-tool" "0" 2 "literal-bytes: t counting requires missing tool: wc" "$tmp/all-pass.log"
)

# Require the same locale for every ordered comparison, even if the caller
# supplied another locale; do not change that caller's locale or pipefail.
(
  export LC_ALL=POSIX
  for candidate in en_US.UTF-8 en_US.utf8 C.UTF-8; do
    if locale -a 2>/dev/null | grep -Fx "$candidate" >/dev/null; then
      export LC_ALL="$candidate"
      break
    fi
  done
  original_locale="$LC_ALL"
  comm() {
    [[ "$LC_ALL" == C ]] || { echo "comm received inconsistent locale" >&2; return 7; }
    builtin command comm "$@"
  }
  uniq() {
    [[ "$LC_ALL" == C ]] || { echo "uniq received inconsistent locale" >&2; return 7; }
    builtin command uniq "$@"
  }
  run_case "locale" "3" 0 "" "$tmp/all-pass.log"
  set +o pipefail
  count_transport_cases "$tmp/all-pass.log" TestDemo "$tmp/cases.tsv" t "$tmp/work-locale" > "$tmp/direct-count"
  [[ "$LC_ALL" == "$original_locale" ]] || { echo "FAIL locale leaked" >&2; exit 1; }
  [[ "$(set -o | awk '$1 == "pipefail" {print $2}')" == off ]] || { echo "FAIL pipefail leaked" >&2; exit 1; }
  echo "PASS caller-state-preserved"
)
