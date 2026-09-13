#!/usr/bin/env bash
# Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
#
# Driving test for the two identities and the file roles of the cmd/go
# package backend (D8): the library transpile checks the tested package's
# files under the tested package's OWN identity (--go-import-path <pkg>, what
# cmd/go compiles ptest under) and never asserts the TestMain fact; the
# generated _testmain.go alone runs under cmd/go's testmain identity
# (<pkg>.test) with --go-test-main; every file is handed once under cmd/go's
# own role for it (--go-file / --go-test-file / --go-xtest-file), so a package
# whose tests are all external (cmd/internal/testdir) is one xtest unit that
# forms the external test package by that role. It replays the outside-corpus
# fixture module testdata/go-backend/testdata/sprint165/package-identity/
# through the exact patched Go 1.27 cmd/go in both modes — the same overlay
# package-gate.sh builds, so the hook is compiled and exercised where it
# really runs — runs the hook's own unit tests in that overlay, verifies the
# recorded plans with package-verify.go (which requires both identities and
# the roles), and checks every package's upstream terminal, verifier status
# and output against expect.tsv. The fixture set: `lib` (the full three-role
# shape, the tested package the parent of its own internal tree), `cmdmain`
# (cmd/compile's shape: a main package with an in-package test) and `xonly`
# (cmd/internal/testdir's shape: external tests only). In every one the
# internal import is granted to <pkg> and refused to <pkg>.test, so a library
# checked under the testmain identity surfaces as `could not import` — the
# line no row may carry. Exit 1 on any seam or expectation failure.
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
harness="$root/tools/upstream-harness"
fixture="$harness/testdata/go-backend/testdata/sprint165/package-identity"
tab=$(printf '\t')

if command -v sha256sum >/dev/null 2>&1; then
	sha256() { sha256sum "$1" | awk '{print $1}'; }
	sha256_stdin() { sha256sum | awk '{print $1}'; }
else
	sha256() { shasum -a 256 "$1" | awk '{print $1}'; }
	sha256_stdin() { shasum -a 256 | awk '{print $1}'; }
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
inherited_shellopts=$(env | sed -n 's/^SHELLOPTS=//p')
case ":$inherited_shellopts:" in
	*:posix:*) printf 'FAIL SHELLOPTS selects POSIX mode; run this gate from a non-POSIX environment\n' >&2; exit 1 ;;
esac
inherited_posix=$(env | sed -n '/^POSIXLY_CORRECT=/p; /^POSIX_PEDANTIC=/p')
test -z "$inherited_posix" || { printf 'FAIL inherited POSIX selector blocks the Bash++ direct Go-source interface\n' >&2; exit 1; }
shellrt=${BASHPP_SHELLRT_ROOT:-}
test -n "$shellrt" && test -f "$shellrt/go.mod" && grep -q '^module mvdan.cc/sh/v3$' "$shellrt/go.mod" || {
	printf 'FAIL BASHPP_SHELLRT_ROOT must name the caller-supplied mvdan.cc/sh/v3 source tree\n' >&2
	exit 1
}

tmp=$(mktemp -d "${TMPDIR:-/tmp}/bashpp-s1650-pkgid.XXXXXX")
cleanup() {
	if test "${BASHPP_KEEP_EVIDENCE:-0}" = 1; then
		printf 'evidence retained: %s\n' "$tmp"
	else
		rm -rf "$tmp"
	fi
}
trap cleanup EXIT HUP INT TERM
mkdir -p "$tmp/patched/src/cmd/go/internal/test" "$tmp/gocache" "$tmp/goroot"
# Overlays may not replace files beneath GOMODCACHE; mirror the SDK by symlink.
for entry in "$real_goroot"/*; do ln -s "$entry" "$tmp/goroot/${entry##*/}"; done
go127="$tmp/goroot/bin/go"
cp "$harness/testdata/upstream-go/test.go" "$tmp/patched/src/cmd/go/internal/test/test.go"
patch -s -d "$tmp/patched" -p1 < "$harness/testdata/go-backend/test.go.patch"
printf '{"Replace":{"%s/src/cmd/go/internal/test/test.go":"%s/patched/src/cmd/go/internal/test/test.go","%s/src/cmd/go/internal/test/bashpp_backend.go":"%s/testdata/go-backend/bashpp_backend.go","%s/src/cmd/go/internal/test/bashpp_backend_test.go":"%s/testdata/go-backend/bashpp_backend_test.go"}}\n' \
	"$tmp/goroot" "$tmp" "$tmp/goroot" "$harness" "$tmp/goroot" "$harness" > "$tmp/overlay.json"

