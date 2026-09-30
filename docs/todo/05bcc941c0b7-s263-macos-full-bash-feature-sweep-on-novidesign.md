---
id: 05bcc941c0b7
kind: test
title: S263 macOS full Bash# feature sweep on novidesign
seq: 93
status: todo
priority: p1
labels:
    - macos
created: 2026-09-23T17:28:05.602479Z
sprint: 263
sprint_id: 0d309714-8bdb-5839-9c40-8bff200ed2f5
sprint_title: Bash# full feature sweep on Windows, macOS, and Linux
---

Run every applicable existing Bash++/Bash# feature gate from the S263 inventory on novidesign at clean recorded public candidate heads. Capture exact commands, counts, raw logs, SHA/candidate manifest, environment prerequisites and skips. Reuse Sprint250 receipts only for an identical candidate and gate. File narrow S263 fix stories for failures, repair root cause, rerun affected gates; preserve fixtures and time limits.

## Review 2026-09-30 (steward)

- Status: todo. Blocked on the S263 inventory (d94af6237d36).
- Outdated: "Reuse Sprint250 receipts": the macOS tour and 86/86 rows were re-measured on the macOS test host for v0.31.0 (published-byte QA 5/5, tour 39/0/1, Bash 5.3 86/86). Cite them and do not rerun.
- Host: the macOS test host was in use on 2026-09-28 and is shared with local-model and bench work. Claim it before starting, and keep heavy suites off it while other workers run (flaky reds under load were seen before).
- Known reds: the sh `-tags full` interp tier had ~29 pre-existing reds on macOS at v0.31.0 (10 confirmed on the v0.30.0 engine too). The inventory must list each one by ID with a story or an explicit reason, or this story can't close.
- Next step: run the inventory's Bash#-specific rows (acceptance/lowering, decorators, agentic, polyglot/manifest, bashsharp go test, sh full tier) at the recorded candidate (default: the bashy v0.31.0 tag and its pinned siblings). Keep one evidence record.
- Acceptance: every applicable row passes, or its failures map to known-red IDs or new repair stories, with command, SHA, denominator and log hash.
- Deps: d94af6237d36; host claim.
