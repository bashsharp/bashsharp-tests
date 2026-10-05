---
id: dd8b7332d053
kind: bug
title: S358.15 Restore polyglot gate lazy Python invariant under provisioning
seq: 106
status: assigned
priority: p1
labels:
    - bashsharp
created: 2026-10-05T05:40:26.990206Z
weave: 1
assignee: codex-gpt6-sol
sprint: 358
sprint_id: 59941ed8-d6fb-50fb-8ad3-4ffb362c155e
sprint_title: 'PowerShell and C# fences: Windows users at home on every OS'
---

On frozen bashy 536e825, existing bashsharp-tests/tools/polyglot-gate.sh fails at line 306: PATH=/nonexistent Bashy --bashpp direct Python fence succeeds, so `missing Python unexpectedly succeeded`. Reproduces on current bashsharp-tests main before S358.13 changes; Bashy now provisions a cached/pinned Python runtime. Update the gate to test the intended lazy-runtime invariant in a deterministic isolated environment using bashy-supported toolchain/cache controls, or change the assertion to match current product semantics while retaining meaningful coverage. Do not weaken new PowerShell/C# cases. Run full polyglot-gate.sh and startsites --check; submit for review. S358.10 final gate depends on this.
