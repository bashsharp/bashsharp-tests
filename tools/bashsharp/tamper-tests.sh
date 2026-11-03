#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
tmp="$(mktemp -d "${TMPDIR:-/tmp}/bashsharp-tamper.XXXXXX")"
trap 'rm -rf "$tmp"' EXIT

mkdir -p "${tmp}/tests" "${tmp}/tools"
cp -R "${ROOT}/tests/bashsharp" "${tmp}/tests/bashsharp"
cp -R "${ROOT}/tools/bashsharp" "${tmp}/tools/bashsharp"

run_validator() {
  BASHPP_SHARP_ROOT="${tmp}" \
  BASHPP_SHARP_MATRIX="${tmp}/tests/bashsharp/matrix.tsv" \
  BASHPP_SHARP_TOOLS_ROOT="${tmp}/tools/bashsharp" \
    "${tmp}/tools/bashsharp/validate.sh"
}
expect_rejected() {
  local label="$1"
  if run_validator >/dev/null 2>&1; then
    echo "Bash# tamper: ${label} mutation was accepted" >&2
    exit 1
  fi
}

run_validator >/dev/null

fake_log="${tmp}/executions.log"
: > "${fake_log}"
cat > "${tmp}/fake-engine" <<'EOF'
#!/usr/bin/env bash
set -u
printf 'engine\t%s\n' "$*" >> "${BASHSHARP_FAKE_LOG}"
enabled=${BASHSHARP_FAKE_DEFAULT:-false}
posix=false
fixture=''
while [ "$#" -gt 0 ]; do
  case "$1" in
    --bashpp) enabled=true ;;
    --no-bashpp) enabled=false ;;
    --posix) posix=true ;;
    -c) exit 0 ;;
    *) fixture="$1" ;;
  esac
  shift
done
family="$(basename "$PWD")"
if ${posix}; then
  if ${enabled}; then
    case "${BASHSHARP_FAKE_POSIX_MUTATION:-}" in
      stdout) printf 'changed-output\n' ;;
      stderr) printf 'changed-error\n' >&2 ;;
      status) exit 9 ;;
    esac
  fi
  printf 'posix:%s/%s\n' "${family}" "${fixture}"
  exit 8
fi
if ! ${enabled}; then
  printf 'plain:%s/%s\n' "${family}" "${fixture}"
  exit 7
fi
row="$(awk -F '\t' -v needle="${fixture}" '$1 !~ /^#/ && $4 == needle { print; exit }' cases.tsv)"
[ -n "${row}" ] || exit 99
IFS=$'\t' read -r case_id driver expectation source expected_rc stdout stderr <<< "${row}"
if [ "${expectation}" = plain-parity ]; then
  printf 'plain:%s/%s\n' "${family}" "${fixture}"
  exit 7
fi
[ "${stdout}" = empty ] || cat "${stdout}"
[ "${stderr}" = empty ] || cat "${stderr}" >&2
exit "${expected_rc}"
EOF
cat > "${tmp}/fake-bashy" <<'EOF'
#!/usr/bin/env bash
set -u
printf 'front-door\t%s\n' "$*" >> "${BASHSHARP_FAKE_LOG}"
[ "${1:-}" != transpile ] || exit 98
[ "${1:-}" = check ] || exit 99
shift
enabled=false
fixture=''
while [ "$#" -gt 0 ]; do
  case "$1" in --bashpp) enabled=true ;; *) fixture="$1" ;; esac
  shift
done
${enabled} || exit 0
row="$(awk -F '\t' -v needle="${fixture}" '$1 !~ /^#/ && $4 == needle { print; exit }' cases.tsv)"
[ -n "${row}" ] || exit 99
IFS=$'\t' read -r case_id driver expectation source expected_rc stdout stderr <<< "${row}"
[ "${stdout}" = empty ] || cat "${stdout}"
[ "${stderr}" = empty ] || cat "${stderr}" >&2
exit "${expected_rc}"
EOF
chmod +x "${tmp}/fake-engine" "${tmp}/fake-bashy"
BASHSHARP_FAKE_LOG="${fake_log}" BASH_ENGINE_BIN="${tmp}/fake-engine" BASHY_BIN="${tmp}/fake-bashy" \
  "${tmp}/tools/bashsharp/acceptance.sh" >/dev/null
