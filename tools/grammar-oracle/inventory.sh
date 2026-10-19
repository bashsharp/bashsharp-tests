#!/bin/sh
# Generate the grammar-oracle fixture manifest from the EXISTING inventories.
#
#   tools/grammar-oracle/inventory.sh > docs/grammar-oracle/manifest.tsv
#
# Deterministic: the output depends only on the bytes of the sources below.
# Every entry of every source inventory becomes exactly one row; an entry the
# oracle does not compare is still a row, class not-applicable, with a reason.
# A missing source file, a hash mismatch, an unknown inventory value or a
# duplicate fixture_id is an error (exit 1). The existing validators own their
# inventories' own checks and are run first, fail-closed.
#
# Sibling repositories are read-only. They resolve from SH_ROOT and
# BASHSHARP_ROOT, or else from DHNT_ROOT (the directory holding the sh and
# bashsharp checkouts: $DHNT_ROOT/sh, $DHNT_ROOT/bashsharp). There is no
# default; a missing root is an error. Rows name them only repo-qualified
# (sh:, bashsharp:, tests:), never by absolute path. Columns and classes: docs/grammar-oracle/CONTRACT.md.
set -eu
LC_ALL=C
export LC_ALL

here=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$here/../.." && pwd)
S6_SHA256=1cbc58250667661eeac6d5f99c94c8b5afb9d34f3b1c3026e03d30ecc16a3f71

die() { echo "inventory.sh: $*" >&2; exit 1; }
need() { [ -f "$1" ] || die "missing source: $1"; }
root_of() { # VAR value repo
	if [ -n "$2" ]; then r=$2
	elif [ -n "${DHNT_ROOT:-}" ]; then r=$DHNT_ROOT/$3
	else die "set $1 (the $3 checkout) or DHNT_ROOT (the directory holding sh and bashsharp)"
	fi
	[ -d "$r" ] || die "missing sibling repository $3: $r ($1 / DHNT_ROOT)"
	(cd "$r" && pwd)
}
SH=$(root_of SH_ROOT "${SH_ROOT:-}" sh) || exit 1
BS=$(root_of BASHSHARP_ROOT "${BASHSHARP_ROOT:-}" bashsharp) || exit 1

if command -v sha256sum >/dev/null 2>&1; then
	sha() { sha256sum <"$1" | cut -c1-64; }
else
	sha() { shasum -a 256 <"$1" | cut -c1-64; }
fi

US=$(printf '\037')
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# rows: TSV data rows with empty fields kept (tabs become \037 for `read`).
rows() { need "$1"; grep -v '^#' "$1" | grep -v '^[[:space:]]*$' | tr '\t' "$US" || true; }

# row fixture_id lane source sha256 dialect policy class expected boundary reason
row() { printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$@"; }

# Decode the printf %b subset the start-site corpus uses (\n \t \\); any other
# %b escape is refused so the bytes stay identical to classify.sh's probes.
pct_b() {
	awk 'BEGIN { s = ARGV[1]; ARGV[1] = ""; out = ""
		while ((i = index(s, "\\")) > 0) {
			c = substr(s, i + 1, 1)
			if (c ~ /^[abcefrvx0-7uU]$/) { print "unsupported %b escape \\" c > "/dev/stderr"; exit 1 }
			out = out substr(s, 1, i - 1)
			if (c == "n") out = out "\n"; else if (c == "t") out = out "\t"
			else if (c == "\\") out = out "\\"; else out = out "\\" c
			s = substr(s, i + 2)
		}
		printf "%s\n", out s }' "$1"
}

validate() {
	for v in tools/tour/validate.sh tools/tour/validate-corpus.sh tools/go-by-example/validate.sh; do
		[ -x "$ROOT/$v" ] || die "missing validator: $v"
		"$ROOT/$v" >&2 || die "$v failed"
	done
}

