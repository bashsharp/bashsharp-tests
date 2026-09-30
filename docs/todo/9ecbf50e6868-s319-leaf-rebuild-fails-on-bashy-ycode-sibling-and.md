---
id: 9ecbf50e6868
kind: bug
title: S319 leaf rebuild fails on Bashy ycode sibling and lacks candidate hash guard
seq: 104
status: todo
priority: p0
created: 2026-09-30T11:02:04.562974Z
sprint: 319
sprint_id: 8a7c3136-c235-5e11-933f-2cef714109b6
sprint_title: 'Bash# interpreted compiler packages: residual roots after Sprint 281 (R45)'
---

Current bashsharp-tests/tools/upstream-harness/rebuild-candidate.sh clones seven siblings but Bashy go.mod now replaces ../ycode and ../ycode/examples/genie. Fresh Linux rebuild of sh 4324d9ba failed because those directories are absent. Also leaf-run accepts an old Bash# binary after sh checkout changes (R8-R11 evidence defect). Update the rebuild/leaf harness to hydrate ycode and fail closed on interpreter binary identity; prove an old binary cannot be credited. Sprint 319 shared baseline uses a guarded, recorded candidate while this is repaired.
