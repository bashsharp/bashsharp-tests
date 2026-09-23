---
id: 346ff40e3dde
kind: bug
title: S250 macOS Go by Example interpreted signals timeout
seq: 90
status: done
priority: p0
labels:
    - macos
    - harness
created: 2026-09-23T16:20:34.871045Z
assignee: codex-s250
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
closed: 2026-09-23T17:21:10.61076Z
closed_by: codex-s250
---

Authenticated macOS novidesign full 255-attempt Go by Example gate on candidate 4f2d2d7a reports interpreted examples/signals/signals.go fail_incomplete(timeout), while oracle and compiled pass. Retain full run evidence, diagnose Bashy versus harness versus timing root cause under original source and deadline, fix narrowly, and rerun affected row/full gate.
