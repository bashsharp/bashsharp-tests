# Sprint 197 decorator acceptance fixtures

Two stories own files in this directory:

- **Layout fixtures** (`call-forms.bpp`, `declaration-layout.bpp`,
  `compound-bodies.bpp`) — S197.1 `515da1278005`. Graded only by the
  harness's generic fixture loop; not listed in `cases.tsv`.
- **Acceptance battery** (everything in `cases.tsv`, with `.out` stdout
  sidecars) — S197.4 `bb5ee9f84a12`. Graded by
  `ruby tools/decorators/acceptance.rb` from the repository root, with
  `BASHY_BIN` pointing at the product binary and GNU Bash 5.3 on `PATH` (or
  `BASH53=/path/to/bash`).

The battery translates a deliberately small subset of Python's documented
decorator semantics and CPython's `Lib/test/test_decorators.py` — reference
behaviors, not source ports. The corpus, upstream URLs, and the explicit
dhnt differences (per-call late resolution; the `c *Call` context contract
instead of Python's higher-order shape) are recorded in
[`../../docs/decorators/python-reference.md`](../../docs/decorators/python-reference.md)
and `../../docs/decorators/corpus.tsv`.

## Honesty contract

`tests/manifest.tsv` is the status of record. Every fixture here except
`near-miss-shell.bpp` is **planned**: the S197.2 engine slice has not merged,
so the runner executes the product matrix, reports those cases as
**PLANNED**, and claims no decorator product evidence. A planned case that
starts passing is printed as a ratchet candidate — flipping it to
`supported` is the sprint manager's move at acceptance, never the runner's.

What **is** enforced today, fatally, on every case and entry mode:

- **Dialect-off isolation.** Classic (`--no-bashpp`), POSIX
  (`--posix --no-bashpp`) and the Classic front door
  (`--posix --bashpp`) parse verdicts must match independent GNU Bash 5.3
  (the combined profile uses Classic GNU semantics, matching the product
  resolver). Decorator grammar must be invisible with the dialect off.
- **Never-claimed shapes.** `near-miss-shell.bpp` (bare `@t`; `@t()` +
  compound command) is executed in every mode and stream-compared with the
  GNU oracle; it is `supported` because it must pass on today's product.

## Matrix

Each ledger case runs through **file, stdin and `-c`** entry. Product mode
(`--bashpp`) compares exit status, exact stdout against the `.out` sidecar,
and a declared stderr class (`empty`, or containment of the design's
diagnostic codes `EDECO-UNDEF`, `EDECO-SELF`, `EDECO-CYCLE`, `EDECO-SIG`,
and the reserved dotted-name rejection). Every subprocess is bounded at 15
seconds with process-group kill. `BASHY_HINTS=off` and `BASHY_ADVICE` unset
keep stderr deterministic and the battery advice-free.

Coverage per the sprint plan: nested stack order, decorator arguments
(positional/keyword/default + per-invocation evaluation), dotted-name
reservation, call counting/state with late resolution, guard failure
(skip/zero results/Status), recursion re-entering the chain, error timing
(undefined at call, self at registration, cycle at call before FUNCNEST,
non-`*Call` signature), FUNCNAME frame visibility and shell-function forms,
and the dialect-off/near-miss matrix. Python-only object replacement,
arbitrary expressions, classes/descriptors, metadata attributes, and
memoization are out of scope for the `c *Call` contract.
