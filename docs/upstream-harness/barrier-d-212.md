# Barrier D (barrier-d-212) — the Bash# front door + rename, replayed against Barrier D

Sprint: #211 · Story: S211.5 (`d351213bcd3d`) with S212.0 (`8838347d736c`).
One coordinator on the leaf host, fresh `/srv/sprint162/barrier-d-212`, the
same upstream Go 1.27.1 harness partition as Barrier D (harness `cce9612`:
only the tool resolution changed — `BASHPP_TOOL` names `cmd/bashsharp`),
`GOMAXPROCS=2 GOFLAGS=-p=2`, 60 s backend bound. 2026-09-18
**06:59–08:42Z** (103 min), `corpus exit=3`, survivors 0.

| | value |
|---|---|
| candidate `integ-212` | **sh `a3fd0fbe`** (07e501a0 + the `set -o bashsharp` alias) · **bashsharp `4167be8`** · **bashy `b8fc397`** · coreutils `5de4206b` · yoke `2854c03` · readline `b958823` · filebrowser `cde11469` |
| tool measured | **`bashsharp`** (`cmd/bashsharp`) sha256 `2d142dc619ae092f…` — the language's own binary, not `bashy.real` |
| harness | `cce9612` (Barrier D's `8efe846` + Sprint 211's tool resolution and sibling names) |
| Go | **1.27.1** linux/amd64, binary `30969f97…` (authenticated SDK, `/srv/sprint206`) |
| what changed vs Barrier D | the front door: bashy's dialect selector + direct Go-source interface + transpile moved to the `bashsharp` module and run through its own binary; the language renamed (`.bsh`, `--bashsharp`, aliases); ONE engine line (`set -o bashsharp` accepted as an alias). The evaluator, lower, gosource, polyglot and grammar are the same code as Barrier D/D′ |

## Result

| | PASS (both modes) | FAIL | SKIP |
|---|---:|---:|---:|
| Barrier D (integ-206) | 2,827 | 631 | 39 |
| Barrier D′ (integ-208) | 2,826 | 632 | 39 |
| **Barrier D (integ-212)** | **2,826** | **632** | **39** |

By ID against Barrier D's manifests (`tools/upstream-harness/barrier-byid.py`,
665 failing keys → 666): **0 fixed, 1 new, 29 first-line flips** — every flip
a tmp path, a pointer, an object count or a nondeterministic reflect message,
the same set D′ saw.

| key | Barrier D | integ-212 full run | re-measure alone (`delta-remeasure/`, 08:45–08:47Z) |
|---|---|---|---|
| `testdir:fixedbugs/issue22781.go` interpreted | PASS | `command exceeded time limit` | **PASS** |

The one delta is the deadline family (the 60 s bound on 2 vCPU, design-level
since Sprint 153); re-measured alone on the same candidate it passes.
**Barrier D on the Bash# front door ≡ Barrier D by ID.**

## The run before it: barrier-d-211 (the finding)

The first full run of this sprint, `barrier-d-211/` (candidate `integ-211`:
sh `07e501a0`, bashpp `a2c48c0`, bashy `3972eb8`, tool `bashpp`
`abcc39f8…`, 04:20–05:57Z), also came out 2,826 / 632 / 39 but by ID showed
**1 fixed + 3 new**: two deadline roots and **`testdir:args.go` interpreted
`panic: argc`** — a real regression of the new front: under
`--go-file=args.go -- arg1 arg2` the binary took `arg1` as the program
operand. Fixed in bashsharp `e2a30b0` (an operand is taken only when no
`--go-file` was given, exactly as bashy), verified on every argv form, and
the full replay above is on the fixed binary. That is what a barrier is for.

## What this proves and what it does not

It proves the front-door split and the rename did not change what the
engine does on the Go corpus, measured through the language's own binary
rather than assumed from the engine commit being (nearly) identical. It is
not a new baseline for Sprint 209: the target catalog stays keyed to Barrier
D. It says nothing about Tour/GbE (unchanged code paths, measured through
bashy; not re-run this sprint) or about the contract/decorator natives,
which are bashy's and are not on the corpus harness's argv.
