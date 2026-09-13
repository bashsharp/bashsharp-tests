# Sprint 165 — lane harness-4 (S165.0 / #73: package identities and file roles, D8)

Seam: `tools/upstream-harness/testdata/go-backend/**` (the patched cmd/go
runner's Bash++ hook and its unit test), `package-verify.go` /
`package-verify_test.go`, `corpus-verify.go` (one reader), `backend-pin.tsv`.
Authority: the exact upstream Go 1.27 cmd/go. Product side: sh story #98
(`f206111602d9`, package roots compiled) — the identity-keyed `internal`
visibility (D8 (a)) decides on the DECLARED identity, so the harness must
hand cmd/go's own identity for each unit, and each file under cmd/go's own
role. No SDK pin, manifest, verdict, denominator or timeout changed; the
four D8 negatives of the verifier (fact not declared, identity is the
package, flag missing, flag on another identity) are kept; there is still no
native tested-source fallback (the compiled script is `set -e` and cmd/go
runs only on the overlay that mapped every original).

## Rows (Barrier C `active-package-manifest.tsv`, leaf evidence on integ-4)

| root | mode | first cause | harness defect | status |
| --- | --- | --- | --- | --- |
| package:cmd/compile | compiled | integ-4: `main.go:8:2: could not import cmd/compile/internal/amd64 (use of internal package … not allowed)` (Barrier C: `internal/buildcfg`, the top-level shape the sh mechanism closed) | the library transpile was checked under `--go-import-path cmd/compile.test --go-test-main`; `cmd/compile.test` is outside `cmd/compile/`, so the tested package's OWN internal siblings are refused — cmd/go grants them to `cmd/compile` (ptest is compiled `-p cmd/compile`) | seam fixed: the library is checked under `<pkg>`, the fact is never asserted on a library (leaf pending) |
| package:cmd/internal/testdir | compiled | C: `testdir_test.go:17:2: could not import internal/testenv` (closed by sh on the identity); integ-4: the xtest-only unit must form the external test package | the harness hands the one file as `--go-xtest-file` under `<pkg>` (verified: `xonly` fixture, role `xtest`, identity `cmd/internal/testdir`); bashy's `transpile --go-library` derives the tested package's name from the in-package unit and refuses `--go-xtest-file must form the external test package` when there is none | harness honest; **request to the bashy seam** (below) |
| package:cmd/compile/internal/{amd64, loopvar, rangefunc, reflectdata, ssa, types, types2}, go/types, internal/types/errors | compiled | C: `exit status 1` (no message) | the old walk listed the xtest files under the tested package's variant (cmd/go's ptest still carries `XTestGoFiles` as metadata) AND under `<pkg>_test` → `library output collision` → `configuration-error`, `/bin/sh -c exit 1` — nine harness rows that never reached Bash++ | seam fixed: the in-package variant never takes `XTestGoFiles`; each file once under its role (leaf pending) |
| every package with in-package test files (20 of 26) | interpreted | C: `gosource: duplicate file "…/x_test.go" in package "<pkg>"` | cmd/go's ptest lists the test files in `GoFiles` AND `TestGoFiles`; the map handed them twice | seam fixed: the map lists each file once (interpreted stays by ID, D1 — the product's next defect surfaces) |
| every package with xtest files and Go files (9) | interpreted | (masked by the identity refusal or the duplicate) | the xtest files were filed into the tested package's map entry (`package x_test; expected package x`) | seam fixed |
| compiled route, any package whose transpile succeeded | compiled | `package-verify`: `json: cannot unmarshal object into … planRecord.overlay`; then `read overlay JSON: … no such file`; then `compile trace does not prove …`; `corpus-verify`: `incomplete compiled package plan` on the proof line | four pre-existing evidence defects of the S162 overlay route (never exercised by a verifier on a passing package): `readPlans` decoded the proof as a plan; the overlay dir lived in cmd/go's `$WORK` (deleted before verification); the `-n` trace captured stdout (empty — a dry run prints to stderr); `corpus-verify` refused the proof record | fixed; pinned by `TestReadPlansSeparatesProofsFromPlans`, `TestCorpusVerifyAcceptsCompiledOverlayProof`, and the driving gate's `PACKAGE-PASS` rows |
| compiled route, any package Bash++ refused | compiled | `package-verify`: `wanted exactly one overlay proof, got 0` (a seam FAIL, exit 1) | a refusal left no proof and was indistinguishable from a broken script | the script records `transpile.status`; a refusal is `PACKAGE-PRODUCT-FAIL`, a clean transpile without a proof stays a seam failure |