tour() {
	rows "$ROOT/tests/tour/inventory.tsv" | while IFS=$US read -r path kind _start appl _exc _diff _bytes hash; do
		id="tour/${path#_content/tour/}"
		case $appl in
		applicable_go_program | build_only_go_program)
			f="$ROOT/tour/$path"
			need "$f"
			[ "$(sha "$f")" = "$hash" ] || die "tour sha256 mismatch: $path"
			row "$id" unit "tests:tour/$path" "$hash" go-unit gcsyntax required-positive accept go-unit \
				"tour $kind $appl: package-led unit, Go-source frontend" ;;
		excluded_fragment)
			row "$id" unit "tests:tests/tour/inventory.tsv#$path" - - - not-applicable - - \
				"tour $kind excluded_fragment: bytes not committed (inventory pins upstream sha256 $hash); not a complete unit" ;;
		*) die "unknown tour applicability: $appl ($path)" ;;
		esac
	done
	rows "$ROOT/docs/tour/pin.tsv" | while IFS=$US read -r repo version commit _rest; do
		row "pin/tour" unit "tests:docs/tour/pin.tsv#$repo" - - - not-applicable - - \
			"provenance pin $repo@$version ($commit), checked by tools/tour/validate.sh"
	done
}

gbe() {
	rows "$ROOT/docs/go-by-example/inventory.tsv" | while IFS=$US read -r path kind _beh _norm _adapter _req _bytes hash; do
		id="gbe/${path#examples/}"
		f="$ROOT/$path"
		need "$f"
		[ "$(sha "$f")" = "$hash" ] || die "go-by-example sha256 mismatch: $path"
		case $kind in
		program | test_program)
			row "$id" unit "tests:$path" "$hash" go-unit gcsyntax required-positive accept go-unit \
				"go-by-example $kind: package-led unit, Go-source frontend" ;;
		provenance | runtime_asset)
			row "$id" unit "tests:$path" "$hash" - - not-applicable - - \
				"go-by-example $kind: not a program (classification.tsv not_a_program)" ;;
		*) die "unknown go-by-example kind: $kind ($path)" ;;
		esac
	done
	rows "$ROOT/docs/go-by-example/pin.tsv" | while IFS=$US read -r repo commit _rest; do
		row "pin/go-by-example" unit "tests:docs/go-by-example/pin.tsv#$repo" - - - not-applicable - - \
			"provenance pin $repo@$commit; classification.tsv and inventory.tsv are joined by tools/go-by-example/validate.sh"
	done
}

bpp_manifest() {
	rows "$ROOT/tests/manifest.tsv" | while IFS=$US read -r path status feature phase _origin; do
		f="$ROOT/$path"
		need "$f"
		case $path in *.sh) dialect=bash ;; *) dialect=bashsharp ;; esac
		case $status in
		supported) class=required-positive expected=accept ;;
		planned) class=planned expected=- ;;
		n/a) class=not-applicable expected=- ;;
		*) die "unknown tests/manifest.tsv status: $status ($path)" ;;
		esac
		row "bpp/${path#tests/}" mixed "tests:$path" "$(sha "$f")" "$dialect" sh-syntax "$class" "$expected" - \
			"tests/manifest.tsv $status ($feature $phase); status declared there, not reclassified"
	done
}

s6() {
	tsv="$BS/godelta/go-delta.tsv"
	need "$tsv"
	got=$(sha "$tsv")
	[ "$got" = "$S6_SHA256" ] || die "S6 table sha256 $got, frozen $S6_SHA256: STOP"
	rows "$tsv" | while IFS=$US read -r id surface _spec _prod status _reason _work diag fixture ex; do
		[ "$id" = id ] && continue
		f="$ROOT/tests/go-delta/$fixture"
		need "$f"
		case $fixture in
		*.go) dialect=go-unit policy=gcsyntax ;;
		*) if [ "$surface" = unit ]; then dialect=go-unit policy=gcsyntax; else dialect=bashsharp policy=sh-syntax; fi ;;
		esac
		case $id in
		M09 | M10 | M11)
			class=planned expected=reject
			why="S6 gap, refused by sh/syntax today ($diag); failure phase measured at the syntax seam in checkpoint 2; a gap is not an exclusion" ;;
		M15 | M16)
			class=required-negative expected=reject
			why="S6 excluded: refused by sh/syntax at the mixed start site ($diag)" ;;
		*)
			class=required-positive expected=accept
			why="S6 $status, fixture exit $ex"
			[ "$ex" = 0 ] || why="$why: the refusal is not syntax ($diag), reported separately" ;;
		esac
		row "s6/$id" "$surface" "tests:tests/go-delta/$fixture" "$(sha "$f")" "$dialect" "$policy" "$class" "$expected" - "$why"
	done
	# Support files beside the row fixtures; *.out are expected stdout, not inputs.
	for f in "$ROOT"/tests/go-delta/*; do
		b=${f##*/}
		case $b in
		*.out) continue ;;
		I03-main.go) why="embed target of s6/I03; parsed as that row's Go payload" ;;
		I07.nomod) why="empty marker: s6/I07 runs without a go.mod" ;;
		*) tr '\t' "$US" <"$tsv" | cut -d"$US" -f9 | grep -qx "$b" && continue
			die "unaccounted tests/go-delta file: $b" ;;
		esac
		row "s6-support/$b" island "tests:tests/go-delta/$b" "$(sha "$f")" - - not-applicable - - "$why"
	done
}

