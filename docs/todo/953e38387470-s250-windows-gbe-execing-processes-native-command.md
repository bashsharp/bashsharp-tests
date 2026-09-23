---
id: 953e38387470
kind: bug
title: S250 Windows GBE spawning-processes command fixture
seq: 97
status: todo
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T18:29:46.354709Z
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
---

Final Windows GBE row 61 spawning-processes: the unchanged source invokes date, grep, and bash, while the current Windows process_exec PATH provides none of those commands. All three modes panic at the first date lookup, before the authored output is produced. Diagnose the retained streams and provide deterministic Windows command setup shared by all three modes, with authenticated bytes and exact semantic checks. Preserve source corpus, normalizer, and original limits; verify the focused authenticated row and full 255 gate. Row 23 syscall.Exec native Windows parity is tracked separately.
