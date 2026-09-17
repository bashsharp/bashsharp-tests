# Barrier D — Sprint 206's frozen go1.27.1 candidate, full corpus gate

Sprint: #206 · Story: S206.3 (`77a9182086df`). One coordinator on the
certification host, fresh `/srv/sprint162/barrier-d`, the exact upstream
Go 1.27.1 harness, `GOMAXPROCS=2 GOFLAGS=-p=2`, 60 s backend bound.
2026-09-17 **20:37–22:19Z** (102 min), `corpus exit=3`, survivors 0.

| | value |
|---|---|
| candidate `integ-206` | **sh `a83e6a33`** · **bashy `88f7dbf`** · **coreutils `20205973`** · readline `b958823` · filebrowser `cde11469` |
| `bashy.real` sha256 | `9b3d25063d15f037…` |
| harness | `8efe846` (pins moved to go1.27.1; partition rules unchanged) |
| Go | **1.27.1** linux/amd64, binary `30969f97…` (authenticated SDK, `/srv/sprint206`) |
| native witness | testdir 2728 terminals / 37 non-PASS (the frozen skips, unchanged); typechecker 2 raw non-PASS; packages 26/0 — Barrier C's witness plus the two go1.27.1 roots |

## Result

| | PASS (both modes) | FAIL | SKIP | of applicable |
|---|---:|---:|---:|---|
| Barrier C (integ-3, go1.27.0) | 2,722 | 734 | 39 | 79 % of 3,456 |
| **Barrier D (integ-206, go1.27.1)** | **2,827** | **631** | **39** | **82 % of 3,458** |

By runner: testdir 2,092 PASS / 599 FAIL / 37 SKIP; typechecker 735 / 6 / 2
(the 156 native-only leaves listed by ID with zero credit, as at B and C);
package 0 / 26 / 0 in the required both-modes sense — **25 of 26 PASS
compiled** (C: 3), interpreted recorded by ID (D1).

Per lane (`corpus-verify`): interpreted testdir 596 FAIL, compiled testdir
**30** FAIL (C: 109 compiled rows); typechecker 6 raw non-PASS in each
backend lane. The verifier reports the same 312 native executions as B and
C (156 native-only typechecker leaves × 2 lanes) and no other violation;
`corpus-verify.summary.txt` and its read manifest are beside the manifests.

The two roots go1.27.1 added: `fixedbugs/issue81165.go` PASS both modes;
`fixedbugs/issue80976.go` FAIL both modes (`gosource: unsupported call target`,
instantiated method type arguments) — owner 151. Both denominators are
published: 3,495 original / 3,497 adjusted.

## Partition v10.6 owners (roots)

| owner | Barrier C | Barrier D |
|---|---:|---:|
| 151 evaluator | 326 | **252** |
| 152 lowering | 29 | **16** |
| 153 runtime | 143 | **111** |
| 154 diagnostics | 3 | **2** |
| package | 26 | **26** (25 pass compiled; interpreted by ID) |
| unclassified | 32 | **35** |
| retained (interpreted artifact rows only) | 175 | **189** |
| **total** | **734** | **631** |

## Against the Sprint 174 baseline (umbrella reconciliation)

All 661 blocking modes of the 559-root Barrier C baseline now have a D
verdict: **175 pass / 486 fail / 0 unmeasured** (was 137 / 192 / 332
unmeasured). 106 of the 559 roots pass every originally blocking mode
(75 had strict evidence before). Of the 93-key Sprint 200 residual ledger, 1
key passes (`fixedbugs/issue54220.go` interpreted), 92 persist.

## Regressions against Barrier C (2 keys)

- `fixedbugs/bug285.go` interpreted — owner 151.
- `fixedbugs/issue43164.go` compiled — `LOWER-ETYPE: SplitAfterN redeclared
  in this block` — owner 152.

Both are recorded on the Sprint 206 residual ledger under their owner cards.
No timeout was raised, no fixture edited, nothing relabeled.

## Evidence

`/srv/sprint162/barrier-d/{logs,evidence,manifests}` on the certification
host (build caches removed); `/srv/sprint162/candidates/integ-206/` is the
frozen candidate tree; `/srv/sprint206/` holds the authenticated go1.27.1 SDK
and its identity record. Manifests copied here: `barrier-d/active-*.tsv`,
`barrier-d/status.txt`, `barrier-d/corpus-verify.*`. The umbrella's
`docs/sprint-206-*.tsv` + `sprint-206-reconciliation.json` carry the per-key
reconciliation and the receipt digests.
