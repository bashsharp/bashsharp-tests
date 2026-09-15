# Sprint 183 unchanged Python packages

`tools/python-package-gate.sh` runs these fixtures against two optional local
checkouts without copying or changing their source:

- nanochat at `dc54a1a3077cab11d68fac4c5d1cd5c51f5d8c7a`;
- mini-SWE-agent at `04d809ceab9df28f9adaed044884180159172930`.

An absent checkout is reported as `SKIP`; a present checkout at another commit
fails. Sprint closure must run both, so its `OK` tally must be `2/2`.

The `.bpp` files cover the public Bash++ surface: direct package imports,
positional arguments, keyword arguments, object attributes, `__name__`, an
exception-bearing nanochat result, and a missing attribute diagnostic. The Go
fixture uses the same `sh/polyglot` runtime through its public object API to
close and restart the worker, then proves the old nanochat result handle is
rejected. It exists at the Go layer because Sprint 183 defines no source-level
handle-release syntax.
