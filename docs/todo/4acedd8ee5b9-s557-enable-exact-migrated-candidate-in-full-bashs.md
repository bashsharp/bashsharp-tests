---
id: 4acedd8ee5b9
kind: bug
title: S557 enable exact migrated candidate in full Bashsharp corpus gate
seq: 111
status: wontfix
priority: p1
labels:
    - superseded
created: 2026-11-01T15:38:03.989859Z
assignee: s557-public-tooling
sprint: 557
sprint_id: f5ccf563-f5d1-5b94-8be4-9388e51ef86d
sprint_title: 'Bashy v1.0.0 preparation: split private/public development and retire qiangli/dhnt'
closed: 2026-11-02T05:32:08.725784Z
---

The full harness invokes tools/go-by-example/gate.sh without its required candidate manifest. Support a reviewed exact migrated candidate and explicit manifest input through the existing full harness and public workflow. Preserve candidate identity/hash checks and existing corpus PASS/FAIL/PLANNED accounting; never bypass candidate review or auto-approve unknown artifacts. Determine the existing registry contract, implement the missing plumbing and record reviewable candidate evidence. Full suites remote only. No private data or licensed fixtures.


Follow-up implementation for the first-row input mutation: exact candidate
Yoke revision 621b5f18 requires a producer-owned tree completion record, while
the harness only seeded an SDK symlink. Initialize each reviewed coordinate
through the authenticated candidate before row snapshots, retain bounded
initialization evidence, and re-authenticate original and installed SDKs.
No input-guard exclusions or marker synthesis. The original failed attempt
lost detailed mutation paths, so historical attribution remains a hypothesis.
Focused mocked initialization/provisioning tests pass; full hosted replay is
pending manager integration. Status remains assigned for manager acceptance.

## Superseded by consolidation — 2026-11-02

Operator requested consolidation of implementation repairs into acceptance owners. Scope, original acceptance, unresolved work and evidence transfer to public umbrella #6, story `bf7624a5c2f2`. This record is closed by decision with label `superseded`; it is NOT a delivery acceptance, test pass, waiver or feature retirement. The parent remains open. Continue corrections under that parent rather than creating another repair story. Original content above is retained unchanged.
