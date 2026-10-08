#!/usr/bin/env bash
# Go-delta gate: every row of bashsharp/godelta/go-delta.tsv is proven by its
# fixture in tests/go-delta/ — exit status, expected stdout (<fixture>.out, when
# present) and, for refusals, the diagnostic text the row publishes.
#
#   BASHY_BIN   built bashy (default ../bashy/bin/bashy)
#   DELTA_TSV   row source (default ../bashsharp/godelta/go-delta.tsv)
#
# Island rows need a toolchain bashy can provision (network or a warm cache);
# run this on the Linux host or CI, not as a short unit test.
set -uo pipefail
HERE="$(cd "$(dirname "$0")/.." && pwd)"
BASHY_BIN="${BASHY_BIN:-$HERE/../bashy/bin/bashy}"
DELTA_TSV="${DELTA_TSV:-$HERE/../bashsharp/godelta/go-delta.tsv}"
FIX="$HERE/tests/go-delta"
[ -x "$BASHY_BIN" ] || { echo "go-delta-gate: no bashy at $BASHY_BIN" >&2; exit 2; }
[ -r "$DELTA_TSV" ] || { echo "go-delta-gate: no row table at $DELTA_TSV" >&2; exit 2; }

pass=0; fail=0
while IFS=$'\037' read -r id surface spec production status reason workaround diagnostic fixture exit_want; do
  [ "$id" = id ] && continue
  work="$(mktemp -d)"
  cp "$FIX"/* "$work"/
  [ "$id" = I07 ] || printf 'module godelta\n\ngo 1.24\n' > "$work/go.mod"
  case "$fixture" in
    *.bsh) cmd=("$BASHY_BIN" --bashsharp "$fixture") ;;
    *)     cmd=("$BASHY_BIN" "$fixture") ;;
  esac
  (cd "$work" && "${cmd[@]}" >"$work/.stdout" 2>"$work/.stderr" </dev/null)
  got=$?
  bad=""
  [ "$got" = "$exit_want" ] || bad="$bad exit=$got want $exit_want;"
  if [ -f "$FIX/$id.out" ] && ! cmp -s "$work/.stdout" "$FIX/$id.out"; then
    bad="$bad stdout differs;"
  fi
  if [ -n "$diagnostic" ] && ! grep -qF -- "$diagnostic" "$work/.stdout" "$work/.stderr"; then
    bad="$bad diagnostic \"$diagnostic\" not emitted;"
  fi
  if [ -n "$bad" ]; then
    fail=$((fail+1)); echo "FAIL $id ($fixture):$bad"
    [ -n "${GO_DELTA_VERBOSE:-}" ] && { sed 's/^/  out: /' "$work/.stdout"; sed 's/^/  err: /' "$work/.stderr"; }
  else
    pass=$((pass+1)); echo "PASS $id"
  fi
  rm -rf "$work"
done < <(tr '\t' '\037' < "$DELTA_TSV")
echo "go-delta: $pass passed, $fail failed"
[ "$fail" = 0 ]
