#!/usr/bin/env bash
# Sprint: #557; Story: #112; Story-ID: cebf026837c7
#
# Portable exact case-name accounting for tests/literal-bytes/gate.sh.
#
# Replaces `rg -c '^    --- PASS: <Prefix>/' log || true`, which miscounted
# (measured 30/expected 60, failed=2) on hosts without rg: the missing tool
# failed inside the command substitution, `|| true` masked it to an empty
# string, and the empty count failed the per-transport check with no usable
# diagnostic. This library uses only POSIX tools (awk, sort, comm, uniq, wc)
# that exist on every hosted runner, counts exact subtest names instead of
# bare lines, and distinguishes missing, duplicate, unexpected, failed, and
# skipped cases.
#
# This file is a library: sourcing it must have no side effects. It takes no
# action when executed directly.
if [[ "${BASH_SOURCE[0]:-}" == "$0" ]]; then
  echo "count-transport.sh is a library; source it from gate.sh" >&2
  exit 2
fi

# usage: count_transport_cases <go-test-log> <subtest-prefix> <cases.tsv> <transport> <workdir>
# stdout: exactly one line holding the integer count of expected cases that
#   passed exactly once (printed even on failure so callers keep an exact
#   measured/expected summary).
# stderr: one `literal-bytes:` diagnostic line per missing, duplicate,
#   unexpected, failed, or skipped case.
# exit: 0 only when every expected case passed exactly once with no failure,
#   skip, duplicate, or unexpected name; 1 on accounting mismatch; 2 when a
#   prerequisite tool is absent, an input is unreadable, or a counting
#   command fails (fail fast).
count_transport_cases() (
  # Do not leak shell options or locale into callers. In particular, callers
  # deliberately disable errexit to retain counts from a failed accounting.
  set -o pipefail
  export LC_ALL=C
  count_error() {
    echo "literal-bytes: $transport counting command failed" >&2
    printf '0\n'
  }
  local log="$1" prefix="$2" cases="$3" transport="$4" workdir="$5"

  # Explicit prerequisite fail-fast: no silent fallback when the counting
  # tools themselves are unavailable.
  local tool
  for tool in awk sort comm uniq wc; do
    if ! command -v "$tool" >/dev/null 2>&1; then
      echo "literal-bytes: $transport counting requires missing tool: $tool" >&2
      printf '0\n'
      return 2
    fi
  done
  if [[ ! -s "$log" ]]; then
    echo "literal-bytes: $transport go-test log is absent or empty: $log" >&2
    printf '0\n'
    return 2
  fi
  if [[ ! -s "$cases" ]]; then
    echo "literal-bytes: $transport cases file is absent or empty: $cases" >&2
    printf '0\n'
    return 2
  fi
  if [[ ! -d "$workdir" ]]; then
    echo "literal-bytes: $transport workdir is not a directory: $workdir" >&2
    printf '0\n'
    return 2
  fi

  local safe="${transport//[^A-Za-z0-9_.-]/_}"
  local expected="$workdir/$safe.expected"
  local parsed="$workdir/$safe.parsed"
  local pass="$workdir/$safe.pass"
  local pass_uniq="$workdir/$safe.pass-uniq"
  local fail="$workdir/$safe.fail"
  local skip="$workdir/$safe.skip"
  local dup="$workdir/$safe.dup"
  local missing="$workdir/$safe.missing"
  local unexpected="$workdir/$safe.unexpected"
  local ok="$workdir/$safe.ok"

  # Expected case names, mirroring the gate loop: skip blanks and comments.
  # Default field splitting suffices: case names contain no whitespace.
  awk '
    substr($0, 1, 1) == "#" { next }
    $0 == "" { next }
    { print $1 }
  ' "$cases" | LC_ALL=C sort > "$expected" || { count_error; return 2; }

  # Parse `go test -v` result lines with string operations only (no regex
  # engine differences across awk implementations). Emits STATUS<TAB>name.
  awk -v prefix="$prefix" '
    substr($0, 1, 8) != "    --- " { next }
    {
      status = substr($0, 9, 4)
      if (status != "PASS" && status != "FAIL" && status != "SKIP") { next }
      after = substr($0, 15)
      want = prefix "/"
      if (substr(after, 1, length(want)) != want) { next }
      name = substr(after, length(want) + 1)
      i = index(name, " (")
      if (i > 0) { name = substr(name, 1, i - 1) }
      if (name == "") { next }
      print status "\t" name
    }
  ' "$log" > "$parsed" || { count_error; return 2; }

  awk '$1 == "PASS" { print $2 }' "$parsed" | LC_ALL=C sort > "$pass" || { count_error; return 2; }
  awk '$1 == "FAIL" { print $2 }' "$parsed" | LC_ALL=C sort -u > "$fail" || { count_error; return 2; }
  awk '$1 == "SKIP" { print $2 }' "$parsed" | LC_ALL=C sort -u > "$skip" || { count_error; return 2; }
  LC_ALL=C sort -u "$pass" > "$pass_uniq" || { count_error; return 2; }
  uniq -d "$pass" > "$dup" || { count_error; return 2; }
  comm -23 "$expected" "$pass_uniq" > "$missing" || { count_error; return 2; }
  comm -13 "$expected" "$pass_uniq" > "$unexpected" || { count_error; return 2; }
  # Duplicate, failed, or skipped names are never clean measured passes.
  comm -12 "$expected" "$pass_uniq" > "$ok.tmp" || { count_error; return 2; }
  LC_ALL=C sort -u "$fail" "$skip" "$dup" > "$workdir/$safe.bad" || { count_error; return 2; }
  comm -23 "$ok.tmp" "$workdir/$safe.bad" > "$ok" || { count_error; return 2; }

  local count
  count=$(wc -l < "$ok") || { count_error; return 2; }
  count=${count//[[:space:]]/}
  [[ "$count" =~ ^[0-9]+$ ]] || { count_error; return 2; }

  local rc=0 name
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    echo "literal-bytes: $transport missing case '$name'" >&2
    rc=1
  done < "$missing"
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    echo "literal-bytes: $transport duplicate pass '$name'" >&2
    rc=1
  done < "$dup"
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    echo "literal-bytes: $transport unexpected pass '$name'" >&2
    rc=1
  done < "$unexpected"
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    echo "literal-bytes: $transport failed case '$name'" >&2
    rc=1
  done < "$fail"
  while IFS= read -r name; do
    [[ -n "$name" ]] || continue
    echo "literal-bytes: $transport skipped case '$name'" >&2
    rc=1
  done < "$skip"

  printf '%s\n' "$count"
  return "$rc"
)
