---
id: 346ff40e3dde
kind: bug
title: S250 macOS Go by Example interpreted signals timeout
seq: 90
status: todo
priority: p0
labels:
    - macos
    - harness
created: 2026-09-23T16:20:34.871045Z
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
---

Authenticated macOS novidesign full 255-attempt Go by Example gate on candidate 4f2d2d7a reports interpreted examples/signals/signals.go fail_incomplete(timeout), while oracle and compiled pass. Retain full run evidence, diagnose Bashy versus harness versus timing root cause under original source and deadline, fix narrowly, and rerun affected row/full gate.
