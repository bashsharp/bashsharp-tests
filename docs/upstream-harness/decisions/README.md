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
| D2-deadline.tsv | 22 | interpreter per-call cost: the 18 Barrier B roots + 4 Barrier C regressions (162 D2) |
| D10-deadline-new.tsv | 8 | PENDING the D10 profile shard (7 newly-reached interpreted + `rangegen` compiled) |
| D3b-frontend.tsv | 5 | 3 source-preserving front-end rows + 2 execute-phase `.s` companions (162 D3(b)) |
| D4-cgo.tsv | 12 | cgo (162 D4; 6 blocking in 152 + 6 retained) — D4 said 13: the 13th is not a cgo declaration by source (harness-1 FINDINGS) |
| D5-gc-only.tsv | 29 | gc-only checks go/types cannot express (162 D5; `notinheap2` joined at C) |
| D7-unsafe-pointer.tsv | 26 | `unsafe.Pointer` memory reinterpretation + GC/finalizer observation (162 D7) |
| D9-finalizer-makefunc.tsv | 29 | SetFinalizer+GC 15 · MakeFunc 6 · AllocsPerRun 3 · retained ValueOf 5 (165 D9) |

Blocking roots by ID: 143 (+ 8 pending D10). `rangegen.go` appears in D2
(interpreted) and D10 (compiled) — two modes, one root.