# The hook's own unit tests (roles, identities, the overlay script), run
# where the hook compiles: the patched toolchain.
GOROOT="$tmp/goroot" GOTOOLCHAIN=local GOCACHE="$tmp/gocache" \
	"$go127" test -count=1 -overlay="$tmp/overlay.json" cmd/go/internal/test -run='^Test(LibraryOverlayUsesOneInvocationAndAllFileClasses|TestVariantFilesClassifyEachFileOnce|IdentitiesSeparateTheLibraryFromTheTestMain)$' -v \
	> "$tmp/hook-unit.txt" 2>&1 || { cat "$tmp/hook-unit.txt" >&2; printf 'FAIL hook unit tests\n' >&2; exit 1; }
for name in TestLibraryOverlayUsesOneInvocationAndAllFileClasses TestTestVariantFilesClassifyEachFileOnce TestIdentitiesSeparateTheLibraryFromTheTestMain; do
	grep -q "^--- PASS: $name " "$tmp/hook-unit.txt" || { cat "$tmp/hook-unit.txt" >&2; printf 'FAIL hook unit test %s did not run\n' "$name" >&2; exit 1; }
done
printf 'PASS hook unit tests in the patched cmd/go overlay\n'

# The patched go command IS the runner; it is built once through -overlay.
GOROOT="$tmp/goroot" GOTOOLCHAIN=local GOCACHE="$tmp/gocache" "$go127" build -overlay="$tmp/overlay.json" -o "$tmp/go-bashpp" cmd/go
GOROOT="$tmp/goroot" GOTOOLCHAIN=local GOCACHE="$tmp/gocache" "$go127" build -o "$tmp/package-verify" "$harness/package-verify.go"

# The verifier's matrix, generated from the fixture (capability, package,
# action, sha256, companions): the fixture is source-controlled, not pinned.
# A package is one directory of the module with test files.
module=$(awk '$1 == "module" { print $2; exit }' "$fixture/go.mod")
matrix="$tmp/matrix.tsv"
: > "$matrix"
for dir in "$fixture"/*/; do
	dir=${dir%/}
	name=${dir##*/}
	ls "$dir"/*_test.go >/dev/null 2>&1 || continue
	digest=$(for f in "$dir"/*.go; do printf '%s %s\n' "$(sha256 "$f")" "${f##*/}"; done | sort -k2 | sha256_stdin)
	printf '%s\t%s\t%s\t%s\t%s\n' "package-identity-$name" "$module/$name" package "$digest" - >> "$matrix"
done
test -s "$matrix" || { printf 'FAIL fixture has no packages with tests\n' >&2; exit 1; }

terminal() { awk -v pkg="$2" 'BEGIN{FS="\""} /"Action":"(pass|fail|skip)"/ && !/"Test":/ { for (i = 1; i <= NF; i++) if ($i == "Package" && $(i+2) == pkg) for (j = 1; j <= NF; j++) if ($j == "Action") print $(j+2) }' "$1"; }

# Native-equivalence canary on the fixture: native go vs the patched go with
# the backend off must reach the same package terminal.
canary="$module/lib"
mkdir -p "$tmp/evidence-native"
(cd "$fixture" && GOROOT="$tmp/goroot" GOTOOLCHAIN=local GOCACHE="$tmp/gocache" GOWORK=off "$go127" test -count=1 -json "$canary" > "$tmp/evidence-native/canary-go.json" 2>/dev/null || true)
(cd "$fixture" && GOROOT="$tmp/goroot" GOTOOLCHAIN=local GOCACHE="$tmp/gocache" GOWORK=off "$tmp/go-bashpp" test -count=1 -json "$canary" > "$tmp/evidence-native/canary-patched.json" 2>/dev/null || true)
native_terminal=$(terminal "$tmp/evidence-native/canary-go.json" "$canary")
patched_terminal=$(terminal "$tmp/evidence-native/canary-patched.json" "$canary")
test "$native_terminal" = pass && test "$native_terminal" = "$patched_terminal" || { printf 'FAIL patched go is not native-equivalent with the backend off: native=%s patched=%s\n' "$native_terminal" "$patched_terminal" >&2; exit 1; }
printf 'PASS patched cmd/go native-equivalent with the backend off (%s: %s)\n' "$canary" "$native_terminal"

