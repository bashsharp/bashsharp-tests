---
id: cebf026837c7
kind: bug
title: S557 repair literal-byte transport test accounting prerequisite
seq: 112
status: wontfix
priority: p1
labels:
    - superseded
created: 2026-11-01T19:27:41.345884Z
weave: 1
assignee: s557-public-tooling
sprint: 557
sprint_id: f5ccf563-f5d1-5b94-8be4-9388e51ef86d
sprint_title: 'Bashy v1.0.0 preparation: split private/public development and retire qiangli/dhnt'
closed: 2026-11-02T05:32:08.638886Z
---

Sprint557 migration repair. Work only in assigned isolated workspace. Read local CLAUDE/AGENTS. Preserve all test assertions, coverage, thresholds and deadlines; diagnose before modifying. No release/push/parent pin changes. No heavy/full local builds/suites: bounded targeted tests only GOMAXPROCS=2 GOFLAGS=-p=2, full validation hosted by manager. Commit named files with Sprint: #557 and correct Story/Story-ID trailers and injected BASHY_AGENT. Report exact evidence and limitations. Scope tests/literal-bytes/gate.sh lines109/122. Hosted41334176466 failed measured30/expected60 failed2 because rg command missing, masked by ||true; Go15cases each passed. Preserve15cases x4transports=60 and all byte/refusal assertions. Repair portable exact case-name accounting or explicit prerequisite fail-fast. Add bounded counting regressions missing/duplicate/unexpected/failed/skipped; no full local corpus. Evidence/tmp/s557-corpus-latest-failed.log. Do not alter approved candidate registry.

Independent review and takeover delivery:

Preserve external worker commit c5ba188 and add a focused follow-up. The original
portable helper correctly rejected duplicate outcomes but still counted duplicate
passes as measured successes; exclude duplicates alongside failures and skips.
Keep all ordered comparisons in LC_ALL=C, scoped to a subshell so caller settings
are unchanged. Explicitly propagate counting-tool and pipeline errors even when
the caller disables errexit; include wc in prerequisite checks.

Sixteen bounded synthetic accounting checks pass, covering all-pass, missing,
duplicate, unexpected, failed, skipped, absent log, mixed PASS/FAIL and PASS/SKIP,
command failure, pipeline-left failure, wc failure, missing tool, non-C caller
locale, and caller-state preservation. Syntax checks pass for all three shell
files. The corpus remains 15 cases across four transports; no byte/refusal
assertions, candidate approval records, or suite counts were changed. Full
literal-byte/corpus execution remains a hosted manager gate.

## Superseded by consolidation — 2026-11-02

Operator requested consolidation of implementation repairs into acceptance owners. Scope, original acceptance, unresolved work and evidence transfer to public umbrella #6, story `bf7624a5c2f2`. This record is closed by decision with label `superseded`; it is NOT a delivery acceptance, test pass, waiver or feature retirement. The parent remains open. Continue corrections under that parent rather than creating another repair story. Original content above is retained unchanged.
