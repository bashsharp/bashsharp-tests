# Sprint 165 — lane harness-1 (S165.0: partition v10.6 + the backend seam)

Seam: `tools/upstream-harness/**` (partition-emit, backend-verify,
package-verify, the testdir backend hook `testdata/backend/`, the cmd/go
hook `testdata/go-backend/`). Authority: the exact upstream Go 1.27
harness. Every row below is a Barrier C row this lane looked at; the
partition rows move by rule commit + regeneration
(`docs/upstream-harness/barrier-c-v10.6/`), never in place.

## Partition v10.6 (D11) — rows moved by rule

| root | first cause (Barrier C) | mechanism | status |
| --- | --- | --- | --- |
| chan/doubleselect, const8, bug115, bug273, bug424, issue24693, issue4370, issue54911, issue6847, shift3 | compiled `LOWER-ETYPE` / `LOWER-EUNDEFINED` on the generated file | v10.6a lower-diagnostic | moved 151 → 152 in 4525499 |
| fixedbugs/issue19467 | compiled runtime frame names `main.(*__gosource_pkg_0_WaitGroup).Add` | v10.6a mangled-name | moved 151 → 152 in 4525499 |
| interface/embed3 | compiled panic message names `main.__gosource_pkg_0_I1` (the interpreted row wants `p.I1` — the same mangled type name) | v10.6a mangled-name | moved 153 → 152 in 4525499 |
| fixedbugs/issue18459, fixedbugs/issue18882 | gc refuses the `//go:` pragma at `main.go:6` of the generated file | v10.6a pragma-position | moved 151 → 152 in 4525499 |
| fixedbugs/issue20298 | gc `"bufio" imported and not used` on the generated file (position class) | v10.6a unused-import | moved 151 → 152 in 4525499 |
| fixedbugs/issue23586 | gc `"fmt" imported and not used` on the generated file; the root itself is the by-ID front-end row (gc type-checks after syntax errors on its own tree) | v10.6a unused-import | moved 151 → 152 in 4525499 — the ledger keeps it by ID; the manifest owner is now the lower owner, whose compiled row it is |
| linkname3 | compiled `compilation succeeded unexpectedly` (gc found nothing in the generated file: the `//go:linkname` pragma did not survive lowering); interpreted is D5 | v10.6a unexpected-success | moved 151 → 152 in 4525499 |
| blank | `gosource: package-level name T._ declared twice in the lowered file` (both modes; the converter's flat namespace) | v10.6a lowered-namespace | moved unclassified → 152 in 4525499 |
| escape_runtime_atomic | compiled `use of internal package internal/runtime/atomic not allowed`; interpreted retained (-m) | v10.6b internal-visibility | moved 151 → package in 4525499 (D8 set) |
| intrinsic | `use of internal package internal/runtime/sys not allowed` (both modes) | v10.6b internal-visibility | moved 152 → package in 4525499 (D8 set) |
| 15 `package:` roots with the same refusal | `could not import internal/… (use of internal package … not allowed)` | v10.6b annotated | already package; the D8 set is now visible as one set in `active-rules.tsv` (17 roots) |
| nul1 | `class=multiplicity`: the expected NUL diagnostic matched and a second one surfaced at the same position | v10.6c multiplicity-verdict | 154 by verdict (no move); the "position" reading in the story brief was stale at C |
| fixedbugs/issue47185 | `module package issue47185.dir/bad has non-Go inputs [bad.go]` — bad.go imports "C" | v10.6d cgo-root | 152 (no move) — D4, not D3(b); annotated |
| bug514, issue40954, issue42032, issue42076, issue46903, issue51733 (retained); issue34968, issue36705, issue47227, issue71225, issue71226 (152) | run roots importing "runtime/cgo" / "C" | v10.6d cgo-root | annotated, no move: 12 roots under v10.6d (D4 recorded 13 — the 13th is not a cgo declaration by source: see below) |
| package:cmd/compile/internal/ir | compiled `LOWER-ETYPE: "unsafe" imported and not used` | v10.6a lower-diagnostic on a `package:` root | stays package (every package: root is the package owner); annotated in `active-rules.tsv` for the lower owner |

Design decisions taken inside the rules (recorded so nobody re-derives them):

- **(d) keys on imports + recipe, not on `//go:build cgo`.** The brief's
  literal reading ("a root whose source carries `//go:build cgo`") would have
  moved bug513, issue20780b, issue29329 and issue36516 to retained: they
  carry the constraint for `-race` and never touch cgo — a partition-made
  reduction of the blocking count. An errorcheck root importing
  `runtime/cgo` (notinheap, notinheap2, notinheap3, typeparam/issue54765) is
  checked by the check interface like any package; its verdict is D5's
  (`compilation succeeded unexpectedly`), not cgo. So: cgo root := imports
  "C" anywhere in root/companions, or imports "runtime/cgo" under a recipe
  that executes the program. D4's "+1" (13 vs 12) is therefore not a root
  with a cgo declaration in its source — the manager's ledger should name
  it from the decision text, not from this rule.
- **(a) is root-level and testdir-only.** "Whatever owner it sits in today"
  means the compiled lowering row routes the root over the rank of the
  interpreted row (const8's interpreted `BASHPP-EEXPR-UNDEFINED: undefined:
  iota` still exists and is the evaluator's; the emitter row must be fixed
  first and the evaluator is not staffed on the root). `package:` roots stay
  with the package owner (v10.5), `ir` included.
- **(a) includes two shapes the brief did not list**: `compilation
  succeeded unexpectedly` in *compiled* mode (linkname3 — gc compiled the
  generated file and found nothing; interpreted unexpected success stays
  D5) and `gosource: … in the lowered file` (blank). Both are named in the
  plan's list of 16 and could not be reached by the four listed shapes.
- **The replay mode** (`partition-emit -manifests`) exists because the
  Barrier C evidence lives on the certification host (not this lane's
  venue). It replays the root-level rules over recorded rows and annotates
  (c)/(d). Run 0 (below) proves the evidence mode gives the same movement.

## Run 0 reconciliation (leaf-165r0, integ-3, harness 70b1072, 529 roots)

Run 0's evidence was read over ssh (read-only) and the v10.6 emitter run
over it in evidence mode with the corpus: **528 FAIL / 1 PASS**; owners
151 309 · 152 46 · 153 112 · 154 3 · package 28 · unclassified 30. The
v10.6 movement over real events is exactly the 20 roots of the Barrier C
replay (same names, same targets). Against Barrier C (minus the 30 deadline
roots that were not in the input) the only difference is
`fixedbugs/bug130.go`: unclassified at C (`bash++: task failed: gosource:
incomplete native selection reply`), PASS at run 0 — an intermittent bridge
reply row (Barrier C already listed it as a regression against B). Owner:
153 by first cause; it should not be filed as fixed until two consecutive
leaves pass it.

## Backend seam — generate-phase `.s` companions (D3(a))

| root | first cause (Barrier C) | mechanism | status |
| --- | --- | --- | --- |
| asmhdr, fixedbugs/issue22877, fixedbugs/issue37513, fixedbugs/issue47317, linknameasm, retjmp | `Bash++ backend unsupported generate phase: compile input "…/a.s" is not a Go source file` — builddir/buildrundir recipes (not compile/errorcheck/compiledir as the brief assumed): upstream hands the .s files to `go tool asm` twice around one direct `go tool compile -asmhdr -symabis` | assemble the companions natively with upstream's exact `go tool asm` argv as a recorded phase; transpile + direct-compile the Go files with -asmhdr/-symabis retained; pack/link/execute adopt the objects | seam fixed in e2d94bf; compiled verdict now the product's (leaf pending); interpreted row keeps the refusal (retained by the partition) |
| fixedbugs/issue15609, fixedbugs/issue74648 | `unsupported execute phase: module package … has non-Go inputs [a.s]` (runindir → cmd/go) | D3(b) | by ID, untouched (the fixture `asmrun.go` pins the refusal) |
| fixedbugs/issue47185 | same phase, but bad.go imports "C" | D4 | by ID (v10.6d), untouched |

Fixture set `testdata/backend/testdata/sprint165/asm-companion/` driven by
`asm-companion-gate.sh` through the exact patched runner (both modes, dev
host darwin/arm64; the amd64 twins run on the leaf): positive `asmdir`
(body-less declaration implemented in assembly, buildrundir), `asmonly`
(builddir), `asmhdr` (go_asm.h constant + struct layout of the generated
file — PASS: the lowering preserves names and layout), `plaindir` (Go-only
directory, both modes PASS); negative `asmmissing` (a declaration no
assembly implements → link `relocation target main.g not defined`, FAIL),
`asmrun` (D3(b) refusal unchanged). What the leaf will show for the six
corpus roots is the product's: linknameasm needs `//go:linkname` to survive
lowering (the linkname3 row, lower owner); issue37513 raises SIGILL on
purpose and compares output; asmhdr's constants passed on the fixture.

## Backend seam — the TestMain fact (D8)

- Today the cmd/go hook (`testdata/go-backend/bashpp_backend.go`,
  `bashppTestPlan`) passes `--go-import-path <pkg>.test` in both modes:
  `pmain` is cmd/go's generated testmain package (`load/test.go`:
  `ImportPath: p.ImportPath + ".test"`, `GoFiles: _testmain.go`), and the
  hook is invoked at the one patched site of `cmd/go/internal/test/test.go`
  where the built test binary would be executed. For the directory
  packages of a testdir root the testdir hook passes `--go-import-base
  <-D> --go-import-path <-p>` from upstream's own compile argv, and for
  runindir the main package's `go list` ImportPath.
- 1ee4c31 adds `--go-test-main` directly after `--go-import-path
  <pkg>.test` on the interpreted run and the compiled library transpile,
  records `import_path` / `test_main` on the plan event, and the package
  verifier requires both. **Spelling proposed to the gosource lane:
  `bashy --go-test-main` (boolean, valid only with `--go-import-path`) →
  `gosource.Options.TestMain`**, exactly §4.1's name. cmd/go's own rule the
  flag stands for: importer stack label `testmain` → `testing/internal/…`
  admitted; the tested `internal/…` package itself is admitted because the
  testmain's identity `<pkg>.test` has the parent-of-internal prefix.
- **Merge order (manager):** bashy must accept the flag before 1ee4c31 is
  pinned for a package leaf. Verified through the patched cmd/go on
  `cmd/compile/internal/abt` with the Barrier C bashy: the plan is right in
  both modes and bashy answers `transpile: unknown flag: --go-test-main`
  (interpreted: usage) — every package root would fail on the flag until
  then.

## Requests to other seams

- **gosource (S165.2, `gosource-vis` lane):** accept `--go-test-main` on
  `bashy --bashpp --source=go` (run/check) and `bashy transpile --bashpp
  --source=go` (library and program forms) → `Options.TestMain`; refuse it
  without `--go-import-path`. The backend never sets it for a testdir root.
- **lower (S165.4):** the 16 + 2 rows moved under v10.6a are in
  `docs/upstream-harness/barrier-c-v10.6/active-152-manifest.tsv`; `ir`'s
  row stays in the package manifest with its `v10.6a` annotation in
  `active-rules.tsv`. `linknameasm` (once the seam runs it) will surface the
  `//go:linkname` pass-through as its first line.
- **manager (by-ID ledger):** `active-rules.tsv` names the 17 D8 roots
  (v10.6b) and 12 cgo roots by source (v10.6d); D4's 13th needs a name from
  the decision text.
