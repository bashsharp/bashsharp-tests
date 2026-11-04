# Python fence corpus findings

Pinned fence runtime: CPython 3.15.0 (`v3.15.0`, commit
`d220409a13b77de7a6c30abe16ea90db01528edc`). The following are engine gaps,
not skipped or xfailed corpus cases.

## Tutorial transcript conversion

The increment started from 355 inventory rows (343 pinned tutorial examples
plus 12 boundary examples): 21 `case`, 225 `interactive`, 80 `fragment`, 7
`env-dependent`, 5 `prose`, 16
`missing-feature:python-class-declarations`, and 1 `diagnostic`.

It now has 252 `case`, 56 `fragment`, 36 `env-dependent`, 10 `prose`, and 1
`diagnostic`. There are no `interactive` or class-declaration missing-feature
rows. The 16 class-declaration examples were placed naturally inside generated
Python functions; the current engine accepted them, so there is no raw class
failure to record. The remaining environment-dependent rows name the concrete
unstable value (address, path/platform/build data, filesystem/network state,
unseeded randomness, date/locale/timing/thread scheduling, set order, garbage
collector state, or unscripted input).

The 11 grouped tutorial cases were run directly with `bashy --bashsharp` and
matched goldens produced by executing the exact generated fence body under the
pinned native CPython 3.15.0.

## `missing-feature:python-object-selectors`

The worker returns an object as a handle record, but the assigned Bash# local
has no typed selector path, so `h.read()` reports `BASHPP-ESELECTOR-TYPE`.

```bash
~~~py as py
import io
def make() -> io.StringIO: return io.StringIO("hello")
~~~
h, err := py.make()
v, err := h.read()
```

This also blocks iterator `next()` calls and callback handles crossing the
boundary (`missing-feature:python-iterators` and
`missing-feature:python-callbacks`). Python-internal method, iterator, and
callback use remains available inside a declared function.

## `missing-feature:python-typed-assignment-error`

Command mode correctly exits 1 for a bad typed call (`py.typed nope`). In
assignment mode, argument conversion currently prints an error plus
`assignment mismatch` but exits 0 and does not populate the opt-in error local:

```bash
~~~py as py
def typed(value: int) -> int: return value + 1
~~~
v, err := py.typed(nope)
```

The ordinary foreign-exception opt-in path is covered by
`boundary-error-opt-in`; the typed command failure is covered separately by
`boundary-typed-command-failure`.
