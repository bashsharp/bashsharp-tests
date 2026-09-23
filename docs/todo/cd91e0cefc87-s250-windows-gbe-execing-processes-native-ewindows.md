---
id: cd91e0cefc87
kind: bug
title: S250 Windows GBE execing-processes native EWINDOWS parity
seq: 98
status: todo
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T18:36:51.468086Z
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
---

Final Windows GBE row 23 execing-processes: the unchanged source calls LookPath("ls") and syscall.Exec. Native Go on Windows implements syscall.Exec as EWINDOWS, so installing ls cannot make this row execute Unix replacement semantics. Retained oracle and compiled traces panic on missing ls, while interpreted emits a serialized native *os/exec.Error handle. Establish exact native Windows oracle semantics with focused controls, then repair only the proven product or harness error-parity mechanism. Preserve source corpus, normalizer licenses, and original limits; verify focused authenticated row and full 255 gate.
