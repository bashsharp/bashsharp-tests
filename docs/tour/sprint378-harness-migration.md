# Sprint 378 / Story 1546: tutorial script entry

The 2026-10-04 operator addition supersedes the proposed corpus preload.
An explicit `--bashpp` / `--bashsharp` selects interpretation even with
`--source=go` and a `.go` operand. The five interpreted argument builders in
`tools/upstream-harness/testdata/backend/bashpp_backend_test.go` and
`tools/upstream-harness/testdata/go-backend/bashpp_backend.go` all pass
`--bashpp`; none omit it. `TestInterpretedBuildersOverrideGoExtension` checks
those builders. The shell gates delegate to these hooks. Upstream sources,
package plans, and corpus gating policy are unchanged by this story.

Tour's 97 executable sources are renamed `.go` to `.bsh`, byte for byte.
Interpreted execution is plain `bashy x.bsh`; the four build-only rows keep
`--check` and do not execute their bodies. Native execution builds a temporary
`.go` copy. Both staged forms are hash-checked after execution. Transpile keeps
`--bashpp --source=go`, including when the operand has the `.bsh` extension.
Refresh and upstream-copy validation translate only executable inventory paths;
the nine excluded Go fragments and 62 inline fragments retain their keys.

Go by Example stages byte-identical `.bsh` copies at runtime and invokes
`bashy x.bsh [args...]`. Its testing row retains a bound, unchanged source copy
and uses a separately generated `.bsh` entry containing that source's test
assertions plus the existing `testing.Main` driver. The entry helper is bound
in the evidence recipe. Native and compiled recipes retain their Go inputs.
No Go by Example source, inventory, classification or source pin is changed.
No summary tool was added; successful Go by Example output remains the gate's
existing `*.jsonl.pass` convention.

## Exactly which tables and pins changed

- `tests/tour/inventory.tsv`: 97 executable path suffixes only; 168 total rows.
  Every source byte count and SHA-256 is unchanged. Data SHA-256 changes from
  `a63b57fe844b26387c7eeb511035ed77dbf7255d6250a57cb96d83289965cf78` to
  `64c086e517d257960328e711a0c9825ce20076793d0dc67616e41c1190390697`.
- `docs/tour/pin.tsv`: inventory data hash and rename provenance; upstream
  repository, commit, module checksum, license, and denominator unchanged.
- `tests/tour/results.tsv`, `results.linux-x86_64.tsv`, and
  `results.windows_nt-x86_64.tsv`: 97 path suffixes each. Observations, source
  digests, statuses, and toolchain identities are unchanged, not newly measured.
- `docs/tour/baseline-pin.tsv`, `baseline-pin.linux-x86_64.tsv`, and
  `baseline-pin.windows_nt-x86_64.tsv`: resulting records/data hash bindings.
- `docs/tour/accepted-observations.tsv`: whole-file result/pin hashes for those
  three pairs; platform/toolchain identities unchanged.
- `docs/tour/semantics.tsv` and `volatility.tsv`: executable path suffixes only.
- `docs/tour/executor-contract.tsv`: plain interpreted entry and `{NATIVE_SRC}`
  for the temporary oracle copy; compiled flags unchanged.
- `docs/tour/corpus.tsv`: rename provenance only; counts/license hash unchanged.
- `tools/upstream-harness/backend-pin.tsv`: only `verifier_test`, binding the
  new override assertion. Backend hooks, upstream pins and corpus inventories
  are unchanged. `bashy/.sibling-pins` is unchanged.

Historical JSONL ledgers are intentionally not relabeled or resealed as new
execution evidence. The Linux run must produce a current ledger before using
`executor-tamper-tests.sh` against it.

## Focused host verification

Installed candidate: bashy `35a3aef`, Darwin arm64. No full corpus gate ran.

- Inventory re-derivation and corpus byte comparison against the pinned
  x/website module: 168/168 rows, 97 programs + LICENSE, pass.
- Offline accepted-baseline validation: all three platform tables pass.
- Corpus tamper suite: 18 passed, 0 failed.
- Executor synthetic self-tests: 106 passed, 0 failed.
- Semantic comparator self-tests: 189 passed, 0 failed.
- Go unit tests in `tools/tour` and `tools/go-by-example/gbe`: pass.
- Backend verifier tests, including all five explicit interpreter overrides:
  pass (`go test tools/upstream-harness/backend-verify{,_test}.go`).
- Tour opt-in `TestTourBshSmoke`: `welcome/hello` and `flowcontrol/for` match
  native output in interpreted and compiled modes (`Hello, 世界` and `45`).
- Go by Example opt-in `TestInterpretedBshSmoke`: hello-world,
  command-line-arguments, testing-and-benchmarking pass as plain `.bsh` entries;
  program arguments and both test bodies are checked.

Limitations observed, not reported as passing:

- The older baseline tamper suite passed 17 probes but its bounded process-tree
  cleanup probe failed on this sandbox (process-list access unavailable and a
  missing run-output file after cleanup). Recheck on Linux.
- Real-ledger executor tamper testing against the committed historical `.go`
  ledger yielded 9 passes / 18 failures because its source keys, recipe and
  hash bindings predate this migration (and it already carries older candidate
  bindings). Re-run with freshly generated Linux evidence; the synthetic gate
  tests above exercise the new contract without claiming a full execution.

Focused runtime tests can be repeated from their module directories with
`S378_BASHY=/absolute/bashy` (and `S378_GO=/absolute/pinned/go` for Tour), using
`go test -run TestTourBshSmoke -v .` or
`go test -run TestInterpretedBshSmoke -v .`. They are opt-in so routine unit tests
do not run either full gate or depend on a locally installed candidate.
