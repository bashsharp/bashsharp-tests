---
id: 97d0ef5cf6b7
kind: bug
title: S250 Windows GBE snapshot effects use relative paths
seq: 100
status: todo
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T19:36:49.4937Z
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
---

Final authenticated Windows GBE defer row writes identical tmp/defer.txt content in all modes, but effect deltas contain mode-specific absolute C:\\...\\run\\{mode}\\tmp\\defer.txt and interpreted/compiled fail_effects. corpus.go snapshot uses strings.TrimPrefix(path, root+"/"); filepath.WalkDir on Windows yields backslashes, so relative-key derivation fails. Replace with filepath.Rel and filepath.ToSlash, preserve all effect entries and strict digest comparison, verify exact unchanged row under original 60s plus full 255/independent validator. Do not change source corpus or deadline.
