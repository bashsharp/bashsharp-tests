# bashsharp-tests — the conformance gate for Bash#

This repo decides what [Bash#](https://github.com/bashsharp/bashsharp) may
claim. It is **TDD-first and red on purpose** where the language is not
finished: a test that fails here is a target with an owner, not a bug in the
suite. A claim about Bash# that is not a result from this repo is not a claim
— the numbers are kept, with their corpora, in
[bashsharp/docs/claims.md](https://github.com/bashsharp/bashsharp/blob/main/docs/claims.md).

> Renamed from `bashpp-tests` on 2026-09-18 with the language. Harness
> internals keep their `BASHPP_*` spellings; the tests are the same tests.

## What is measured, and by what

| lane | corpus | tool | what PASS means |
|---|---|---|---|
| **Classic — GNU Bash 5.3** | Bash's own test suite, every runnable fixture (86) | `tools/classic-gate.sh` (runs `bashy`'s `make test-bash` with the dialect OFF and ON) | OFF 86/86; ON 79 + 7 (the seven name a shell function `func`; listed) |
| **Classic — POSIX** | the POSIX baseline fixtures under `tests/00_superset_posix2016/`, plus the licensed VSC shell arm and yash's POSIX suite run from the bashy repo | `tests/`, `bashy/scripts/yash-posix-suite.sh` | the dialect is inert under `--posix` |
| **Go corpus** | the upstream Go 1.27.1 test corpus: 3,497 roots (`test/` dir roots, typechecker roots, package roots), pinned by SHA with the exact upstream runners | `tools/upstream-harness/barrier-run.sh` (≈ 100 min); `barrier-byid.py` compares two runs by root ID | the native Go oracle passes **and** both Bash# modes — interpreted (`--source=go`) and lowered-then-compiled — reproduce it; native PASS alone earns nothing |
| **Go Tour · Go by Example** | all 291 / 255 programs, pinned upstream sources committed verbatim under `tour/` and `tests/go-by-example/` | `tools/tour/`, `tools/go-by-example/` | same dual-mode rule |
| **Sharp tier** | keyword/default arguments, exhaustive enums, deep `readonly`, null-safety | `tools/bashsharp/acceptance.sh` over `tests/bashsharp/*/cases.tsv` | exact transcript + status, interpreted; lowering parity in `lowering.tsv`; a near-miss must behave exactly as plain bash |
| **Decorators · contracts · agentic** | `tests/decorators/`, `tests/agentic/` (+ `bashy/test/contracts/`) | `tools/decorators/`, `tools/agentic/` | exact transcript + status; the yield status 6 is asserted, never inferred |
| **Polyglot islands** | python · typescript · rust · c/c++ · go · bash/sh fixtures | `tools/polyglot-gate.sh`, `tools/python-package-gate.sh`, `tools/sprint184-typescript-gate.sh` | callables with typed values crossing the boundary, on the host's toolchain |
| **Lowering** | `tests/lowering/` | `tools/lowering/` | interpreted and compiled outputs byte-identical |

Fixture status is declared, never inferred: `tests/manifest.tsv` marks each
adapted fixture `supported` or `planned`, and a `planned` fixture is expected
to fail. The Go corpus inventory is a pinned, checksum-verified denominator
(`tools/go-corpus/refresh.sh` to refresh, `tools/go-corpus/validate.sh` on
every run); `docs/upstream-harness/` holds every barrier record.

## The current standing

From the last full barrier (go1.27.1, 2026-09-17), copied — never edited —
from `bashsharp/docs/go-corpus-state-2026-09-17.md`:

- Go corpus: **2,827 PASS · 631 FAIL · 39 SKIP** of 3,497 roots (3,458 applicable).
- Of the 631: **296 repair** (a defect with a first cause — see *Contributing*),
  6 review, 71 blocked-design (needs a design, never excluded), 258 excluded
  (not achievable by an interpreter; each listed with its reason).
- Bash 5.3 OFF 86/86, ON 79 + 7; Go by Example 255/255; Tour 291/291 on the same revisions.

Whole-corpus 3,497/3,497 will not be claimed.

## Running it

```sh
# the gate that says whether a change broke Bash 5.3 (needs a built bashy next door)
tools/classic-gate.sh --bashy ../bashy --sh ../sh --coreutils ../coreutils --out /tmp/classic

# the Sharp tier, decorators, polyglot lanes
tools/bashsharp/acceptance.sh
tools/polyglot-gate.sh

# the full Go corpus replay (≈ 100 min; a refactor is green only when its barrier equals the previous one BY ID)
tools/upstream-harness/barrier-run.sh
tools/upstream-harness/barrier-byid.py <previous> <this>
```

The harness resolves the binary under test through `BASHPP_TOOL` (default:
`bashsharp`'s `cmd/bashsharp`, the front door over the engine alone); the
classic lanes measure the built `bashy` / `bash`. Everything assumes the
flat-sibling checkout (`../bashy`, `../bashsharp`, `../sh`, `../coreutils`,
`../yoke`).

## Contributing

The 296 *repair* roots are the on-ramp: each is a Go program the upstream
oracle passes and Bash# does not, with a first cause and an owner class in
[go-corpus-targets.md](https://github.com/bashsharp/bashsharp/blob/main/docs/go-corpus-targets.md)
(`.tsv` beside it). Pick one, run it under `bashsharp --source=go`, compare
with the oracle, fix it in `sh/interp` (interpreted) or `sh/lower`
(compiled), add or un-`planned` its fixture, and re-run the lane. A fix that
makes a root pass in one mode only is not done; a fix that regresses the
classic gate is not accepted.

Spelling is exact Go: `go f(x)`, `defer f(x)`, `x, err := f()`,
`make(chan T, n)`, `select { … }`. No substitutes.

## History

The suite's earlier front page — the TDD plan, the 2-tier scope, the
Sprint 98 denominator work and the first Go oracle — is preserved as
[docs/README-history-2026-09-18.md](docs/README-history-2026-09-18.md);
`PLAN.md` and `TDD_PLAN.md` are the original method documents.

## License

The harness, tools and fixtures written here are BSD-3-Clause ([LICENSE](LICENSE)).
Upstream material keeps its own license: the Go test corpus is fetched from
the pinned Go source archive at run time (`tools/go-corpus/refresh.sh`
refuses an archive without its LICENSE) rather than committed; the Go Tour
programs are committed verbatim under `tour/` with upstream's
[LICENSE](tour/LICENSE); the Go by Example fixtures are adapted from
gobyexample.com and attributed in `docs/go-by-example/README.md`.
