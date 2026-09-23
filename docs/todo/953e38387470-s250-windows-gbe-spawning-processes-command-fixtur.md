---
id: 953e38387470
kind: bug
title: S250 Windows GBE spawning-processes command fixture
seq: 97
status: done
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T18:29:46.354709Z
assignee: codex-s250
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
closed: 2026-09-23T20:27:24.090257Z
closed_by: codex-s250
---

Final Windows GBE row 61 spawning-processes: the unchanged source invokes date, grep, and bash, while the current Windows process_exec PATH provides none of those commands. All three modes panic at the first date lookup, before the authored output is produced. Diagnose the retained streams and provide deterministic Windows command setup shared by all three modes, with authenticated bytes and exact semantic checks. Preserve source corpus, normalizer, and original limits; verify the focused authenticated row and full 255 gate. Row 23 syscall.Exec native Windows parity is tracked separately.

## Final Windows acceptance, 2026-09-23

The final full Windows gate passed `examples/spawning-processes/spawning-processes.go` in all three modes using the pinned MinGit process tools and original limits.

Final Windows candidate: Bashy `e48bd7df29a496692f22ebb72520f7f32c932697`, sh `0736c52ec82d39922359548f0496e44ec583c6ba`, Yoke `61c589775457cb2d3cb4fab4288a4e5512ce1616`, binary SHA-256 `d758e09dfd0152b428e1d07db79a7285f4cbf5e43649468c0e07097875d5b8b5`. Gate-time tests `eda9a22856288ebc239cb177c2d206cf69d51da4`; reviewed evidence anchor `019193dd1996420b10fee0413e6fb94bf7c52a5f`. The retained `C:\Users\noviadmin\s250-327-final3\evidence\gbe-final3-green-255.jsonl.pass` has SHA-256 `7614c6adb597b2cb2a5024bc88ce5e00f158e8a18ede198584ab7574c872148f` and independently verified root `6867eacc716b234cd30f830d9ae01b93c7e9ea3c1b8b55d3a06aaa9ce36e154e`; verdict 255/255 PASS, zero missing. Original corpus, fixtures, and deadlines were retained.