polyglot() {
	src="$ROOT/tools/polyglot-gate.sh"
	need "$src"
	# Each `cat >"$scratch/NAME.bpp" <<'MARK'` heredoc, as the bytes bashy runs.
	awk -v dir="$tmp" '
		!body && /^cat[ \t]+>.*\.bpp["'\'']?[ \t]+<<'\''[A-Za-z_][A-Za-z_0-9]*'\''[ \t]*$/ {
			name = $0; sub(/^cat[ \t]+>"?\$scratch\//, "", name); sub(/\.bpp.*$/, ".bpp", name)
			mark = $0; sub(/^.*<<'\''/, "", mark); sub(/'\''.*$/, "", mark)
			file = dir "/poly." NR; printf "" > file; print NR "\t" name; body = 1; next
		}
		body && $0 == mark { close(file); body = 0; next }
		body { print > file }
		END { if (body) exit 1 }' "$src" >"$tmp/poly.idx" || die "unterminated heredoc in tools/polyglot-gate.sh"
	[ -s "$tmp/poly.idx" ] || die "no .bpp heredocs in tools/polyglot-gate.sh"
	while IFS=$(printf '\t') read -r line name; do
		row "polyglot/$name" island "tests:tools/polyglot-gate.sh#L$line" "$(sha "$tmp/poly.$line")" bashsharp sh-syntax \
			required-positive accept fence "polyglot-gate.sh heredoc; engine-accepted (sh grammar TestAgreementWithPublishedCorpora)"
	done <"$tmp/poly.idx"
}

delta_testdata() {
	for want in accept reject; do
		d="$SH/grammar/testdata/$want"
		[ -d "$d" ] || die "missing source: $d"
		for f in "$d"/*; do
			b=${f##*/}
			src="sh:grammar/testdata/$want/$b"
			case $b in *.bsh) ;; *)
				row "delta/$want/$b" island "$src" "$(sha "$f")" - - not-applicable - - "embed target file, not a script"
				continue ;;
			esac
			case $b in embed-* | fence-* | unclosed-fence*) lane=island ;; *) lane=mixed ;; esac
			case $b in
			at-function.bsh) kind=at-function ;;
			typed-method.bsh) kind=typed-method ;;
			embed-alias.bsh) kind=embed-line ;;
			*) kind=- ;;
			esac
			if [ "$b" = embed-missing.bsh ]; then
				row "delta/$want/$b" "$lane" "$src" "$(sha "$f")" bashsharp sh-syntax contextual reject embed-line \
					"missing embed target: grammatically an embed line; the reference rejects in the frozen directory (filesystem context)"
			elif [ "$want" = accept ]; then
				row "delta/$want/$b" "$lane" "$src" "$(sha "$f")" bashsharp sh-syntax required-positive accept "$kind" \
					"sh grammar TestFixtureAgreement accept fixture"
			else
				row "delta/$want/$b" "$lane" "$src" "$(sha "$f")" bashsharp sh-syntax required-negative reject - \
					"sh grammar TestFixtureAgreement reject fixture"
			fi
		done
	done
}

