# Sprint 562 / Story 3047: run-scoped provisioning

Changed only this runner directory: `adapter.go`, new `toolchains.go`,
`build.sh`, `selftest.sh`, `testdata/runner.sh`, README and this evidence.
No upstream examples or dispositions added by this infrastructure change.
The committed synthetic inventory remains one source, one case, disposition
`case`; selftest creates temporary mutations, including a two-case run.

Verification on darwin/arm64:

- `sh tools/fence-corpus/selftest.sh`: all positive and negative assertions PASS.
  New assertions cover first-install stderr isolation across two cases,
  read-only managed files, reuse, fatal provisioning errors, ambient variable
  isolation, rc plus both stream diagnostics, and bounded excerpts.
- Temporarily fetched issue-19 `agent/weave-issue-19` at
  `ecd73ea085fc82ea3b720966fad2dc2786dc0123`, then merged with
  `git merge --no-commit --no-ff FETCH_HEAD` for the requested check.
- Ran the exact clean-environment command:

  ```sh
  env -i HOME=$HOME PATH=/usr/bin:/bin:$(dirname $(command -v bashy)) BASHY_BIN=$(command -v bashy) sh tools/fence-corpus/check.sh --corpus python-by-example
  ```

  Result: **22 cases, 22 PASS, 0 failures**. A fresh run-scoped cache downloaded
  CPython 3.15 once; all download/install warnings were in `provision.log`.
  Each case used a fresh HOME/work/tmp and offline toolchain settings.
  This explicit requested check is the only full corpus run performed.
- Removed the test merge with `git merge --abort`; no issue-19 content is
  included in this delivery.
- Clean-environment `sh tools/fence-corpus/build.sh` also passed after the
  wrapper correction; the underlying SDK digest verification is unchanged.

The tested issue-19 inventory has 343 upstream examples plus 12 boundary
references (355 rows): case 21, diagnostic 1, prose 5, fragment 80,
interactive 225, env-dependent 7, missing-feature:python-class-declarations 16.
Those classifications belong to issue-19 and are unchanged here.

Findings: uv requires a writable scratch cache even for `python find`, so
cases share the read-only managed interpreter directory but receive separate
uv scratch caches. The inherited build resolver incorrectly chose `go.exe`
when only bashy was on PATH on macOS; the generated wrapper now selects the
platform spelling. No shared engine changes or weakened assertions were needed.
Other language provisioning probes follow the candidate's fence resolvers;
only Python was exercised against a real corpus in this bounded local check.
