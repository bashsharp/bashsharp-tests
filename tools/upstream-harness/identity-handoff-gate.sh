#!/usr/bin/env bash
# Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
#
# Driving test for the identity handoff of the testdir backend (D8): every
# compile phase whose native argv carries upstream's own -p (and -D) hands it
# to Bash++ as --go-import-path (--go-import-base), single-file phases
# included, and the execute phase of a single-package directory program runs
# under the identity its compile phase was checked under. It replays the
# outside-corpus fixture set testdata/backend/testdata/sprint165/
# identity-handoff/ through the exact patched Go 1.27 testdir runner in both
# modes — the same overlay the corpus gate builds, so the hook is compiled
# and exercised where it really runs (Sprint 162 trap 4) — runs the hook's
# own unit test of the argv reading in that overlay, verifies the recorded
# events with backend-verify.go (which requires the recorded import_path to
# be upstream's own), and checks every root's upstream terminal, verifier
# status, first-phase admission and output against expect.tsv. The fixture
# set carries the positive controls (escape_runtime_atomic's single-file
# `-p=p` shape with and without -m; intrinsic's single-package `-p=main`
# directory shape; an ordinary errorcheck root), and the negative set (a
# recipe's own dotted -p — refused with gc's wording as the expected
# diagnostic; a `go run` root with no -p — refused, no identity invented).
# Exit 1 on any seam or expectation failure.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
harness="$root/tools/upstream-harness"
fixture="$harness/testdata/backend/testdata/sprint165/identity-handoff"
tab=$(printf '\t')

if command -v sha256sum >/dev/null 2>&1; then
	sha256() { sha256sum "$1" | awk '{print $1}'; }
else
	sha256() { shasum -a 256 "$1" | awk '{print $1}'; }
fi

go_tool=${GO127_TOOL:-go}
if test -z "${GO127_TOOL:-}"; then export GOTOOLCHAIN=go1.27.0; fi
go_version=$($go_tool version)
case "$go_version" in
	'go version go1.27.0 '*) ;;
	*) printf 'FAIL Go pin: %s\n' "$go_version" >&2; exit 1 ;;
esac
real_goroot=$($go_tool env GOROOT)
host_arch=$($go_tool env GOARCH)

bashpp_tool=${BASHPP_TOOL:-}
if test -z "$bashpp_tool"; then bashpp_tool=$(command -v bashy || true); fi
test -x "$bashpp_tool" || { printf 'FAIL Bash++ tool is unavailable: %s\n' "$bashpp_tool" >&2; exit 1; }
bashpp_version=$($bashpp_tool --version)
shellrt=${BASHPP_SHELLRT_ROOT:-}
test -n "$shellrt" && test -f "$shellrt/go.mod" && grep -q '^module mvdan.cc/sh/v3$' "$shellrt/go.mod" || {
	printf 'FAIL BASHPP_SHELLRT_ROOT must name the caller-supplied mvdan.cc/sh/v3 source tree\n' >&2
	exit 1
}

tmp=$(mktemp -d "${TMPDIR:-/tmp}/bashpp-s1650-identity.XXXXXX")
cleanup() {
	if test "${BASHPP_KEEP_EVIDENCE:-0}" = 1; then
		printf 'evidence retained: %s\n' "$tmp"
	else
		rm -rf "$tmp"
	fi
}
trap cleanup EXIT HUP INT TERM
mkdir -p "$tmp/patched/src/cmd/internal/testdir" "$tmp/goroot" "$tmp/gocache"
cp "$harness/testdata/upstream/testdir_test.go" "$tmp/patched/src/cmd/internal/testdir/testdir_test.go"
cp "$harness/testdata/instrumented/bashpp_events_test.go" "$tmp/patched/src/cmd/internal/testdir/bashpp_events_test.go"
patch -s -d "$tmp/patched" -p1 < "$harness/testdata/instrumented/testdir_test.go.patch"
patch -s -d "$tmp/patched" -p1 < "$harness/testdata/backend/testdir_test.go.patch"
patch -s -d "$tmp/patched" -p1 < "$harness/testdata/backend/events_test.go.patch"

