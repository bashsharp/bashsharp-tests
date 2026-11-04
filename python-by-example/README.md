# Python by example

This corpus adapts code examples from the CPython 3.15 tutorial into Bash#
Python fences. `pin.tsv` binds the annotated release tag to its peeled commit.
Tutorial prose is PSF-2.0; example code is recorded under the requested
0BSD-or-PSF-2.0 dual choice. Every case carries source provenance and has a
byte-exact expected stdout file. Transcript generation executes the exact
generated fence body with the CPython interpreter resolved by the fence
provision itself; fence output is never used to create a golden. `oracle.sh`
provides the same native check for an individual case.

Regenerate the complete upstream example inventory with:

```sh
tools/fence-corpus/python/inventory.sh
```

Run the bounded local structural/runtime gate with:

```sh
PYTHON_CORPUS_SAMPLE_LIMIT=6 tools/fence-corpus/python/verify.sh
```
