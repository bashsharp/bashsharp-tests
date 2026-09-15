---
id: 3e1347caa870
kind: task
title: 'S184.3: gate npm pnpm Bun and Node TypeScript workspaces'
seq: 82
status: todo
priority: p0
created: 2026-09-15T05:28:43.250295Z
sprint: 184
---

After sh S184.2 merges, add `tools/sprint184-typescript-gate.sh` and
minimal fixtures. Build/use the sibling bashy binary. Prove pre-materialized
npm and pnpm workspace layouts import from a TypeScript fence under Node; prove
a Bun-managed layout imports under explicit Bun; prove manager metadata never
overrides explicit runtime and no install/lifecycle command is invoked.

Against unchanged `ycode/priorart/opencode` commit
`e03db9bc6908f75c9334d8aa997deeaac81c0298` with
`BASHPP_BUN=/Users/qiangli/.bun/bin/bun`, import
`opencode/config/paths` inside a fence, await the exported wrapper, and assert
`/tmp/x.json` plus `/tmp/x.jsonc`. Run the unchanged CLI entry as a disposable
Bun process with `--version` and assert `local`. Record before/after fixture
status. Missing OpenCode or Bun is failure.

Preserve `tools/polyglot-gate.sh`, the Python package gate, and
start-site/POSIX checks. Do not install packages, modify the fixture, launch the
TUI, use credentials/network, add direct import syntax, or add a framework.
Stage named files only and commit with:

```text
Sprint: #184
Story: #82
Story-ID: 3e1347caa870
```
