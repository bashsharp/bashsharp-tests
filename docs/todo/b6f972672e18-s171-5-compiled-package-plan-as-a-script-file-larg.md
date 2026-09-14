---
id: b6f972672e18
kind: task
title: 'S171.5 compiled package plan as a script file: large packages exceed the argument limit'
seq: 77
status: done
priority: p1
created: 2026-09-14T14:28:35.962644Z
assignee: spandrel
sprint: 171
closed: 2026-09-14T14:28:59.811592Z
closed_by: spandrel
---

Exposed by leaf integ-171-r1: package:cmd/compile/internal/ssa, types2 and go/types compiled = 'fork/exec /bin/sh: argument list too long' - the hook passed the whole overlay script (every original and generated file path) as one /bin/sh -c argument, exceeding the kernel's single-argument limit before Bash++ ran. Fix: the hook writes the script to <overlay>/plan.sh in the persisted overlay directory and hands /bin/sh <path> (record.script names it); package-verify reads the script from either form through compiledScript() so every identity/role rule still checks the text the shell ran, with negatives for a mismatched or missing file. Driving: TestVerifyIdentityReadsTheScriptFile + package-identity-gate.sh through the exact patched cmd/go (PASS, all three fixtures compiled PASS). Pins refreshed.
