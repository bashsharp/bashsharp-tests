---
id: 74939b0d9269
kind: test
title: S263 Linux full Bash# feature sweep on DO2
seq: 94
status: todo
priority: p1
labels:
    - linux
created: 2026-09-23T17:28:05.682741Z
sprint: 263
sprint_id: 0d309714-8bdb-5839-9c40-8bff200ed2f5
sprint_title: Bash# full feature sweep on Windows, macOS, and Linux
---

Run every applicable existing Bash++/Bash# feature gate from the S263 inventory on a DigitalOcean Linux droplet (DO2) at clean recorded public candidate heads. Capture exact commands, counts, raw logs, SHA/candidate manifest, environment prerequisites and skips. Reuse Sprint250 receipts only for an identical candidate and gate. File narrow S263 fix stories for failures, repair root cause, rerun affected gates; preserve fixtures and time limits.

## Review 2026-09-30 (steward)

- Status: todo. Blocked on the S263 inventory (d94af6237d36).
- Outdated: "DO2" was a Sprint 250-era kept droplet. Linux test hosts now come from the separate ephemeral DigitalOcean team (the ephemeral test-host runbook; a new droplet needs operator approval, and it is destroyed after the run), which is how v0.31.0 QA ran. A kept droplet may be reused only if it is still up and is claimed. "Reuse Sprint250 receipts" is also stale: the Linux tour, GbE, Go Tour and 86/86 rows were re-measured later (the sprint-269 reference; v0.31.0 QA).
- Next step: pick the host (reuse a claimed kept droplet or request one ephemeral droplet), then run the inventory's Bash#-specific Linux rows at the recorded candidate (default: the bashy v0.31.0 tag and its pinned siblings). Keep one evidence record and destroy the droplet afterwards.
- Acceptance: every applicable row has a result with command, SHA, denominator and log hash. Failures get linked repair stories, and nothing is left unexplained.
- Deps: d94af6237d36; operator approval for a new droplet.
