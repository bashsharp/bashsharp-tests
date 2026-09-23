---
id: f6edd9d14f65
kind: bug
title: S250 enable Go Tour and Go by Example harnesses on Windows
seq: 88
status: todo
priority: p0
created: 2026-09-23T09:43:52.909585Z
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
---

On noviwin1 both Go harness Go packages fail to build because their POSIX syscall process/lock/FIFO paths are unguarded; wrappers also lack a Windows toolchain key and Go resolution, and accepted candidate/observation tables lack a Windows arm. Implement the minimum Windows runner equivalents and exact native Go 1.27.1 pins; record a real Windows baseline/candidate; run full Go Tour 291 and Go by Example 255 without fixtures, deadlines, exclusions or fallback changes. Evidence: dhnt docs/sprint-250-story-676-noviwin1-evidence.md. Sprint 250 Story #676.