# One selector probe + four executions for each of 40 cases + seven checker-off probes.
[ "$(wc -l < "${fake_log}" | tr -d ' ')" -eq 168 ] || {
  echo "Bash# tamper: Sprint 114 gate did not execute its 168-command oracle" >&2
  exit 1
}
grep -q $'^front-door\ttranspile' "${fake_log}" && {
  echo "Bash# tamper: Sprint 114 gate invoked transpile" >&2
  exit 1
}

# The same off/on contract must hold for Bashy's default-on native front door.
: > "${fake_log}"
BASHSHARP_FAKE_DEFAULT=true BASHSHARP_FAKE_LOG="${fake_log}" BASH_ENGINE_BIN="${tmp}/fake-engine" BASHY_BIN="${tmp}/fake-bashy" \
  "${tmp}/tools/bashsharp/acceptance.sh" >/dev/null
[ "$(wc -l < "${fake_log}" | tr -d ' ')" -eq 168 ] || {
  echo "Bash# tamper: default-on engine did not execute all 168 commands" >&2
  exit 1
}
# Removing explicit-off must fail positive activation, not pass by
# comparing two enabled executions or silently dropping a fixture.
sed 's/--no-bashpp "${fixture}"/"${fixture}"/' \
  "${ROOT}/tools/bashsharp/acceptance.sh" > "${tmp}/tools/bashsharp/acceptance.sh"
if BASHSHARP_FAKE_DEFAULT=true BASHSHARP_FAKE_LOG="${fake_log}" BASH_ENGINE_BIN="${tmp}/fake-engine" BASHY_BIN="${tmp}/fake-bashy" \
  "${tmp}/tools/bashsharp/acceptance.sh" >"${tmp}/default-on-negative.log" 2>&1; then
  echo "Bash# tamper: implicit selector-off oracle was accepted" >&2
  exit 1
fi
grep -Fq 'kwargs/bind-reordered: positive case does not prove selector activation' "${tmp}/default-on-negative.log"
cp "${ROOT}/tools/bashsharp/acceptance.sh" "${tmp}/tools/bashsharp/acceptance.sh"

# Reintroduce the historical cross-mode comparison, allowing it through the
# structural validator solely in this scratch negative control. The fake's
# legitimate POSIX-specific output/status must reproduce the runtime failure.
sed '/^[[:space:]]*same_result /{s/posix-off/plain/g;s/posix_off_rc/plain_rc/g;}' \
  "${ROOT}/tools/bashsharp/acceptance.sh" > "${tmp}/tools/bashsharp/acceptance.sh"
sed 's/posix-off/plain/g' "${ROOT}/tools/bashsharp/validate.sh" > "${tmp}/tools/bashsharp/validate.sh"
if BASHSHARP_FAKE_LOG="${fake_log}" BASH_ENGINE_BIN="${tmp}/fake-engine" BASHY_BIN="${tmp}/fake-bashy" \
  "${tmp}/tools/bashsharp/acceptance.sh" >"${tmp}/cross-mode-negative.log" 2>&1; then
  echo "Bash# tamper: historical cross-mode oracle unexpectedly passed" >&2
  exit 1
fi
grep -Fq 'kwargs/bind-reordered: --posix --bashpp is not inert' "${tmp}/cross-mode-negative.log"
cp "${ROOT}/tools/bashsharp/acceptance.sh" "${tmp}/tools/bashsharp/acceptance.sh"
cp "${ROOT}/tools/bashsharp/validate.sh" "${tmp}/tools/bashsharp/validate.sh"

# With POSIX held constant, each independently corrupted result must fail.
for mutation in stdout stderr status; do
  if BASHSHARP_FAKE_POSIX_MUTATION="$mutation" BASHSHARP_FAKE_LOG="${fake_log}" BASH_ENGINE_BIN="${tmp}/fake-engine" BASHY_BIN="${tmp}/fake-bashy" \
    "${tmp}/tools/bashsharp/acceptance.sh" >"${tmp}/posix-$mutation.log" 2>&1; then
    echo "Bash# tamper: changed POSIX $mutation was accepted" >&2
    exit 1
  fi
  grep -Fq 'kwargs/bind-reordered: --posix --bashpp is not inert' "${tmp}/posix-$mutation.log"
