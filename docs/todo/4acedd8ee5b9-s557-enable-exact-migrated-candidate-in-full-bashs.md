---
id: 4acedd8ee5b9
kind: bug
title: S557 enable exact migrated candidate in full Bashsharp corpus gate
seq: 111
status: assigned
priority: p1
created: 2026-11-01T15:38:03.989859Z
assignee: s557-public-tooling
sprint: 557
sprint_id: f5ccf563-f5d1-5b94-8be4-9388e51ef86d
sprint_title: 'Bashy v1.0.0 preparation: split private/public development and retire qiangli/dhnt'
---

The full harness invokes tools/go-by-example/gate.sh without its required candidate manifest. Support a reviewed exact migrated candidate and explicit manifest input through the existing full harness and public workflow. Preserve candidate identity/hash checks and existing corpus PASS/FAIL/PLANNED accounting; never bypass candidate review or auto-approve unknown artifacts. Determine the existing registry contract, implement the missing plumbing and record reviewable candidate evidence. Full suites remote only. No private data or licensed fixtures.