## Mechanism (two identities, three roles, one verifier rule each)

- **`bashppIdentity`** (`bashpp_backend.go`): `bashppProgramIdentity(pmain)`
  = `<pkg>.test` + TestMain (`--go-import-path <pkg>.test --go-test-main`),
  handed only to the invocation that receives `_testmain.go` (interpreted);
  `bashppLibraryIdentity(p)` = `<pkg>`, no fact, handed to the compiled
  route's one `transpile --go-library`. cmd/go's own testmain is compiled
  natively under the overlay in compiled mode and needs no fact from Bash++.
- **`bashppFileRole`** (`go` / `test` / `xtest` → `--go-file` /
  `--go-test-file` / `--go-xtest-file`): `bashppTestVariantFiles` classifies
  each of cmd/go's two test variants once per path — ptest's `GoFiles` with
  `TestGoFiles` deciding `test`, never its `XTestGoFiles`; pxtest's
  `GoFiles` as `xtest`. Both modes use it (the interpreted map and the
  library). A package whose tests are all external is one xtest unit.
- **Events**: `import_path` / `test_main` (the program), `library`
  (`import_path`, `test_main: false`, `roles`), per map entry `roles`,
  `transpile_status`; the overlay directory is `<events>.overlay/`.
- **Verifier**: `verifyIdentity` — interpreted argv carries `--go-import-path
  <pkg>.test --go-test-main` and no other identity; the compiled script
  carries `'--go-import-path' '<pkg>' '--go-library'`, never
  `'--go-test-main'`, and the `library` record agrees. `verifyRoles` — every
  file once with a role; `<pkg>` entry: `go` or `test` (a `_test.go` name is
  never `go`), `<pkg>_test` entry: `xtest` only; compiled: the script hands
  each file under its role's flag and the library's roles agree.
  `transpileRefused` reads the status file.

## Fixture set — `testdata/go-backend/testdata/sprint165/package-identity/` (driving test `package-identity-gate.sh`)

