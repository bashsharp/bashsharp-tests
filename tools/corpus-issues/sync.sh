#!/usr/bin/env bash
# Open one GitHub issue per REPAIRABLE Go-corpus root, from the language repo's
# target catalog (bashsharp/docs/go-corpus-targets.tsv, class = repair), and
# close the issue of a root that has left the repair class. Idempotent: the
# issue title carries the root id, and an existing open issue is left alone.
#
#   tools/corpus-issues/sync.sh [--limit N] [--dry-run] [--catalog PATH]
#
# Needs `gh` authenticated with write access to the issues repo. Creation is
# paced (GitHub's content-creation secondary limit); --limit bounds one run.
set -euo pipefail
repo=${CORPUS_ISSUES_REPO:-qiangli/bashsharp-tests}
catalog=${CATALOG:-$(dirname "$0")/../../../bashsharp/docs/go-corpus-targets.tsv}
limit=1000; dry=0
while [ $# -gt 0 ]; do
  case "$1" in
    --limit) limit=$2; shift 2 ;;
    --dry-run) dry=1; shift ;;
    --catalog) catalog=$2; shift 2 ;;
    *) echo "usage: $0 [--limit N] [--dry-run] [--catalog PATH]" >&2; exit 2 ;;
  esac
done
[ -f "$catalog" ] || { echo "sync: catalog not found: $catalog" >&2; exit 2; }

area_of() { # internal family -> public area label
  case "$1" in
    151) echo evaluator ;; 152) echo lowering ;; 153) echo runtime ;; 154) echo diagnostics ;;
    package) echo packages ;; *) echo triage ;;
  esac
}
area_hint() {
  case "$1" in
    evaluator) echo 'the interpreted evaluator (`sh/interp`, the `bashpp_*` files): expressions, values, pointers, method sets' ;;
    lowering) echo 'the lowering compiler (`sh/lower`): the emitted Go differs from the interpreted result, or fails to build' ;;
    runtime) echo 'the runtime bridge (`sh/interp` + `sh/lower/shellrt`): goroutines, channels, callbacks, deadlines, output ordering' ;;
    diagnostics) echo 'diagnostics: the program is rejected or reported differently from gc' ;;
    packages) echo 'package roots: multi-file / multi-package programs' ;;
    *) echo 'not yet classified — the first job is to say which of the above it is' ;;
  esac
}

# existing open corpus-repair issues, keyed by root id in the title
declare -A open_issue
while IFS=$'\t' read -r num title; do
  root=$(printf '%s' "$title" | sed -n 's/^corpus-repair: \([^ ]*\) .*/\1/p')
  [ -n "$root" ] && open_issue["$root"]=$num
done < <(gh issue list -R "$repo" --label corpus-repair --state open --limit 1000 --json number,title --jq '.[] | "\(.number)\t\(.title)"')

created=0; skipped=0; closed=0
declare -A in_repair
# Rows are re-separated with US (0x1f): bash's `read` treats consecutive TABs
# as one (IFS whitespace collapsing), which would shift every column after an
# empty `reason`.
while IFS=$'\x1f' read -r root runner action mode class family reason destination first_line; do
  [ "$class" = repair ] || continue
  [ -n "${in_repair[$root]:-}" ] && continue   # one issue per root, whatever the modes
  in_repair["$root"]=1
  if [ -n "${open_issue[$root]:-}" ]; then skipped=$((skipped+1)); continue; fi
  [ "$created" -lt "$limit" ] || continue
  area=$(area_of "$family")
  path=${root#*:}
  first=$(printf '%s' "$first_line" | sed 's/^"//; s/"$//; s/""/"/g' | cut -c1-140)
  title="corpus-repair: $root — ${first:-no first line recorded}"
  body=$(cat <<EOB
**Root:** \`$root\` (runner \`$runner\`, action \`$action\`, mode \`$mode\`)
**Area:** \`$area\` — $(area_hint "$area")
**First line of the failure:**
\`\`\`
${first_line:-"(none recorded)"}
\`\`\`

This is one of the *repair* roots in the Go-corpus target catalog — a program the upstream Go 1.27.1 oracle passes and Bash# does not, with a first cause. Whole-corpus figures and the four classes: [docs/claims.md](https://github.com/qiangli/bashsharp/blob/main/docs/claims.md) · [go-corpus-targets.md](https://github.com/qiangli/bashsharp/blob/main/docs/go-corpus-targets.md).

**Reproduce**
1. \`tools/go-corpus/refresh.sh\` once (fetches and verifies the pinned Go 1.27.1 source; needs Go ≥ 1.27 on PATH).
2. Oracle: \`go run go/test/$path\` (or the action the upstream runner uses for \`$action\`).
3. Bash#: \`bashsharp --source=go go/test/$path\` (interpreted) and, for the compiled mode, \`bashsharp transpile --source=go go/test/$path -o /tmp/t.go && go run /tmp/t.go\` inside a module that replaces \`mvdan.cc/sh/v3\` with a \`github.com/qiangli/sh\` checkout.
4. Diff the two. The fix is done when **both** Bash# modes match the oracle and the classic gate (\`tools/classic-gate.sh\`) is unchanged.

**Where the fix lives:** \`github.com/qiangli/sh\` (\`interp/\` for interpreted, \`lower/\` for compiled); un-\`planned\` or add the fixture here; one PR per root, citing this issue.
EOB
)
  if [ "$dry" = 1 ]; then echo "would create: $title"; created=$((created+1)); continue; fi
  if ! url=$(gh issue create -R "$repo" --title "$title" --body "$body" --label corpus-repair --label "good first issue" --label "area:$area" 2>&1); then
    case "$url" in
      *"secondary rate limit"*) echo "sync: GitHub's content-creation limit — stopping; re-run later (idempotent). created=$created" >&2; exit 3 ;;
      *) echo "sync: create failed for $root: $url" >&2; exit 1 ;;
    esac
  fi
  echo "created: $(printf '%s' "$url" | tail -1)"
  created=$((created+1)); sleep 6      # GitHub's content-creation secondary limit bites at ~1 issue every few seconds
done < <(tail -n +2 "$catalog" | awk -F'\t' 'BEGIN{OFS="\x1f"}{$1=$1; print}')

# roots that left the repair class: close their issues
for root in "${!open_issue[@]}"; do
  [ -n "${in_repair[$root]:-}" ] && continue
  if [ "$dry" = 1 ]; then echo "would close: #${open_issue[$root]} ($root)"; continue; fi
  gh issue close -R "$repo" "${open_issue[$root]}" -c "This root is no longer in the repair class of the target catalog." >/dev/null && closed=$((closed+1)) && sleep 1
done
echo "sync: created=$created skipped(open)=$skipped closed=$closed repo=$repo"