run_mode() {
	mode=$1
	dir="$tmp/evidence-$mode"
	mkdir -p "$dir"
	while IFS="$tab" read -r capability pkg action want companions; do
		case "$capability" in ''|'#'*) continue ;; esac
		case_id=$(printf '%s' "$pkg" | tr '/.' '__')
		(cd "$fixture" && \
		BASHPP_GOTEST_BACKEND="$mode" BASHPP_GOTEST_TOOL="$bashpp_tool" BASHPP_GOTEST_VERSION="$bashpp_version" \
		BASHPP_GOTEST_GO="$go127" BASHPP_SHELLRT_ROOT="$shellrt" BASHPP_GOTEST_EVENTS="$dir/$case_id.events.jsonl" \
		BASHY_OTEL_SPOOL="$tmp/bashy-otel.jsonl" BASHY_HINTS=0 BASHY_NO_COACH=1 \
		GOROOT="$tmp/goroot" GOTOOLCHAIN=local GOCACHE="$tmp/gocache" GOWORK=off \
		"$tmp/go-bashpp" test -count=1 -json "$pkg" > "$dir/$case_id.go-test.json" 2> "$dir/$case_id.stderr" || true)
	done < "$matrix"
	"$tmp/package-verify" -matrix "$matrix" -evidence "$dir" -mode "$mode" -version "$bashpp_version" -tool "$bashpp_tool" > "$dir/verify.txt" || true
	cat "$dir/verify.txt"
}

# upstream_output prints the go test output lines of one package, unescaped
# enough for the expectation regexps (tabs and newlines only).
upstream_output() {
	awk -v pkg="$1" 'index($0, "\"Package\":\"" pkg "\"") && /"Action":"output"/ { s = $0; sub(/^.*"Output":"/, "", s); sub(/"(,"OutputType":"[a-z]+")?}$/, "", s); gsub(/\\t/, "\t", s); gsub(/\\n/, "\n", s); gsub(/\\"/, "\"", s); print s }' "$2"
}

fail=0
for mode in interpreted compiled; do
	printf 'Bash++ %s mode: %s (host %s)\n' "$mode" "$bashpp_version" "$host_arch"
	run_mode "$mode"
	# expect.tsv: package (relative to the module), mode, upstream terminal
	# (ERE), verifier status (ERE), an ERE the upstream output must match,
	# one it must not.
	while IFS="$tab" read -r name emode eterminal status must mustnot; do
		case "$name" in ''|'#'*) continue ;; esac
		test "$emode" = "$mode" || continue
		pkg="$module/$name"
		case_id=$(printf '%s' "$pkg" | tr '/.' '__')
		json="$tmp/evidence-$mode/$case_id.go-test.json"
		got_terminal=$(terminal "$json" "$pkg")
		got_status=$(awk -v p="$pkg" '$NF == p { print $1; exit }' "$tmp/evidence-$mode/verify.txt")
		output=$(upstream_output "$pkg" "$json")
		ok=1
		printf '%s' "$got_terminal" | grep -Eqx "$eterminal" || ok=0
		printf '%s' "$got_status" | grep -Eqx "$status" || ok=0
		if test "$must" != '-' && ! printf '%s\n' "$output" | grep -Eq "$must"; then ok=0; fi
		if test "$mustnot" != '-' && printf '%s\n' "$output" | grep -Eq "$mustnot"; then ok=0; fi
		if test "$ok" -ne 1; then
			printf 'FAIL %s %s: terminal %s (want /%s/), verifier %s (want /%s/), output must /%s/ and not /%s/\n' \
				"$name" "$mode" "${got_terminal:-none}" "$eterminal" "${got_status:-none}" "$status" "$must" "$mustnot" >&2
			printf '%s\n' "$output" | sed -n '1,40p' >&2
			sed -n '1,20p' "$tmp/evidence-$mode/$case_id.stderr" >&2 || true
			fail=1
		else
			printf 'ok   %s %s: %s %s\n' "$name" "$mode" "$got_terminal" "$got_status"
		fi
	done < "$fixture/expect.tsv"
done
if test "$fail" -ne 0; then
	printf 'FAIL S165.0 package identity\n' >&2
	exit 1
fi
printf 'PASS S165.0 package identity\n'
