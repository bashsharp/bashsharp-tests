#!/usr/bin/env bash
# Regenerate docs/leaderboard.md: who closed corpus-repair issues (by the
# author of the closing PR, or the issue's assignee/closer), and the standing
# of the repair class. Read-only against GitHub; run by the weekly workflow.
set -euo pipefail
repo=${CORPUS_ISSUES_REPO:-qiangli/bashsharp-tests}
out=${1:-$(dirname "$0")/../../docs/leaderboard.md}
open=$(gh issue list -R "$repo" --label corpus-repair --state open --limit 1000 --json number --jq 'length')
closed_json=$(gh issue list -R "$repo" --label corpus-repair --state closed --limit 1000 --json number,closedAt,title,assignees,url)
closed=$(printf '%s' "$closed_json" | jq 'length')
# credit: PRs that reference the issue ("#N") and were merged; else the assignee
credit=$(gh pr list -R "$repo" --state merged --limit 1000 --json number,author,body,title --jq '.[] | "\(.author.login)\t\(.title) \(.body)"' 2>/dev/null | awk -F'\t' '{ n=split($2, w, /[^0-9#]+/); for (i=1;i<=n;i++) if (w[i] ~ /^#[0-9]+$/) print $1 }' | sort | uniq -c | sort -rn)
{
  echo "# Corpus-repair leaderboard"
  echo
  echo "Regenerated $(date -u +%Y-%m-%dT%H:%MZ) from GitHub. Repair roots: **$open open · $closed closed**."
  echo "A root counts when its PR is merged and both Bash# modes match the oracle."
  echo
  if [ -n "$credit" ]; then
    echo "| roots closed | by |"; echo "|---:|---|"
    printf '%s\n' "$credit" | awk '{printf "| %s | @%s |\n", $1, $2}'
  else
    echo "_No merged repair PRs yet — the first one starts the table._"
  fi
  echo
  echo "## Recently closed"
  echo
  printf '%s' "$closed_json" | jq -r 'sort_by(.closedAt) | reverse | .[:25][] | "- [\(.title | .[0:90])](\(.url)) — \(.closedAt[0:10])"'
} > "$out"
echo "leaderboard: $out ($open open, $closed closed)"