# The fixture directory is the whole test tree the runner sees.
for entry in "$real_goroot"/*; do
	name=${entry##*/}
	if test "$name" != test; then ln -s "$entry" "$tmp/goroot/$name"; fi
done
ln -s "$fixture" "$tmp/goroot/test"

printf '{"Replace":{"%s/src/cmd/internal/testdir/testdir_test.go":"%s/patched/src/cmd/internal/testdir/testdir_test.go","%s/src/cmd/internal/testdir/bashpp_events_test.go":"%s/patched/src/cmd/internal/testdir/bashpp_events_test.go","%s/src/cmd/internal/testdir/bashpp_backend_test.go":"%s/testdata/backend/bashpp_backend_test.go"}}\n' \
	"$tmp/goroot" "$tmp" "$tmp/goroot" "$tmp" "$tmp/goroot" "$harness" > "$tmp/overlay.json"

GOROOT="$tmp/goroot" GOTOOLCHAIN=local GOCACHE="$tmp/gocache" \
	"$tmp/goroot/bin/go" build -o "$tmp/backend-verify" "$harness/backend-verify.go"

# The hook's own reading of upstream's compile argv, tested where the hook
# compiles: the patched toolchain.
GOROOT="$tmp/goroot" GOTOOLCHAIN=local GOCACHE="$tmp/gocache" \
	"$tmp/goroot/bin/go" test -count=1 -overlay="$tmp/overlay.json" cmd/internal/testdir -run='^TestCompilerIdentityReadsUpstreamArgv$' \
	> "$tmp/hook-unit.txt" 2>&1 || { cat "$tmp/hook-unit.txt" >&2; printf 'FAIL hook unit test\n' >&2; exit 1; }
grep -q '^ok' "$tmp/hook-unit.txt" || { cat "$tmp/hook-unit.txt" >&2; printf 'FAIL hook unit test did not run\n' >&2; exit 1; }

# The verifier's matrix, generated from the fixture (capability, test,
# action, sha256, companions): the fixture is source-controlled, not pinned.
matrix="$tmp/matrix.tsv"
: > "$matrix"
for file in "$fixture"/*.go; do
	test=${file##*/}
	action=$(awk 'NR == 1 { sub(/^\/\/ */, ""); print $1; exit }' "$file")
	companions="-"
	dir="${file%.go}.dir"
	if test -d "$dir"; then
		companions=$(find "$dir" -type f | sort | while read -r c; do printf '%s=%s,' "${c#"$fixture"/}" "$(sha256 "$c")"; done)
		companions=${companions%,}
	fi
	printf '%s\t%s\t%s\t%s\t%s\n' "identity-handoff-${test%.go}" "$test" "$action" "$(sha256 "$file")" "$companions" >> "$matrix"
done

run_mode() {
	mode=$1
	dir="$tmp/evidence-$mode"
	mkdir "$dir"
	while IFS="$tab" read -r capability test action want companions; do
		case_id=$(printf '%s' "$test" | tr '/.' '__')
		file_re=$(printf '%s' "$test" | sed 's/\./\\./g')
		BASHPP_TESTDIR_EVENTS="$dir/$case_id.events.jsonl" \
		BASHPP_TESTDIR_BACKEND="$mode" \
		BASHPP_TESTDIR_TOOL="$bashpp_tool" \
		BASHPP_TESTDIR_GO="$go_tool" \
		BASHPP_TESTDIR_VERSION="$bashpp_version" \
		BASHPP_SHELLRT_ROOT="$shellrt" \
		BASHY_OTEL_SPOOL="$tmp/bashy-otel.jsonl" BASHY_HINTS=0 BASHY_NO_COACH=1 \
		GOROOT="$tmp/goroot" GOTOOLCHAIN=local GOCACHE="$tmp/gocache" \
		"$tmp/goroot/bin/go" test -count=1 -json -overlay="$tmp/overlay.json" cmd/internal/testdir -run="^Test$/^$file_re$" \
		>"$dir/$case_id.go-test.json" 2>"$dir/$case_id.stderr" || true
	done < "$matrix"
	"$tmp/backend-verify" \
		-matrix "$matrix" -evidence "$dir" -mode "$mode" -version "$bashpp_version" -tool "$bashpp_tool" > "$dir/verify.txt" || true
	cat "$dir/verify.txt"
}

