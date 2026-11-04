#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/../../.." && pwd)"
pin="$root/python-by-example/pin.tsv"
out="$root/python-by-example/inventory.tsv"
cache="${PYTHON_CORPUS_CACHE:-$root/.cache/python-fence-corpus/cpython}"

IFS=$'\t' read -r source repo commit code_license prose_license < <(sed -n '2p' "$pin")
[ "$source" = v3.15.0 ]
[ "$commit" = d220409a13b77de7a6c30abe16ea90db01528edc ]
[ "$code_license" = '0BSD OR PSF-2.0' ]
[ "$prose_license" = PSF-2.0 ]

if [ ! -d "$cache/.git" ]; then
  mkdir -p "$(dirname "$cache")"
  git clone --filter=blob:none --no-checkout "$repo" "$cache"
fi
git -C "$cache" fetch --quiet --depth=1 origin "$commit"
git -C "$cache" checkout --quiet --detach "$commit"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
printf 'source_ref\tdisposition\tcase_id\treason\n' >"$tmp"

awk '
function emit(    ref,disp,id,reason,key) {
  if (!active || first == 0) return
  ref = "Doc/tutorial/" file ":" first "-" last
  key = file ":" first
  id = "-"
  if (key == "introduction.rst:58") id = "tutorial-introduction"
  else if (key == "controlflow.rst:63") id = "tutorial-controlflow"
  else if (key == "datastructures.rst:100") id = "tutorial-datastructures"
  else if (key == "modules.rst:28") id = "tutorial-modules"
  else if (key == "inputoutput.rst:32") id = "tutorial-inputoutput"
  else if (key == "errors.rst:95") id = "tutorial-errors"
  else if (key == "classes.rst:168") id = "tutorial-classes"
  else if (key == "stdlib.rst:164") id = "tutorial-stdlib"
  else if (key == "stdlib2.rst:19") id = "tutorial-stdlib2"
  else if (key == "floatingpoint.rst:98") id = "tutorial-floatingpoint"

  if (id != "-") { disp = "case"; reason = "deterministic native-oracle adaptation" }
  else if (body ~ /(^|\n)[ ]*>>> /) { disp = "interactive"; reason = "upstream REPL transcript retained in inventory" }
  else if (lang ~ /(none|text|console|shell|bash)/) { disp = "diagnostic"; reason = "non-Python or diagnostic transcript" }
  else if (body ~ /(^|\n)[ ]*class[ ]/) { disp = "missing-feature:python-class-declarations"; reason = "top-level fence declarations accept functions, imports, docstrings, and constants" }
  else { disp = "fragment"; reason = "non-interactive tutorial fragment lacks a standalone oracle entry point" }
  print ref "\t" disp "\t" id "\t" reason
  active = 0; first = last = 0; body = ""; lang = ""
}
function indent(s,    n) { n=match(s,/[^ ]/); return n ? n-1 : length(s) }
FNR == 1 { if (NR > 1) emit(); file=FILENAME; sub(/^.*\//,"",file); pending=0; active=0 }
{
  line=$0
  if (active) {
    if (line ~ /^[ ]*$/) { body=body "\n" line; last=FNR; next }
    if (indent(line) < base) { emit() }
    else { body=body "\n" line; last=FNR; next }
  }
  if (pending) {
    if (line ~ /^[ ]*$/ || line ~ /^[ ]+:[A-Za-z-]+:/) next
    if (line ~ /^[ ]+/) { active=1; base=indent(line); first=last=FNR; body=line; pending=0; next }
    pending=0
  }
  if (line ~ /^\.\. (code-block|doctest|testcode)::/) {
    pending=1; lang=line; sub(/^.*::[ ]*/,"",lang); next
  }
  if (line ~ /::$/ && line !~ /^\.\. /) { pending=1; lang="python"; next }
}
END { emit() }
' "$cache"/Doc/tutorial/*.rst >>"$tmp"

mv "$tmp" "$out"
trap - EXIT
printf 'inventory: %s examples from %s@%s\n' "$(( $(wc -l <"$out") - 1 ))" "$source" "$commit"
