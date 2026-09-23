---
id: 830e60d27c55
kind: bug
title: S250 normalize Windows GBE ls total metadata summary
seq: 99
status: todo
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T19:08:26.171358Z
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
---

Final Windows GBE spawning-processes row 61 now executes all three modes with pinned MinGit tools and exit 0. Raw ls -a -l -h output lists identical entries, but its total summary is 0 for oracle/compiled and 4.0K for interpreted; this is directory block metadata. The existing file_metadata normalizer covers detailed metadata fields but leaves the total summary byte-exact. Add only a strict total-line normalization under the row’s existing file_metadata license, retaining entry names, order, command output and all other lines. Keep original source, adapters and limits. Verify focused row 61, independent validator, and full 255 gate.
