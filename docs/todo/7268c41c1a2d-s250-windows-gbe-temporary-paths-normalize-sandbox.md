---
id: 7268c41c1a2d
kind: bug
title: S250 Windows GBE temporary paths normalize sandbox prefix
seq: 101
status: todo
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T19:44:33.779076Z
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
---

Final Windows GBE temporary-files-and-directories row69 produces identical source behavior and file contents in all modes, but existing tmp_path normalization replaces only sampleN/sampledirN leaf under backslash paths, leaving mode-specific absolute sandbox root in normalized stdout and effects. Extend only the declared tmp_path normalization for Windows drive-rooted tmp paths, preserving path labels, content SHA, statuses, other lines, corpus source and original 60s limit. Exact row69 plus independent validator and full 255 required.
