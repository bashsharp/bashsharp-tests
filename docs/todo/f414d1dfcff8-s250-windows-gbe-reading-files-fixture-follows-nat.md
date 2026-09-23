---
id: f414d1dfcff8
kind: bug
title: S250 Windows GBE reading-files fixture follows native temp path
seq: 96
status: todo
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T18:29:40.001384Z
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
---

Final Windows GBE row 51 reading-files: harness stages pinned dat bytes under per-mode root/tmp and sets TMPDIR only, but the pinned Go Windows os.TempDir resolves C:\Windows, so oracle, interpreted, and compiled all attempt C:\Windows\dat and panic. Diagnose exact retained row and make the common Windows run environment point native TEMP/TMP at the existing staged root/tmp for all three modes. Preserve example source, fixture bytes, normalizer, and original limits. Verify a focused authenticated row with output/effects parity and full 255 gate before closure.
