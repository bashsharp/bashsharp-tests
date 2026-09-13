# Sprint 165 — lane harness-3 (S165.0: the go/types continuation fold)

Seam: `tools/upstream-harness/testdata/types-backend/gotypes/
bashpp_types_check_test.go` (the patched go/types `check_test.go` runner's
Bash++ hook), its unit test, `backend-pin.tsv` (`gotypes_hook`). Authority:
the exact upstream Go 1.27 `go/types/check_test.go`. Request: sh run #191
(`sh/gosource/testdata/sprint165/gosource-diag/FINDINGS.md` §1 (a), §4.1).

## Defect

`541f27c` (Sprint 154) made the go/types hook drop EVERY TAB-prefixed line
of the check-interface output as "gc's continuation of the previous error".
That is right for a POSITIONED continuation (`\t<file>:<l>:<c>: <msg>`, the
sub-error of a multi-part diagnostic): go/types reports it as the separate
secondary `Error` upstream `check_test.go` ignores (`": \t"`). It is lossy
for an UNPOSITIONED one (`\thave ()`, `\twant (int)`, `\tT does not
implement I (...)`, `\t\thave m[P any](P) P`): go/types joins a sub-error
without a position into the primary's own `Msg` (`errors.go` report) and
the fixtures' `ERROR` comments match against that joined message, so the
hook handed upstream a first-line-only `Msg` and the root failed with
`no error expected: "<first line only>"`. Product output was right
(`ErrorList.Error()` prints the joined message); the runner's reconstruction
was the defect.

## Fix (the #191 diff, verbatim in mechanism)

In the TAB branch: a line whose TAB-stripped remainder matches
`bashppDiagRx` (positioned) is dropped as before (unattributed only with no
primary to belong to); any other TAB line folds into the previous `Error`'s
`Msg` exactly as the pre-`541f27c` `m == nil` path did, or is unattributed
with no primary. Comparison is untouched: the upstream `": \t"` rule and
the ERROR-comment matching are unchanged, no expected output is normalized.

Unit tests (same package, run by `typechecker-gate.sh` under the overlay
with `-run '^TestBashppParse'`, all 8 PASS on go1.27.0; the two `Folds*`
tests FAIL against the `541f27c` hook):

- `TestBashppParseGotypesFoldsUnpositionedContinuation` — the #191 shape:
  have/want folded, a `": \t"` secondary and a gc-shaped `\t<pos>:` secondary
  both dropped, `unparsed == 0`.
- `TestBashppParseGotypesFoldsMixedContinuations` — mixed order: a
  method-signature detail (`\t\thave M(int)` / `\t\twant M(string)`) folds,
  a gc-shaped positioned secondary between two unpositioned lines is
  dropped without breaking the fold, a second primary starts a new `Msg`.
- `TestBashppParseGotypesLeadingUnpositionedContinuationIsUnparsed` — an
  unpositioned TAB line with no primary is unattributed, like the
  positioned leading case already pinned.

## Pin

`typechecker-gate.sh` authenticates the hook with `check_pin gotypes_hook`
(sha256 of `bashpp_types_check_test.go`) before anything runs; against the
unrefreshed pin the gate stops at `FAIL pin gotypes_hook: expected
cebaa697…, got 799242e8…`. `gotypes_hook` is refreshed to
`799242e89788aa44a92ceae08f72c14d58872b0098c7af61a6777e8712da39f9` (both
rows of the duplicated block); the gate then passes every pin check. The
types2 hook, patches, matrix and root list are unchanged.

## Rows this enables (per #191 §1; measured there on the patched runner
## with the fold at go/types 367/371 — NOT re-measured here)

Harness fold alone ((a) only), all `typechecker:go/types/`:
`TestFixedbugs/issue39634.go`, `TestFixedbugs/issue49005.go`,
`TestFixedbugs/issue50816.go`, `TestFixedbugs/issue54942.go`,
`TestFixedbugs/issue58742.go`, `TestFixedbugs/issue70150.go`,
`TestFixedbugs/issue70526.go`, `TestFixedbugs/issue76103.go`,
`TestSpec/methods.go` — 9 roots.

Harness fold + the #191 product candidate: `TestCheck/issues0.go` ((a) +
(b), needs sh `87da0fac`).

Harness fold clears only the have/want rows: `TestCheck/expr3.go` ((e)
3-index-slice columns remain — design, not this seam).

The types2 runner already joined continuations into its one-message shape
(`541f27c`) and has no row here. The corpus verdict for these rows is the
manager's measurement with the #191 candidate; this note claims none.
