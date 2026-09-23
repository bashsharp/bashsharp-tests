---
id: 7268c41c1a2d
kind: bug
title: S250 Windows GBE temporary paths normalize sandbox prefix
seq: 101
status: done
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T19:44:33.779076Z
assignee: codex-s250
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
closed: 2026-09-23T20:27:24.659292Z
closed_by: codex-s250
---

Final Windows GBE temporary-files-and-directories row69 produces identical source behavior and file contents in all modes, but existing tmp_path normalization replaces only sampleN/sampledirN leaf under backslash paths, leaving mode-specific absolute sandbox root in normalized stdout and effects. Extend only the declared tmp_path normalization for Windows drive-rooted tmp paths, preserving path labels, content SHA, statuses, other lines, corpus source and original 60s limit. Exact row69 plus independent validator and full 255 required.

## Final Windows acceptance, 2026-09-23

The final full Windows gate passed `examples/temporary-files-and-directories/temporary-files-and-directories.go` in all three modes with declared tmp path normalization and original 60-second limit; the earlier focused three-row replay passed 9/9.

Final Windows candidate: Bashy `e48bd7df29a496692f22ebb72520f7f32c932697`, sh `0736c52ec82d39922359548f0496e44ec583c6ba`, Yoke `61c589775457cb2d3cb4fab4288a4e5512ce1616`, binary SHA-256 `d758e09dfd0152b428e1d07db79a7285f4cbf5e43649468c0e07097875d5b8b5`. Gate-time tests `eda9a22856288ebc239cb177c2d206cf69d51da4`; reviewed evidence anchor `019193dd1996420b10fee0413e6fb94bf7c52a5f`. The retained `C:\Users\noviadmin\s250-327-final3\evidence\gbe-final3-green-255.jsonl.pass` has SHA-256 `7614c6adb597b2cb2a5024bc88ce5e00f158e8a18ede198584ab7574c872148f` and independently verified root `6867eacc716b234cd30f830d9ae01b93c7e9ea3c1b8b55d3a06aaa9ce36e154e`; verdict 255/255 PASS, zero missing. Original corpus, fixtures, and deadlines were retained.
