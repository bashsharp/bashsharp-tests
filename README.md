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
| **Go corpus** (script claim) | the upstream Go 1.27.1 test corpus: 3,497 roots (`test/` dir roots, typechecker roots, package roots), pinned by SHA with the exact upstream runners | `tools/upstream-harness/barrier-run.sh` (≈ 100 min); `barrier-byid.py` compares two runs by root ID | the native Go oracle passes **and** both Bash# script modes — interpreted (`--bashsharp --source=go`, the harness route) and lowered-then-compiled — reproduce it; native PASS alone earns nothing. Reported per mode with exact failures and exclusions; lowered-then-compiled is not the island route |
| **Go Tour · Go by Example** | 97 / 85 programs (291 / 255 observations over native, interpreted and compiled), pinned upstream sources committed verbatim under `tour/` and `examples/` | `tools/tour/`, `tools/go-by-example/` | all three modes agree with native; mandatory script gates |
| **Go delta table** (Sprint 379 S6) | `tests/go-delta/`: one fixture per row of `bashsharp/godelta/go-delta.tsv` (mixed, unit and island surfaces) | `tools/go-delta-gate.sh`, wired into `harness/run.sh` | exit status, exact stdout and the published refusal text of every row; needs the AgentOS `bashy` and a provisionable Go toolchain for the island rows |
| **Sharp tier** | keyword/default arguments, exhaustive enums, deep `readonly`, null-safety | `tools/bashsharp/acceptance.sh` over `tests/bashsharp/*/cases.tsv` | exact transcript + status, interpreted; lowering parity in `lowering.tsv`; a near-miss must behave exactly as plain bash |
| **Decorators · contracts · agentic** | `tests/decorators/`, `tests/agentic/` (+ `bashy/test/contracts/`) | `tools/decorators/`, `tools/agentic/` | exact transcript + status; the yield status 6 is asserted, never inferred |
| **Literal bytes (Sprint 381 S2; add to Sprint 263 inventory)** | `tests/literal-bytes/cases.tsv` (15 source-byte families) | `tests/literal-bytes/gate.sh`, wired into `harness/run.sh` | exact file/argv bytes through `bashy -c`, ycode's JSON tool input, yoke MCP `run_tool`, and the generated install-agent shim; any red fails |
| **Polyglot islands** | python · typescript · rust · c/c++ · go · powershell · csharp · bash/sh fixtures | `tools/polyglot-gate.sh`, `tools/python-package-gate.sh`, `tools/sprint184-typescript-gate.sh` | callables with typed values crossing the boundary, on the host's toolchain (powershell/csharp through the S358.11 tour fixtures on the provisioned runtime) |
| **Lowering** | `tests/lowering/` | `tools/lowering/` | interpreted and compiled outputs byte-identical |

Fixture status is declared, never inferred: `tests/manifest.tsv` marks each
adapted fixture `supported` or `planned`, and a `planned` fixture is expected
to fail. The Go corpus inventory is a pinned, checksum-verified denominator
(`tools/go-corpus/refresh.sh` to refresh, `tools/go-corpus/validate.sh` on
every run); `docs/upstream-harness/` holds every barrier record.

## The current standing

From the last full barrier (go1.27.1, Sprint 376 final m8, 2026-10-05,
engine `sh 0e1b00108`), copied — never edited — from
`bashsharp/docs/go-corpus-state-2026-10-05.md`:

- Go corpus, interpreted: **3,177 PASS · 281 FAIL · 39 SKIP** of 3,497 roots
  (3,458 applicable). Native and lowered-then-compiled: **3,458/3,458 PASS**
  each.
- Every one of the 281 is a reviewed exclusion by root ID (compiler
  diagnostics 108, asmcheck 75, gc-only checks 30, `unsafe` reinterpretation
  20, gc observation 16, compute-bound 11, whole-package source 9, cgo 8,
  assembly input 4). **0 new failures, 0 regressions, 0 missing IDs.**
- Tour of Go **97/97** and Go by Example **85/85**, in all three modes.
- The Go constructs that differ from Go are the rows of
  [`bashsharp/docs/go-delta.md`](https://github.com/bashsharp/bashsharp/blob/main/docs/go-delta.md)
  (`bashy explain go`), proven by `tools/go-delta-gate.sh`.
- Bash 5.3 OFF 86/86, ON 79 + 7 (last measured on the `sprint-269` tag).

Whole-corpus 3,497/3,497 will not be claimed, and no Go percentage is.

## Running it

```sh
# the gate that says whether a change broke Bash 5.3 (needs a built bashy next door)
tools/classic-gate.sh --bashy ../bashy --sh ../sh --coreutils ../coreutils --out /tmp/classic

# the Go delta table: each row's fixture, against a built bashy
BASHY_BIN=../bashy/bin/bashy tools/go-delta-gate.sh

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

As of the 2026-10-05 barrier no *repair* roots remain in the Go corpus: every
interpreted failure is a reviewed exclusion (see the standing above). The
on-ramp is the **`gap` and `limited` rows of the Go delta table**
([go-delta.md](https://github.com/bashsharp/bashsharp/blob/main/docs/go-delta.md);
`bashy explain go`): each names a construct, the diagnostic, the workaround
and the fixture in `tests/go-delta/` that pins today's behaviour. Fix it in
`sh/interp` (interpreted) or `sh/lower` (compiled), change the row in
`bashsharp/godelta/go-delta.tsv` and its fixture in the same change (a row
that stops being true fails `tools/go-delta-gate.sh`), run `go generate
./godelta`, and re-run the lane. A fix that makes a construct work in one
mode only is not done; a fix that regresses the classic gate is not
accepted. The earlier [go-corpus-targets.md](https://github.com/bashsharp/bashsharp/blob/main/docs/go-corpus-targets.md)
catalog describes the 2026-09-17 classification and is history.

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
[LICENSE](tour/LICENSE).

The Go by Example programs under `examples/` are the upstream files from
[mmcgrana/gobyexample](https://github.com/mmcgrana/gobyexample) at commit
`7d705626375ba0263b616865a286e1587d6989c8`, byte for byte, saved with the Bash#
extension `.bsh`. Credit goes to Mark McGranaghan and the upstream contributors; the
licence is CC BY 3.0. The unchanged `examples/UPSTREAM-README.md` states: “This work is
copyright Mark McGranaghan and licensed under a [Creative Commons Attribution 3.0
Unported License](http://creativecommons.org/licenses/by/3.0/).”
