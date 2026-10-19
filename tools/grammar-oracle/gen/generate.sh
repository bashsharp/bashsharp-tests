#!/bin/sh
# Regenerate the grammar-oracle Go recognizer (Sprint 384, #1586).
#
#   tools/grammar-oracle/gen/generate.sh           install into grammar/go-target/ and goparser/
#   tools/grammar-oracle/gen/generate.sh --check   regenerate into a temp dir; diff -r against the committed files
#
# Run from the development machine. ANTLR (Java) runs only on a claimed remote
# macOS or Linux host with a JDK path through the java runner, named by
# GRAMMAR_ORACLE_HOST (required; passwordless ssh, `zsh -lc` login environment
# with bashy and python3). The remote side:
#   1. fetches antlr-4.13.2-complete.jar into $BASHY_HOME/custom/antlr/4.13.2/
#      (once) and refuses it unless its SHA-256 equals docs/grammar-oracle/pin.tsv;
#   2. applies the Go-target transform by running the pinned upstream
#      grammar/upstream/Go/transformGrammar.py unchanged (python3), then every
#      grammar/go-target/patches/*.patch in name order (patch -p1);
#   3. registers gen/java-runner.sh in a private BASHY_COMMANDS_DIR ring and
#      runs gen/generate.bsh, whose ~~~java fence calls org.antlr.v4.Tool.
# The transformed grammars and the generated .go files come back; the remote
# temp dir is always deleted. goparser/generated.sha256 lists every output,
# relative to tools/grammar-oracle (shasum -c works there). go_parser_base.go is the upstream base with its
# package clause renamed (parser -> goparser), nothing else.
set -eu

here=$(CDPATH= cd -- "$(dirname "$0")" && pwd -P)
root=${here%/gen}
pin="$root/../../docs/grammar-oracle/pin.tsv"
mode=${1:-install}
case "$mode" in install|--check) ;; *) echo "usage: generate.sh [--check]" >&2; exit 2 ;; esac

die() { echo "generate.sh: $*" >&2; exit 1; }

host=${GRAMMAR_ORACLE_HOST:-}
[ -n "$host" ] || die "GRAMMAR_ORACLE_HOST is not set: export the ssh name of a claimed remote macOS or Linux host with a JDK path through the java runner (it is never written into a file)"

jar_url=$(awk -F '\t' '$1 == "antlr-tool" { print $3 }' "$pin")
jar_sha=$(awk -F '\t' '$1 == "antlr-tool" { print $5 }' "$pin")
case "$jar_sha" in [0-9a-f]*) [ ${#jar_sha} -eq 64 ] || die "bad antlr-tool sha256 in pin.tsv" ;; *) die "no antlr-tool row in pin.tsv" ;; esac

lt=$(mktemp -d "${TMPDIR:-/tmp}/grammar-oracle-gen.XXXXXX")
rd=""
cleanup() {
	case "$rd" in */grammar-oracle-gen.*) ssh -o BatchMode=yes "$host" "rm -rf '$rd'" || echo "generate.sh: could not delete $host:$rd" >&2 ;; esac
	rm -rf "$lt"
}
trap cleanup EXIT INT TERM

# Stage the inputs.
mkdir -p "$lt/stage/parser" "$lt/stage/patches"
cp "$root/grammar/upstream/GoLexer.g4" "$root/grammar/upstream/GoParser.g4" "$lt/stage/parser/"
cp "$root/grammar/upstream/Go/transformGrammar.py" "$here/generate.bsh" "$here/java-runner.sh" "$here/java-pin.tsv" "$lt/stage/"
for p in "$root"/grammar/go-target/patches/*.patch; do [ -e "$p" ] && cp "$p" "$lt/stage/patches/"; done
cat >"$lt/stage/run.sh" <<'EOF'
set -eu
d=$1 url=$2 want=$3
echo "os: $(uname -sm); $(bashy --version | head -n 1); $(python3 --version)" >&2
jar_dir=${BASHY_HOME:-$HOME/.bashy}/custom/antlr/4.13.2
jar=$jar_dir/antlr-4.13.2-complete.jar
if [ ! -f "$jar" ]; then
	mkdir -p "$jar_dir"
	curl -sSfL "$url" -o "$jar.part"
	mv "$jar.part" "$jar"
fi
got=$(shasum -a 256 "$jar" | awk '{ print $1 }')
[ "$got" = "$want" ] || { echo "run.sh: $jar sha256 $got != pinned $want" >&2; exit 1; }
echo "jar: antlr-4.13.2-complete.jar sha256 $got (pinned)" >&2
cd "$d"
python3 transformGrammar.py >&2
rm -f parser/*.bak
for p in patches/*.patch; do [ -e "$p" ] || continue; echo "patch: $p" >&2; patch -s -p1 -d parser <"$p"; done
export BASHY_COMMANDS_DIR="$d/ring"
mkdir -p "$BASHY_COMMANDS_DIR" out
bashy commands register java-runner --set exec.0="$d/java-runner.sh" --set effects.0=exec >/dev/null
cd parser
CLASSPATH=$jar JAVA_RUNNER_CLASSPATH=$jar bashy --bashsharp "$d/generate.bsh" "$d/out"
EOF

rd=$(ssh -o BatchMode=yes "$host" 'mktemp -d "${TMPDIR:-/tmp}/grammar-oracle-gen.XXXXXX"')
case "$rd" in */grammar-oracle-gen.*) ;; *) die "bad remote temp dir: $rd" ;; esac
tar -C "$lt/stage" -cf - . | ssh -o BatchMode=yes "$host" "tar -C '$rd' -xf -"
ssh -o BatchMode=yes "$host" "zsh -lc 'sh $rd/run.sh $rd $jar_url $jar_sha'" || die "remote generation failed on $host"
mkdir -p "$lt/back"
ssh -o BatchMode=yes "$host" "tar -C '$rd' -cf - parser/GoLexer.g4 parser/GoParser.g4 out/go_lexer.go out/go_parser.go" | tar -C "$lt/back" -xf -

# Assemble the result exactly as it is committed.
res="$lt/result"
mkdir -p "$res/grammar/go-target" "$res/goparser"
cp "$lt/back/parser/GoLexer.g4" "$lt/back/parser/GoParser.g4" "$res/grammar/go-target/"
cp "$lt/back/out/go_lexer.go" "$lt/back/out/go_parser.go" "$res/goparser/"
sed 's/^package parser$/package goparser/' "$root/grammar/upstream/Go/go_parser_base.go" >"$res/goparser/go_parser_base.go"
(cd "$res" && shasum -a 256 grammar/go-target/GoLexer.g4 grammar/go-target/GoParser.g4 goparser/go_lexer.go goparser/go_parser.go goparser/go_parser_base.go) >"$res/goparser/generated.sha256"
cat "$res/goparser/generated.sha256"

if [ "$mode" = --check ]; then
	diff -r "$res/goparser" "$root/goparser" || die "regenerated goparser/ differs from the committed files"
	for f in GoLexer.g4 GoParser.g4; do
		cmp "$res/grammar/go-target/$f" "$root/grammar/go-target/$f" || die "regenerated go-target/$f differs"
	done
	echo "generate.sh --check: byte-identical to the committed output"
else
	cp "$res/grammar/go-target/GoLexer.g4" "$res/grammar/go-target/GoParser.g4" "$root/grammar/go-target/"
	rm -f "$root"/goparser/*
	cp "$res"/goparser/* "$root/goparser/"
	echo "generate.sh: installed grammar/go-target/ and goparser/"
fi
