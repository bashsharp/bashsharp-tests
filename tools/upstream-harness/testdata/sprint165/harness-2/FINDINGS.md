# Sprint 165 — lane harness-2 (S165.0: the identity handoff, D8)

Seam: `tools/upstream-harness/testdata/backend/**` (the patched testdir
runner's Bash++ backend hook), `backend-verify.go` / `backend-verify_test.go`,
`backend-pin.tsv`. Authority: the exact upstream Go 1.27 harness. The
identity-keyed `internal` visibility rule (D8 = (a)) is on sh master
(`syntax.BashPPInternalImportVisible`, the map importer, the runtime
resolver, `lower.moduleImporter.ImportFromPackage`); it decides on the
DECLARED identity, so a phase that passes none falls to the directory rule.
This lane makes the backend pass upstream's OWN identity — never a new one —
at the two places it was missing, per the "harness + backend seam" requests
of the gosource-vis ledger (`sh/gosource/testdata/sprint165/gosource-vis/
FINDINGS.md`).

## Rows

| root | mode | first cause (Barrier C / run 0) | mechanism | status |
| --- | --- | --- | --- | --- |
| testdir:escape_runtime_atomic.go | compiled | `escape_runtime_atomic.go:12:2: could not import internal/runtime/atomic (use of internal package … not allowed)` — upstream's `compileFile` compiles it with `-p=p`; the backend's transpile argv carried no `--go-import-path`, so the identity rule could not apply | request (a): every compile phase whose direct `go tool compile` argv carries `-p` (and `-D`) hands it to Bash++ as `--go-import-path` (`--go-import-base`) | seam fixed in 844626f; the compiled verdict is now the product's — leaf pending on the next integrated sh candidate |
| testdir:escape_runtime_atomic.go | interpreted | `-m` optimizer diagnostics, no check-interface meaning | — | retained (non-blocking), unchanged |
| testdir:intrinsic.go | compiled | `intrinsic.dir/main.go:9:4: … internal/runtime/sys` at the directory compile — the phase already carried `--go-import-path main`; refused by `lower`'s directory rule until the gosource-vis + lower diff | gosource-vis (sh) | not a harness row; closes with the sh candidate |
| testdir:intrinsic.go | interpreted | same first line at `--check`; after gosource-vis the execute phase of the single-package directory program carried **no** identity (`program.mapArgs` filled only with earlier packages) and would fall to the directory rule at execute | request (b): the remembered program of every directory compile phase carries the identity and its map args, single-package included | seam fixed in 844626f; the next defect on this root is the product's — measured on the fixture (`sysdir.go` interpreted): the check is admitted, the execute reaches the bridge worker build, which still applies the directory rule (`gosource: build dependency bridge: … use of internal package internal/runtime/sys not allowed`) — §4.3 of the multi-package design, reported "not applied" by gosource-vis; owner runtime/bridge lane (`interp/bashpp_native_bridge.go`, `bashpp_import_scratch.go`). Also `-d=ssa/intrinsics/debug` expectations have no interpreted meaning (the compiled row is the one that can pass) |
| fixture:identity-handoff (sysdir compiled) | compiled | `backend-verify`: `link adopted artifact …/a.exe, want the last compile phase's […/main.o]` on a passing single-package rundir root | pre-existing verifier defect (not a hook defect): since the direct-gc route (Sprint 162, 5badc98) the link phase records the linked `a.exe` as the program artifact; `verifyRunDirRow` still wanted the compile object there, so every compiled rundir root verified as a seam FAIL line (the corpus verdicts come from upstream's terminal and were unaffected; `rundir-gate.sh` compiled mode would have reported a seam defect) | fixed in 13fb238: the link input must be the compile object by name, the program artifact must be the link's own artifact (as `verifyBuildDirRow` already checked); pinned by `TestVerifierRunDirLinkAdoptsCompiledObject` |

## Mechanism (one reading, two handoffs, one verifier rule)

- **`compilerIdentity(nativeArgv, compileInputs)`** in the hook (and its
  twin in `backend-verify.go`): the `-D` / `-p` of a direct `go tool compile`
  argv as the compiler's own flag parsing reads them — the LAST occurrence
  wins. This matters: `nowritebarrier.go` (`errorcheck -+ -p=runtime`) and
  `internal/runtime/sys/inlinegcpc.go` (`-p=internal/runtime/sys`) carry
  their own `-p` after upstream's `-p=p`, and gc compiles them under the
  recipe's identity; the seam hands exactly that one. Compile inputs are
  excluded by name; `go run` / `go build` / `go tool asm` / link / execute
  argv carry none and none is invented. Which phases gain an identity:
  `compile` (`-p=p`), `errorcheck` / `errorcheckwithauto` (`-p=p`),
  `errorcheckoutput`'s compile (`-p=p`), the `builddir` / `buildrundir` Go
  compile (`-D . -p=main`; the `-D` is upstream's own and handed too — the
  same handoff as the directory phase's `-D test`). `run` / `runoutput` /
  `build` / `buildrun` / `asmcheck` / `runindir`: none (cmd/go shapes; the
  `run` fast path with `-p=main` is disabled in backend mode by the backend
  patch, so it never reaches the seam).
- **The remembered program** (`backendProgram`) carries `identity` and the
  full `mapArgs` (`--go-import-base <-D> --go-import-path <-p>` + earlier
  groups) for every directory compile phase — the `len(earlier) != 0`
  condition is gone — and for the directory build. The interpreted execute
  phase's `run-remembered-program` argv therefore starts with the identity
  for a single-package program too.
- **Events**: every backend event records `import_base` / `import_path`
  (the identity of the phase being planned; kept in `backendIdentities`, a
  per-test `sync.Map` like `backendPackages`, for the duration of
  `backendPlan`); the program record records both as well.
- **Verifier** (`checkIdentity`, in `verifyRow`'s per-phase loop, every
  row): for every `check-*` / `transpile-*` disposition the recorded
  identity must equal the one derived from the recorded native argv (a
  dropped one and an invented one are both seam failures), and a directory
  phase's upstream `package` record must agree with its argv.
  `checkProgramIdentity` (rundir link, builddir pack): the program's identity
  is its compile phase's and its map args hand both flags. The go-backend
  (package) seam's `import_path` / `test_main` are untouched (harness-1).

## Fixture set — `testdata/backend/testdata/sprint165/identity-handoff/` (driving test `identity-handoff-gate.sh`)

Through the exact patched runner in both modes on the dev host
(darwin/arm64, bashy built from the umbrella at the sh commit carrying the
gosource-vis mechanism and the lower `ImportFromPackage` diff):

| root | shape | interpreted | compiled |
| --- | --- | --- | --- |
| `atomicfile.go` | `// compile`, `package p`, imports `internal/runtime/atomic` (identity `p`) | pass, COMPILE-ONLY-PASS, admitted | pass, COMPILE-ONLY-PASS, admitted |
| `atomicescape.go` | `// errorcheck -0 -m -l` (escape_runtime_atomic's shape) with the escape expectations gc emits on the generated file | fail, UNSUPPORTED (optimizer diagnostics, as before) | pass, DIAG-PASS, admitted |
| `plainfile.go` | `// errorcheck`, no internal import, one ordinary diagnostic (canary for the `-p=p` handoff on every errorcheck root) | pass, DIAG-PASS | pass, DIAG-PASS |
| `dottedfile.go` | `// errorcheck -p=example.com/dotted` importing `internal/runtime/atomic`; the refusal `use of internal package internal/runtime/atomic not allowed` is the expected diagnostic at the import line | pass, DIAG-PASS (refused, gc's wording, upstream's own errorCheck matched it) | pass, DIAG-PASS |
| `nonefile.go` | `// run` importing `internal/runtime/atomic` (native `go run`, no `-p`) | fail, RUN-PRODUCT-FAIL, output `nonefile.go:14:2: could not import internal/runtime/atomic (use of internal package … not allowed)` | same |
| `sysdir.go` + `sysdir.dir/main.go` | `// rundir`, single package `main` importing `internal/runtime/sys` (intrinsic's shape), `sysdir.out` | fail, RUNDIR-PRODUCT-FAIL: check admitted (first phase exit 0), execute reaches the bridge worker build (§4.3, see rows); output must NOT carry `could not import internal/runtime/sys` | pass, RUNDIR-PASS, admitted |

The gate also runs the hook's own unit test of the argv reading
(`TestCompilerIdentityReadsUpstreamArgv`) inside the patched overlay (trap
4: the hook compiles only there). `asm-companion-gate.sh` re-run after the
change: PASS (the directory build's compile now carries `--go-import-base .
--go-import-path main`; plaindir both modes PASS, asmdir/asmonly/asmhdr
compiled PASS, asmmissing PRODUCT-FAIL, asmrun D3(b) refusal unchanged).

## Verification (darwin)

`gofmt -l` clean on the hook, verifier and its test; `GO111MODULE=off go
test -count=1 backend-verify.go backend-verify_test.go` ok (new:
`TestCompilerIdentityReadsUpstreamArgv`,
`TestVerifierRequiresUpstreamIdentityHandoff`,
`TestVerifierRunDirProgramKeepsIdentity`,
`TestVerifierRunDirLinkAdoptsCompiledObject`; `buildDirEvidence` updated to
carry the identity); `package-verify.go` and `partition-emit.go` tests ok;
`identity-handoff-gate.sh` PASS (cf6066b); `asm-companion-gate.sh` PASS; `git diff
--check` clean; `backend-pin.tsv` updated for `backend_hook`, `verifier`,
`verifier_test` (both blocks, as harness-1 did).

## Requests to other seams

- **runtime/bridge lane (sh interp)**: the interpreted execute of an admitted
  internal import (`intrinsic.go` interpreted, `sysdir.go` on the fixture)
  now reaches the bridge worker build with the identity on the request, and
  the worker's `go build` of the scratch session refuses `internal/runtime/sys`
  by cmd/go's directory rule (`gosource: build dependency bridge: exit status
  1: … use of internal package internal/runtime/sys not allowed`). §4.3 of
  `sh/docs/bashpp-multi-package-execution.md` (worker build with an
  importcfg / the declared identity) is the mechanism; every interpreted
  package row is by ID (D1) this sprint, and `intrinsic.go` interpreted also
  needs `-d=ssa/intrinsics/debug` expectations that the check interface
  cannot emit — so the compiled row is the one D8 closes.
- **manager**: nothing to decide for this lane; the leaf request below is
  the evidence for the two rows.

## Leaf request (manager submits)

`leaf-harness-2-1.tsv` (repo root; `--harness-bundle harness-2.bundle` on
the next integrated sh candidate): `testdir:intrinsic.go`,
`testdir:escape_runtime_atomic.go` + canaries `testdir:escape_sync_atomic.go`,
`testdir:escape_unsafe.go`, `testdir:escape_iface.go` (same `errorcheck -0
-m -l` family, no internal import), `testdir:alias3.go`, `testdir:ddd2.go`
(multi-package rundir, the map path), `testdir:fixedbugs/issue9608.go`,
`testdir:dwarf/dwarf.go` (the only single-package `rundir` roots of the
corpus — request (b)'s exposure; PASS at C), `testdir:alias2.go`,
`testdir:blank1.go` (plain errorcheck, `-p=p` handoff), `testdir:closure6.go`,
`testdir:fixedbugs/bug020.go` (plain compile), `testdir:fixedbugs/issue4909b.go`
(errorcheckoutput), `testdir:nowritebarrier.go` (recipe `-p=runtime` after
upstream's `-p=p` — the last-wins reading; a 151/152 row at C, its verdict
must not change class). Expect: `escape_runtime_atomic` compiled PASS,
`intrinsic` compiled PASS, `intrinsic` interpreted FAIL with the bridge
worker's refusal as the first line (not `could not import`); canaries
unchanged.