Module `example.com/pkgid`; each tested package is the parent of its own
`internal/deep`, so `<pkg>` admits the import and `<pkg>.test` is refused
(cmd/compile's exact shape). Through the exact patched cmd/go in both modes
on the dev host (darwin/arm64, bashy built from the umbrella at sh
`9e4a7d80`), hook unit tests run inside the overlay first:

| package | shape | interpreted | compiled |
| --- | --- | --- | --- |
| `lib` | `lib.go` (go, imports `lib/internal/deep`), `lib_test.go` (test), `lib_x_test.go` (xtest, imports lib and the internal) | fail, PACKAGE-PRODUCT-FAIL: `could not import …/internal/deep (go list failed … go.mod file not found …)` — the interpreter resolves module imports outside the module (product, by ID) | **pass, PACKAGE-PASS** (overlay proof verified: 3 generated files, `-n` trace, digests) |
| `cmdmain` | `main.go` (package main, imports its internal), `main_test.go` | fail, PACKAGE-PRODUCT-FAIL (same `go list` line) | **pass, PACKAGE-PASS** |
| `xonly` | `xonly_test.go` only (package `xonly_test`, imports `xonly/internal/deep`) | fail, PACKAGE-PRODUCT-FAIL: `could not import …/xonly/internal/deep (use of internal package … not allowed)` from the map's `<pkg>_test` identity — cmd/go loads an xtest's imports with the TESTED package as the importer (`load/test.go`: `loadImport(…, p, …)`), the sh rule keys on `<pkg>_test` (request below) | fail, PACKAGE-PRODUCT-FAIL: `transpile: --go-xtest-file must form the external test package` (bashy, request below); status file `2`, no proof |

Mutation checks: handing the library `<pkg>.test --go-test-main` fails the
hook unit test; regressing only the transpile argv fails the gate end to end
(verifier `compiled transpile does not check the library under the tested
package's own identity`, terminal fail, `use of internal package` in the
output). `expect.tsv` asserts compiled `lib`/`cmdmain` PASS and that no row
carries `use of internal package` (compiled) or `duplicate file` /
`expected package` (interpreted).

## Verification (darwin)

`gofmt -l` clean on every changed Go file; `GO111MODULE=off go test
-count=1 package-verify.go package-verify_test.go` ok (new:
`TestVerifyRolesFormTheTestPackages`, `TestReadPlansSeparatesProofsFromPlans`;
`TestVerifyIdentityRequiresTheTestMainFact` keeps the four interpreted D8
negatives and adds the library's); `corpus-verify.go corpus-verify_test.go`
ok (new: `TestCorpusVerifyAcceptsCompiledOverlayProof`, fails against the
old reader); hook unit tests in the overlay ok (`TestTestVariantFilesClassifyEachFileOnce`,
`TestIdentitiesSeparateTheLibraryFromTheTestMain`, the overlay-script test);
`package-identity-gate.sh` PASS; `git diff --check` clean; `backend-pin.tsv`
`gotest_hook` / `package_verifier` refreshed (both blocks). `package-gate.sh`
and the corpus gate need the pinned candidate and were not run here.

## Requests to other seams

- **bashy (`internal/agentos/transpile.go`, `dispatchTranspileLibrary`)**:
  with no in-package unit (`cmd/internal/testdir`, the `xonly` fixture) the
  external test package cannot be formed: `packageName` is empty and the
  `_test` check refuses. go/build's own rule for a directory of external
  tests only: the tested package's name is the xtest package's name without
  `_test` (`build.ImportDir` sets `p.Name` that way). Until then
  `cmd/internal/testdir` compiled stops at that line — a product row, the
  harness hands the role and the identity honestly.
- **sh (`syntax.BashPPInternalImportVisible` at the map site)**: an external
  test package's identity `<pkg>_test` is not inside `<pkg>/`, but cmd/go
  loads an xtest's imports with the tested package as the importer, so
  `<pkg>_test` sees exactly what `<pkg>` sees. The interpreted map (identity
  `<pkg>_test` by cmd/go's own path, which `_testmain.go` imports) is refused
  the tested package's own internal siblings (`xonly` interpreted). No
  corpus xtest imports its own package's internal tree today; recorded for
  the rule, not blocking.
- **manager**: the leaf request below; nothing else to decide.

## Leaf request (manager submits)

`leaf-harness-4-1.tsv` (repo root; `--harness-bundle harness-4.bundle` on
the next integrated sh candidate): `package:cmd/compile` (the identity
row), `package:cmd/internal/testdir` (the xtest-only row), the nine
collision rows' cheap members `package:cmd/compile/internal/amd64`,
`package:cmd/compile/internal/loopvar`, `package:cmd/compile/internal/rangefunc`,
`package:cmd/compile/internal/reflectdata`, `package:cmd/compile/internal/types`,
`package:internal/types/errors`; the compiled PASS canaries
`package:cmd/compile/internal/abt`, `package:cmd/compile/internal/compare`,
`package:cmd/compile/internal/devirtualize` (their interpreted rows lose the
`duplicate file` cause); `package:cmd/compile/internal/base` (top-level
internal under the standard identity `<pkg>`, unchanged expectation) and
`package:cmd/compile/internal/syntax` (in-package test importing
`internal/testenv`). Expect: `cmd/compile` compiled no longer fails at an
internal import of its own tree; the nine collision rows reach Bash++
(their first cause becomes the product's); no interpreted row carries
`duplicate file`; `abt`/`compare`/`devirtualize` compiled stay PASS;
`cmd/internal/testdir` compiled reports the bashy line above unless the
candidate's bashy forms the xtest-only package. Not run here; nothing is
claimed.
