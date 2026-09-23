---
id: 953e38387470
kind: bug
title: S250 Windows GBE execing-processes native command parity
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

Final Windows GBE rows 23 execing-processes and 61 spawning-processes: the unchanged sources call exec.LookPath("ls") / syscall.Exec and exec.Command("date", "grep", "bash"). Current runEnv gives process_exec a Unix PATH string, while those tools are absent on that Windows path; oracle and compiled panic on lookup and interpreted produces different native-handle diagnostics. Diagnose the retained raw streams and distinguish missing deterministic command setup from sh product behavior. Repair only the proven Windows mechanism with a shared three-mode environment and exact semantic checks. Preserve source corpus, normalizer, and original limits; verify focused authenticated rows and full 255 gate.