divergences() {
	src="$SH/grammar/grammar_test.go"
	need "$src"
	# Case rows {"<go literal>", engine, grammar} of TestKnownDivergences; the
	# engine (sh/syntax) verdict is the authored expectation.
	awk '/^func TestKnownDivergences\(/ { on = 1; next }
		on && /^}/ { exit }
		on && /^[ \t]*\{".*", (true|false), (true|false)\},[ \t]*$/ {
			s = $0; sub(/^[ \t]*\{"/, "", s); e = s; sub(/", (true|false), (true|false)\},[ \t]*$/, "", s)
			sub(/^.*", /, "", e); sub(/,.*$/, "", e); print e "\t" s }' "$src" >"$tmp/div.idx"
	[ "$(wc -l <"$tmp/div.idx")" -eq 4 ] || die "TestKnownDivergences: want 4 cases, got $(wc -l <"$tmp/div.idx")"
	n=0
	while IFS=$(printf '\t') read -r engine lit; do
		case $lit in *\\[!n]*) die "TestKnownDivergences: unsupported escape in $lit" ;; esac
		awk 'BEGIN { s = ARGV[1]; ARGV[1] = ""; gsub(/\\n/, "\n", s); printf "%s", s }' "$lit" >"$tmp/div.$n"
		if [ "$engine" = true ]; then
			class=required-positive expected=accept kind=shell
		else
			class=required-negative expected=reject kind=-
		fi
		row "divergence/$n" mixed "sh:grammar/grammar_test.go#TestKnownDivergences/$n" "$(sha "$tmp/div.$n")" bashsharp sh-syntax \
			"$class" "$expected" "$kind" "known coarse-fragment divergence; engine verdict is the expectation"
		n=$((n + 1))
	done <"$tmp/div.idx"
}

startsites() {
	d="$ROOT/tools/startsites"
	need "$d/baseline.tsv"
	rows "$d/posix-known-divergent.tsv" >"$tmp/posix"
	rows "$d/baseline.tsv" >"$tmp/base"
	cut -d"$US" -f1 "$tmp/base" | sort >"$tmp/base.ids"
	rows "$d/shapes.tsv" | cut -d"$US" -f1 | sort >"$tmp/shape.ids"
	cmp -s "$tmp/base.ids" "$tmp/shape.ids" || die "tools/startsites: shapes.tsv and baseline.tsv id sets differ"
	rows "$d/shapes.tsv" | while IFS=$US read -r id phase feature shape probe; do
		class=$(awk -F"$US" -v id="$id" '$1 == id { print $4 }' "$tmp/base")
		pct_b "${probe:-$shape}" >"$tmp/ss" || die "startsites $id: bad probe"
		case $class in
		E) c=required-positive expected=accept kind=shell ;;
		R) c=required-negative expected=reject kind=- ;;
		*) die "startsites $id: baseline class $class" ;;
		esac
		why="start site $phase $feature, class $class under GNU Bash 5.3 -n (baseline.tsv)"
		owner=$(awk -F"$US" -v id="$id" '$1 == id { print $2 }' "$tmp/posix")
		[ -z "$owner" ] || why="$why; posix-known-divergent, owner $owner"
		row "startsite/$id" mixed "tests:tools/startsites/shapes.tsv#$id" "$(sha "$tmp/ss")" bash sh-syntax "$c" "$expected" "$kind" "$why"
	done
	# A posix-known-divergent row must name a shape (the table is empty today).
	cut -d"$US" -f1 "$tmp/posix" | while read -r id; do
		grep -qx "$id" "$tmp/shape.ids" || die "posix-known-divergent id $id is not a shape"
	done
}

main() {
	printf '# Grammar-oracle fixture manifest (Sprint 384, story #1586). GENERATED by\n'
	printf '# tools/grammar-oracle/inventory.sh; do not hand-edit. Schema: docs/grammar-oracle/CONTRACT.md\n'
	printf '# S6 bashsharp:godelta/go-delta.tsv sha256 %s\n' "$S6_SHA256"
	printf '# fixture_id\tlane\tsource\tsha256\tdialect\tpolicy\tclass\texpected\tboundary\treason\n'
	tour
	gbe
	bpp_manifest
	s6
	polyglot
	delta_testdata
	divergences
	startsites
}

validate
main >"$tmp/manifest"
dups=$(grep -v '^#' "$tmp/manifest" | cut -f1 | sort | uniq -d)
[ -z "$dups" ] || die "duplicate fixture_id: $dups"
cat "$tmp/manifest"