done

rm "${tmp}/tests/bashsharp/kwargs/missing-required.bpp"
expect_rejected "missing-fixture"
cp "${ROOT}/tests/bashsharp/kwargs/missing-required.bpp" "${tmp}/tests/bashsharp/kwargs/missing-required.bpp"

awk 'NR != 3' "${ROOT}/tests/bashsharp/matrix.tsv" > "${tmp}/tests/bashsharp/matrix.tsv"
expect_rejected "missing-approved-row"
cp "${ROOT}/tests/bashsharp/matrix.tsv" "${tmp}/tests/bashsharp/matrix.tsv"

awk 'BEGIN { FS=OFS="\t" } $1 == "kwargs" { $3="E" } { print }' "${ROOT}/tests/bashsharp/matrix.tsv" > "${tmp}/tests/bashsharp/matrix.tsv"
expect_rejected "weakened-row"
cp "${ROOT}/tests/bashsharp/matrix.tsv" "${tmp}/tests/bashsharp/matrix.tsv"

sed '2,$d' "${ROOT}/tests/bashsharp/null-safety/cases.tsv" > "${tmp}/tests/bashsharp/null-safety/cases.tsv"
expect_rejected "zero-execution"
cp "${ROOT}/tests/bashsharp/null-safety/cases.tsv" "${tmp}/tests/bashsharp/null-safety/cases.tsv"

awk '$1 != "mutate-alias"' "${ROOT}/tests/bashsharp/readonly/lowering.tsv" > "${tmp}/tests/bashsharp/readonly/lowering.tsv"
expect_rejected "weakened-lowering"
cp "${ROOT}/tests/bashsharp/readonly/lowering.tsv" "${tmp}/tests/bashsharp/readonly/lowering.tsv"

awk '$1 != "optional-chaining"' "${ROOT}/tests/bashsharp/rejected.tsv" > "${tmp}/tests/bashsharp/rejected.tsv"
expect_rejected "lost-rejected-exclusion"
cp "${ROOT}/tests/bashsharp/rejected.tsv" "${tmp}/tests/bashsharp/rejected.tsv"

sed 's/same_result "${stem}.posix-off.out" "${stem}.posix.out"/same_result "${stem}.enabled.out" "${stem}.posix.out"/' \
  "${ROOT}/tools/bashsharp/acceptance.sh" > "${tmp}/tools/bashsharp/acceptance.sh"
chmod +x "${tmp}/tools/bashsharp/acceptance.sh"
expect_rejected "wrong-POSIX-oracle"
cp "${ROOT}/tools/bashsharp/acceptance.sh" "${tmp}/tools/bashsharp/acceptance.sh"

mv "${tmp}/tools/bashsharp/lowering.sh" "${tmp}/tools/bashsharp/lowering.removed"
expect_rejected "lost-lowering-phase"
mv "${tmp}/tools/bashsharp/lowering.removed" "${tmp}/tools/bashsharp/lowering.sh"

printf '%s\n' "command printf '%s\\n' 'type Color enum { Red; Green }'" > "${tmp}/tests/bashsharp/enums/forced-command.bpp"
expect_rejected "fake-forced-shell"
cp "${ROOT}/tests/bashsharp/enums/forced-command.bpp" "${tmp}/tests/bashsharp/enums/forced-command.bpp"

touch "${tmp}/tests/bashsharp/kwargs/unreferenced.bpp"
expect_rejected "open-denominator"
rm "${tmp}/tests/bashsharp/kwargs/unreferenced.bpp"

ln -s bind-reordered.bpp "${tmp}/tests/bashsharp/kwargs/symlink.bpp"
expect_rejected "symlink"

echo "Bash# tamper tests OK — fixtures, execution denominator, rows, POSIX oracle, escapes, and lowering fail closed"
