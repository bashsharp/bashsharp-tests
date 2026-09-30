---
id: 462f08f99dea
kind: task
title: S263 Windows full Bash# feature sweep on noviwin1
seq: 92
status: todo
priority: p1
labels:
    - windows
    - bashsharp
created: 2026-09-23T17:26:37.742115Z
sprint: 263
sprint_id: 0d309714-8bdb-5839-9c40-8bff200ed2f5
sprint_title: Bash# full feature sweep on Windows, macOS, and Linux
---

On Windows noviwin1, run every applicable existing gate in the Sprint263 inventory against a clean candidate at public heads: Bash# five-family acceptance and lowering, decorators, agentic, fenced languages/manifest integration, Bash++/Bash# interpreter/runtime full-tier tests and relevant Bashy/library tests. Record command, candidate SHA, denominator, raw log/hash, and pass/fail/skip for each gate. Preserve fixture bytes and time limits. File narrow repair stories for real failures, fix, and rerun affected gates until green. Reuse Sprint250 receipts only where exact candidate and gate are identical.

## Review 2026-09-30 (steward)

- Status: todo. Blocked on the S263 inventory (d94af6237d36); there is nothing to run until the gate list exists.
- Outdated: "Reuse Sprint250 receipts": the tour, Go by Example, Go Tour and GNU Bash 5.3 rows were re-measured on the Windows test host later (the sprint-269 reference and the v0.31.0 published-byte QA). Cite those and do not rerun them. "Bash++" now reads Bash#.
- Host: the Windows test host answered over ssh on 2026-09-28 but shows offline on the peer board. Confirm ssh and claim the host before starting. The remote shell is cmd, and podman must be started in the same session.
- Known reds to account for rather than rediscover: story 9df13ad43092 (8 sh/pathconv + 12 coreutils/tool native Windows reds).
- Next step: run only the inventory's Bash#-specific rows that apply to Windows, at the recorded candidate (default: the bashy v0.31.0 tag and its pinned siblings). Keep one evidence record for the host.
- Acceptance: every applicable row has a result with command, SHA, denominator and log hash. Each failure has a linked repair story or is one of the 9df13ad43092 reds.
- Deps: d94af6237d36; host claim.
