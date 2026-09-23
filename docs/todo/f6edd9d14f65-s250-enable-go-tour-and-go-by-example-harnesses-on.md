---
id: f6edd9d14f65
kind: bug
title: S250 enable Go Tour and Go by Example harnesses on Windows
seq: 88
status: done
priority: p0
created: 2026-09-23T09:43:52.909585Z
weave: 17
assignee: codex-s250
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
closed: 2026-09-23T20:27:14.836871Z
closed_by: codex-s250
---

On the Windows test host both Go harness Go packages fail to build because their POSIX syscall process/lock/FIFO paths are unguarded; wrappers also lack a Windows toolchain key and Go resolution, and accepted candidate/observation tables lack a Windows arm. Implement the minimum Windows runner equivalents and exact native Go 1.27.1 pins; record a real Windows baseline/candidate; run full Go Tour 291 and Go by Example 255 without fixtures, deadlines, exclusions or fallback changes. Private Sprint 250 Story #676 holds the raw evidence.

## Final Windows acceptance, 2026-09-23

Windows Go Tour completed 291/291 observations with independent validator PASS (semantic root `95c05e1d282769761a666d81d7c4efca85042468a7a56d9044de29f99fdf8b8e`; ledger SHA-256 `c6e07f66a2551b546c36a434dcb075ca68c35aee0d66a11cbd2332d35e27c84b`). Windows Go by Example completed 255/255, zero missing, independent evidence validator PASS.

Final Windows candidate: Bashy `e48bd7df29a496692f22ebb72520f7f32c932697`, sh `0736c52ec82d39922359548f0496e44ec583c6ba`, Yoke `61c589775457cb2d3cb4fab4288a4e5512ce1616`, binary SHA-256 `d758e09dfd0152b428e1d07db79a7285f4cbf5e43649468c0e07097875d5b8b5`. Gate-time tests `eda9a22856288ebc239cb177c2d206cf69d51da4`; reviewed evidence anchor `019193dd1996420b10fee0413e6fb94bf7c52a5f`. The retained `C:\Users\noviadmin\s250-327-final3\evidence\gbe-final3-green-255.jsonl.pass` has SHA-256 `7614c6adb597b2cb2a5024bc88ce5e00f158e8a18ede198584ab7574c872148f` and independently verified root `6867eacc716b234cd30f830d9ae01b93c7e9ea3c1b8b55d3a06aaa9ce36e154e`; verdict 255/255 PASS, zero missing. Original corpus, fixtures, and deadlines were retained.
