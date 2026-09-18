# Barrier D′ (barrier-d-208) — the coreutils/yoke split, replayed against Barrier D

Sprint: #208 · Story: S208.4 (`5751875647dd`). One coordinator on the leaf
host, fresh `/srv/sprint162/barrier-d-208`, the exact upstream Go 1.27.1
harness at the same commit as Barrier D, `GOMAXPROCS=2 GOFLAGS=-p=2`, 60 s
backend bound. 2026-09-18 **01:15–02:58Z** (103 min), `corpus exit=3`,
survivors 0.

| | value |
|---|---|
| candidate `integ-208` | **sh `a83e6a33`** (= integ-206) · **bashy `38bd54b`** · **coreutils `5de4206b`** · **yoke `2854c03`** · readline `b958823` · filebrowser `cde11469` |
| `bashy.real` sha256 | `ff44c1d80f5d4a25…` |
| harness | `8efe846` (= Barrier D) |
| Go | **1.27.1** linux/amd64, binary `30969f97…` (authenticated SDK, `/srv/sprint206`) |
| what changed vs integ-206 | nothing in the engine: coreutils split into the certified required set + yoke, bashy's imports repointed; the sh commit is identical |

## Result

| | PASS (both modes) | FAIL | SKIP |
|---|---:|---:|---:|
| Barrier D (integ-206) | 2,827 | 631 | 39 |
| **Barrier D′ (integ-208)** | **2,826** | **632** | **39** |

By ID against Barrier D's manifests (665 failing keys → 667):

| key | Barrier D | D′ full run | re-measure (`delta-remeasure/`, 3 roots alone, 03:09–03:12Z) |
|---|---|---|---|
| `testdir:convinline.go` compiled | PASS | `command exceeded time limit` | **PASS** |
| `testdir:ken/chan.go` interpreted | PASS | `command exceeded time limit` | **PASS** |
| `testdir:uintptrescapes.go` interpreted | FAIL, `command exceeded time limit` (153) | FAIL, `BASHPP-EEXPR-FORM … BashPPAddressExpr` (151) | FAIL, `command exceeded time limit` (153) — as at D |

Nothing fixed, nothing else moved. All three deltas are the deadline family
(the 60 s bound on 2 vCPU, recorded as design-level since Sprint 153); the
re-measure restores Barrier D's verdict on every one. **Barrier D′ ≡
Barrier D by ID.** The full-run manifests are in `barrier-d-208/`, the
re-measure in `barrier-d-208/delta-remeasure/`.

## What this proves and what it does not

It proves the coreutils/yoke split did not change what the Bash++ engine
does on the Go corpus — which follows from the sh commit being identical
and is now measured rather than assumed. It is not a new baseline for
Sprint 209: the target catalog stays keyed to Barrier D.
