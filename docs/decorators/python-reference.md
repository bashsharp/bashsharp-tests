# Python/CPython decorator semantic reference corpus — Sprint 197

Story: S197.4 `bb5ee9f84a12`. Design authority:
`../../../docs/bashpp-decorators-and-advice.md` (umbrella); delivery authority:
`../../../docs/sprint-197-master-execution-plan.md`.

This corpus is **reference behavior, not source ports**. It names the small,
deliberately bounded subset of Python's documented decorator semantics that
the Sprint 197 acceptance battery (`tests/decorators/`,
`tools/decorators/acceptance.rb`) translates into Bash++ fixtures, records the
upstream URL for each behavior, and states the **explicit dhnt differences**
so nobody later mistakes a divergence for a bug. Bash++ decorators are
*context-shaped* (`c *Call`), not Python's higher-order shape, and resolution
is **per-call and late-bound**, not definition-time — those two decisions
drive every difference below.

## Sources

| source | URL |
|---|---|
| Python language reference — function definitions (decorator grammar, `@f1(arg) @f2` ≡ `f1(arg)(f2(func))`, evaluation order) | https://docs.python.org/3/reference/compound_stmts.html#function-definitions |
| CPython decorator test suite (pinned tag) | https://github.com/python/cpython/blob/v3.13.0/Lib/test/test_decorators.py |
| PEP 318 — decorators for functions and methods | https://peps.python.org/pep-0318/ |
| PEP 614 — relaxed decorator grammar (arbitrary expressions; **out of scope** here) | https://peps.python.org/pep-0614/ |
| Umbrella's cited primer | https://www.geeksforgeeks.org/python/decorators-in-python |
| go-decorator — source of the `c *Call` skip semantics (`TargetDo`) | https://github.com/dengsgo/go-decorator |

## The two structural differences, once

1. **Contract.** Python: a decorator is any callable applied as
   `f = d(f)` when `def` executes — the function object is *replaced*.
   dhnt: a decorator is an ordinary Bash++ function whose first parameter is
   the predeclared `*Call`; the target is never replaced, its public
   signature is preserved, and the chain runs as ordinary frames around the
   body (`c.Next()`). Skipping `c.Next()` skips the body (go-decorator's
   "not calling `TargetDo`"), yielding zero results and whatever `Status`
   the decorator set.
2. **Timing.** Python evaluates the decorator expression and its arguments
   **once, at definition time** (`test_eval_order`), and unresolved names
   raise `NameError` while `def` executes (`test_errors`). dhnt registers the
   stack and resolves the decorator **at each call** (shell is late-bound;
   the native registry and advice rules may load after the declaration), and
   evaluates decorator arguments **per invocation** — for typed functions in
   the declaration's captured scope, for shell functions in the dynamic
   scope. Unresolved decorators are `EDECO-UNDEF` at first call, status 1.

## Behavior rows

Machine-readable copy: [`corpus.tsv`](corpus.tsv). Fixtures live in
`tests/decorators/`; all product expectations are **PLANNED** until the
S197.2 engine slice passes (`tests/manifest.tsv` is the status of record).

| behavior | Python reference | Python semantic | dhnt semantic / difference | fixture |
|---|---|---|---|---|
| nested stack order | `test_order`, `test_double`; language ref §function definitions | `@a @b def f` applies bottom-up, so `a` is outermost at call time | **same observable order** (nearest the declaration is innermost, stack executes outermost-first) — but as a `c *Call` frame chain, no object replacement | `stack-order.bpp` |
| decorator arguments | `test_argforms`, `test_eval_order` | argument expressions evaluated once, when `def` executes | positional/keyword/default binding is shipped Bash#; arguments are re-evaluated **per invocation**, so a changed variable is visible on the next call | `arguments.bpp` |
| dotted names | `test_dotted` | `@ns.attr` resolves an attribute chain at definition | `@ns.name(` parses (Class R) but is **reserved** — rejected at registration in the MVP | `dotted-reserved.bpp` |
| call counting / state | `countcalls` helper (used by `test_double`) | a stateful decorator observes every call | same observable counting; additionally the decorator may be declared **after** the target (late resolution) where Python raises `NameError` | `counting.bpp` |
| guard failure | `dbcheck` helper, `test_dbcheck` | a failed precondition raises `DbcheckError`; the body does not run | the guard **skips `c.Next()`**: body does not run, zero results (zero values on a typed call), `Status` is whatever the decorator set — no exception machinery | `guard-skip.bpp` |
| recursion | language ref — naming and binding (recursive calls resolve through the rebound module name) | every recursive level re-enters the wrapper | same observable: the chain re-enters on every level; decorator frames also consume `FUNCNEST` and appear in `FUNCNAME` (stated cost, pinned by `shell-functions.bpp`) | `recursion.bpp` |
| invalid / unresolved timing | `test_errors` | unresolved or ill-typed decorators fail while `def` executes | split by design: `EDECO-UNDEF` at first **call** (status 1); `EDECO-SELF` at **registration**; `EDECO-CYCLE` at call time before `FUNCNEST`; `EDECO-SIG` when resolution finds a first parameter that is not `*Call` | `undefined-timing.bpp`, `self-decoration.bpp`, `cycle.bpp`, `signature.bpp` |
| never-claimed near-misses | (no Python analogue — start-site law, umbrella F2) | — | bare `@t` and `@t()` followed by a compound command stay bash's own territory in every mode | `near-miss-shell.bpp` |

## Explicitly out of scope for the `c *Call` contract

Per the sprint plan: Python-only **object replacement**
(`test_single`/`staticmethod` semantics), **arbitrary decorator
expressions** (PEP 614), **classes and descriptors**
(`TestClassDecorators`, `@staticmethod`/`@classmethod`), **metadata
attributes** (`funcattrs`, `functools.wraps` — dhnt deliberately does *not*
hide decorator frames), and **memoization** (`test_memoize`; `@memo` is a
proposed next-wave native decorator and is never advisable).