# upstream_output prints the go test output lines of one root, unescaped
# enough for the expectation regexps (tabs and newlines only).
upstream_output() {
	awk -v t="Test/$1" 'index($0, "\"Test\":\"" t "\"") && /"Action":"output"/ { s = $0; sub(/^.*"Output":"/, "", s); sub(/"(,"OutputType":"[a-z]+")?}$/, "", s); gsub(/\\t/, "\t", s); gsub(/\\n/, "\n", s); gsub(/\\"/, "\"", s); print s }' "$2"
}

fail=0
for mode in interpreted compiled; do
	printf 'Bash++ %s mode: %s (host %s)\n' "$mode" "$bashpp_version" "$host_arch"
	run_mode "$mode"
	# expect.tsv: root, mode, upstream terminal, verifier status (ERE),
	# whether the first backend phase exited 0 (the check/transpile was
	# admitted), an ERE the upstream output must match, one it must not.
	while IFS="$tab" read -r test emode terminal status admitted must mustnot; do
		case "$test" in ''|'#'*) continue ;; esac
		test "$emode" = "$mode" || continue
		case_id=$(printf '%s' "$test" | tr '/.' '__')
		json="$tmp/evidence-$mode/$case_id.go-test.json"
		got_terminal=$(awk -v t="Test/$test" 'BEGIN { RS = "\n" } index($0, "\"Test\":\"" t "\"") && /"Action":"(pass|fail|skip)"/ { match($0, /"Action":"[a-z]+"/); print substr($0, RSTART + 10, RLENGTH - 11); exit }' "$json")
		got_status=$(awk -v t="$test" '$NF == t { print $1; exit }' "$tmp/evidence-$mode/verify.txt")
		first_exit=$(awk '/"kind":"phase_result"/ { match($0, /"exit":-?[0-9]+/); print substr($0, RSTART + 7, RLENGTH - 7); exit }' "$tmp/evidence-$mode/$case_id.events.jsonl" 2>/dev/null || true)
		got_admitted='-'
		case "$first_exit" in '') got_admitted=none ;; 0) got_admitted=yes ;; *) got_admitted=no ;; esac
		test "$admitted" = '-' && got_admitted='-'
		output=$(upstream_output "$test" "$json")
		ok=1
		test "$got_terminal" = "$terminal" || ok=0
		printf '%s' "$got_status" | grep -Eqx "$status" || ok=0
		test "$got_admitted" = "$admitted" || ok=0
		if test "$must" != '-' && ! printf '%s\n' "$output" | grep -Eq "$must"; then ok=0; fi
		if test "$mustnot" != '-' && printf '%s\n' "$output" | grep -Eq "$mustnot"; then ok=0; fi
		if test "$ok" -ne 1; then
			printf 'FAIL %s %s: terminal %s (want %s), verifier %s (want /%s/), first phase exit 0: %s (want %s), output must /%s/ and not /%s/\n' \
				"$test" "$mode" "${got_terminal:-none}" "$terminal" "${got_status:-none}" "$status" "$got_admitted" "$admitted" "$must" "$mustnot" >&2
			printf '%s\n' "$output" | sed -n '1,40p' >&2
			sed -n '1,20p' "$tmp/evidence-$mode/$case_id.stderr" >&2 || true
			fail=1
		else
			printf 'ok   %s %s: %s %s (first phase exit 0: %s)\n' "$test" "$mode" "$got_terminal" "$got_status" "$got_admitted"
		fi
	done < "$fixture/expect.tsv"
done
if test "$fail" -ne 0; then
	printf 'FAIL S165.0 identity handoff\n' >&2
	exit 1
fi
printf 'PASS S165.0 identity handoff\n'
