# Recorded-by-ID ledger — Sprint 165 (from Barrier C, confirmed by run 0)

One TSV per decision (`root · mode · decision · ground`). A root listed here
is FAIL by decision on card #162 / #165 — never relabeled, never `retained`,
and never staffed as a mechanism. Barrier D's second exit clause ("every
remaining root recorded by ID") is evaluated against these files, not
against memory. Decision text: `docs/sprint-162-master-execution-plan.md`
(D1–D7) and `docs/sprint-165-master-execution-plan.md` (D8–D12) in the
umbrella; the card threads carry the recording notes.

| file | roots | decision |
|---|---:|---|
| D1-package-interpreted.tsv | 26 | multi-package interpreted execution (162 D1) |
| D2-deadline.tsv | 18 | interpreter per-call cost: the 18 Barrier B roots (162 D2); the 4 Barrier C regressions moved to D10a — they PASS at the bound on both 165 leaves |
| D10-deadline-new.tsv | 13 | DECIDED 14:45Z on the profile shard (`leaf-165-d10-profile`, 600 s measurement bound): 6 at-the-bound roots are mechanism candidates (D10a, not by ID), 3 join D2 by ID (D10b/c), `rangegen` compiled → 152, `chan/select2` + 4 stack-exceeds rows → 153 (D10d/e) |
| D3b-frontend.tsv | 5 | 3 source-preserving front-end rows + 2 execute-phase `.s` companions (162 D3(b)) |
| D4-cgo.tsv | 12 | cgo (162 D4; 6 blocking in 152 + 6 retained) — D4 said 13: the 13th is not a cgo declaration by source (harness-1 FINDINGS) |
| D5-gc-only.tsv | 29 | gc-only checks go/types cannot express (162 D5; `notinheap2` joined at C) |
| D7-unsafe-pointer.tsv | 26 | `unsafe.Pointer` memory reinterpretation + GC/finalizer observation (162 D7) |
| D9-finalizer-makefunc.tsv | 29 | SetFinalizer+GC 15 · MakeFunc 6 · AllocsPerRun 3 · retained ValueOf 5 (165 D9) |
| S248-runtime-observations.tsv | 7 | gc toolchain/runtime observations interpreted Bash# does not mimic (operator decision D1, 2026-09-22; sh `docs/bashpp-compiler-artifact-contracts.md` Sprint 248 section); interpreted mode only, leaf verdict stays FAIL, compiled stays PASS; replacement contracts in `TestS248RuntimeObservationReplacementConformance` |
| S247-value-provenance.tsv | 4 | fieldtrack-experiment and string-layout roots interpreted Bash# does not mimic (operator decision, 2026-09-22; sh `docs/bashpp-compiler-artifact-contracts.md` Sprint 247 section); interpreted mode only, leaf verdict stays FAIL, compiled stays PASS; replacement contracts in `TestS247DecisionReplacementConformance` |
| S249-compiler-artifacts.tsv | 5 | gc compiler-artifact assertions Bash# does not implement (249 C2, sh@08743660 `docs/bashpp-compiler-artifact-contracts.md`); both requested modes per root, leaf verdicts stay FAIL; replacement contracts pinned by `TestS249CompilerArtifactReplacementConformance` via `S249-evidence-pin.tsv`, authenticated by `tools/upstream-harness/artifact-decision-verify.go` |

Blocking roots by ID: 142 = 143 − 4 (regressions → D10a) + 3 (divmod, heapsampling, rangegen-interpreted → D2 by D10b/c). `rangegen.go` appears in D2
(interpreted) and D10 (compiled) — two modes, one root.
