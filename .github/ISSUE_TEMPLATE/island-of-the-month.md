---
name: "Fenced language of the month"
about: "Add one fenced-language island to Bash# — a bounded feature an outsider can own end to end"
title: "island: <language> — <one real repo it calls>"
labels: island, help wanted
---

## The language

Which language, and which toolchain bashy should find on `PATH` (a compiler
or interpreter you can name exactly, with its version).

## The island

A tilde-fenced block with the language name (`~~~zig as z`) whose top-level
functions become callables from shell text, with typed values crossing the
boundary — the same shape as `04-islands/*.bsh` in the tour. Say which
argument and return types you will support first (integers, strings, and
one more is enough).

## The real repo

One public repository in that language, pinned to a commit, that a `dag.md`
under `bashy/examples/dag/<repo>/` builds and calls through the island — the
pattern every existing island follows. The example is the acceptance test.

## Done when

- `bashsharp-tests/tests/polyglot/<language>/` has a fixture with a pinned transcript for the interpreted mode, and lowering parity where it applies.
- The tour gets a `04-islands/<language>.bsh` that SKIPs by name when the toolchain is absent.
- The classic gate is unchanged.
