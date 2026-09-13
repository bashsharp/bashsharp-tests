# Barrier C — Sprint 162's frozen integrated candidate, full corpus gate

Sprint: #162 · Story: S162.0 (`cda64bde8fea`). One coordinator on the
certification host, fresh `/srv/sprint162/barrier-c`, the exact upstream
Go 1.27 harness, `GOMAXPROCS=2 GOFLAGS=-p=2`, 60 s backend bound.
2026-09-13 **09:03–10:29Z** (86 min), `corpus exit=3`, survivors 0.

| | value |
|---|---|
| candidate `integ-3` | **sh `2004a4a0`** · **bashy `fe204ec`** · **coreutils `539b8fa7`** · readline `b958823` · filebrowser `cde11469` |
| `bashy.real` sha256 | `3b09dfe341dd2a8c…` |
| harness | `b94bfc8a` (partition v10.5; backend: gc-direct compile-only recipes, one-library-per-package overlay route) |
| Go | 1.27.0 linux/amd64, binary `1db869c5…` (authenticated SDK) |
| native witness | testdir 2726 terminals / 37 non-PASS (the frozen skips); typechecker 2 raw non-PASS; packages 26/0 — identical to Barrier B |

## Result

| | PASS (both modes) | FAIL | SKIP | of 3,456 applicable |
|---|---:|---:|---:|---|
| Barrier B (published candidate) | 2,562 | 894 | 39 | 74 % |
| **Barrier C (integ-3)** | **2,722** | **734** | **39** | **79 %** |

By runner: testdir 1,999 PASS / 690 FAIL / 37 SKIP; typechecker 723 / 18 / 2
(the 156 native-only leaves listed by ID with zero credit, as at B);
package 0 / 26 / 0 in the required both-modes sense — **3 of 26 PASS
compiled** (`abt`, `compare`, `devirtualize`) through the D1 overlay route,
interpreted recorded by ID.

Per lane: interpreted 716 testdir non-PASS (B: 850), compiled **109** (B: 203
rows across owners); typechecker 20 raw non-PASS in both backend lanes.

**Sprint 155 D3 blocking** (compiled rows + interpreted-only rows that are
not compiler-artifact recipes): **761 → 559** (compiled roots 203 → 113,
interpreted-only 558 → 446). Not zero: the sprint closes with a residue
ledger, and every remaining root is either a product row owned by
mechanism in Sprint 165 or recorded by ID on card #162 (D1–D7).

`corpus-verify.go -evidence evidence`: the same 312 native executions as
Barrier B — the 156 native-only typechecker leaves × 2 lanes, listed by ID,
zero credit (the S155.11 denominator rule) — and no other violation.

## Partition v10.5 owners (roots)

| owner | Barrier B | Barrier C | movement |
|---|---:|---:|---|
| 151 evaluator | 502 | **326** | 129 → PASS, 39 → 153 (runtime panics now reach the next defect), 11 → unclassified, 1 → retained |
| 152 lowering | 96 | **29** | 26 → PASS (body-less, layout/`//line`, package clause), 41 → retained (their compiled rows pass; the interpreted `-m`/`-d=` rows are zero-credit) |
| 153 runtime | 105 | **143** | 6 → PASS; +39 from 151, +3 from unclassified, +5 regressions (below) |
| 154 diagnostics | 7 | **3** | 3 → PASS (issue11362, import6, slice3err), 1 → 151; issue4468 / issue50372 / nul1 remain |
| package | 26 | **26** | 3 PASS compiled, all 26 interpreted by ID (D1) |
| unclassified | 25 | **32** | 2 → PASS, 3 → 153; +11 from 151, +1 new — to be moved by manifest in S165.5 |
| retained (interpreted artifact rows only) | 133 | **175** | +41 from 152, +1 from 151 |
| **total** | **894** | **734** | |

Compiled rows at C by owner: 151 39 · 152 29 · 153 11 · 154 3 · package 23
· unclassified 8 = 113.

## Regressions against Barrier B (6 roots, PASS at B, non-PASS at C)

- `atomicload.go`, `fixedbugs/issue22781.go`, `rotate1.go`, `rotate2.go` —
  interpreted `command exceeded time limit`. Sprint 153 recorded these four
  as deadline roots that had *just* come under the 60 s bound on the host;
  the per-call cost added by the run-time-error, storage-lock and
  frame-table mechanisms pushed them back over it. D2 applies (never a
  timeout raise); the cost is the design finding of
  `sh/docs/bashpp-interpreter-per-call-cost.md`. Owner 153 in Sprint 165.
- `fixedbugs/bug491.go` — interpreted stray output where the `.out` is
  absent (output barrier). Owner 153.
- `fixedbugs/bug130.go` — unclassified; owner by first cause in S165.5.

## Recorded by ID on card #162 (FAIL by decision, never relabeled)

D1 package roots interpreted (26) · D2 deadline family (19 + the 4 above) ·
D4 cgo (13) · D5 gc-only checks go/types cannot express (28) · D7
`unsafe.Pointer` memory reinterpretation (24) · GC/finalizer and
`reflect.MakeFunc` retained-callback rows · issue23586 / issue4468 /
issue50372 (source-preserving front end).

## Evidence

`/srv/sprint162/barrier-c/{logs,evidence,manifests}` on the certification
host (build caches removed); `/srv/sprint162/candidates/integ-3/` is the
frozen candidate tree. Manifests copied here: `barrier-c/active-*.tsv`,
`barrier-c/status.txt`. Sprint 165 starts from these manifests
(`docs/sprint-165-handoff.md` in the umbrella).
