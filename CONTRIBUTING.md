# Contributing to the Bash# conformance suite

Thank you. This repo is where Bash# claims are decided, so contributions here
are the most direct way to move the language.

## The layout you need

Bash# is built from flat siblings. Clone them next to each other:

```sh
for r in sh coreutils yoke bashsharp bashy bashsharp-tests; do git clone https://github.com/qiangli/$r; done
cd bashy && ./scripts/bootstrap-siblings.sh && make build    # bin/bashy, bin/bash
cd ../bashsharp && go build ./cmd/bashsharp                  # the front door the harness measures
```

`bashy/.sibling-pins` names the exact sibling commits a bashy commit is built
against; `bootstrap-siblings.sh` moves a clean sibling to its pin.

## The three suites, locally

1. **Classic (must never regress):** `tools/classic-gate.sh --bashy ../bashy --sh ../sh --coreutils ../coreutils --out /tmp/classic` — Bash 5.3's own suite with the dialect off (86/86) and on (79 + 7). Needs a controlling terminal (run it in a real terminal, not under `nohup`).
2. **The lane you touched:** `tools/bashsharp/acceptance.sh` (Sharp tier), `tools/decorators/`, `tools/agentic/`, `tools/polyglot-gate.sh`, `tools/lowering/`. Each is a few minutes.
3. **The Go corpus:** `tools/go-corpus/refresh.sh` once (fetches and verifies the pinned Go 1.27.1 source), then either the single root you are fixing (`bashsharp --source=go go/test/<root>` vs `go run`), or the whole barrier `tools/upstream-harness/barrier-run.sh` (≈ 100 min) compared **by root ID** with `barrier-byid.py` against the previous record in `docs/upstream-harness/`.

A PR that fixes a corpus root includes: the engine change (in `qiangli/sh`), the fixture flip here (`planned` → `supported`, or the new case), and the classic gate's unchanged result. Both Bash# modes — interpreted and lowered-then-compiled — must match the oracle; one mode is not done.

## Picking something

- **`corpus-repair` + `good first issue`**: one issue per repairable Go root, with its first cause and area (`area:evaluator`, `area:lowering`, `area:runtime`, `area:diagnostics`). Comment "taking it" so two people don't collide; one PR per root.
- **`island`**: a new fenced language, end to end, from the issue template.
- A behaviour that differs from GNU Bash 5.3 or from POSIX: open an issue with a reproducer; a fixture is worth more than a description.

## Rules that are not negotiable

- **Every number names its corpus.** A result is "N of M on suite S at revision R". Never "100 %", never "conformant", never a number without the suite.
- **Spelling is exact Go.** `go f(x)`, `defer f(x)`, `x, err := f()`, `make(chan T, n)`, `select { … }`. No substitutes.
- **A syntax change is an RFC first** (`bashsharp/rfcs/`): the shape, its collision class against stock bash, how it lowers to plain Go, one fixture. Nothing is admitted that cannot lower to ordinary Go or that re-spells something Go already has.
- **Never edit a pinned transcript to make a check pass.** If the transcript is wrong, the PR says why and re-pins it as its own commit.
- **No private details** — hostnames, user paths, tokens — in any file, commit message, issue or PR. These repos are public.
- Commits carry a **DCO sign-off** (`git commit -s`: "Signed-off-by: Name <email>"), certifying you have the right to submit the work under BSD-3-Clause.

## Questions

Open a [Discussion](https://github.com/qiangli/bashsharp/discussions) on the language repo — the pinned FAQ answers the common ones.
