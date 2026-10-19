---
id: f414d1dfcff8
kind: bug
title: S250 Windows GBE reading-files fixture follows native temp path
seq: 96
status: done
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T18:29:40.001384Z
assignee: codex-s250
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
closed: 2026-09-23T20:27:23.953309Z
closed_by: codex-s250
---

Final Windows GBE row 51 reading-files: harness stages pinned dat bytes under per-mode root/tmp and sets TMPDIR only, but the pinned Go Windows os.TempDir resolves C:\Windows, so oracle, interpreted, and compiled all attempt C:\Windows\dat and panic. Diagnose exact retained row and make the common Windows run environment point native TEMP/TMP at the existing staged root/tmp for all three modes. Preserve example source, fixture bytes, normalizer, and original limits. Verify a focused authenticated row with output/effects parity and full 255 gate before closure.

## Final Windows acceptance, 2026-09-23

The final full Windows gate passed `examples/reading-files/reading-files.go` in all three modes with the original source and limits.

Final Windows candidate: Bashy `e48bd7df29a496692f22ebb72520f7f32c932697`, sh `0736c52ec82d39922359548f0496e44ec583c6ba`, Yoke `61c589775457cb2d3cb4fab4288a4e5512ce1616`, binary SHA-256 `d758e09dfd0152b428e1d07db79a7285f4cbf5e43649468c0e07097875d5b8b5`. Gate-time tests `eda9a22856288ebc239cb177c2d206cf69d51da4`; reviewed evidence anchor `019193dd1996420b10fee0413e6fb94bf7c52a5f`. The retained `<home>\s250-327-final3\evidence\gbe-final3-green-255.jsonl.pass` has SHA-256 `7614c6adb597b2cb2a5024bc88ce5e00f158e8a18ede198584ab7574c872148f` and independently verified root `6867eacc716b234cd30f830d9ae01b93c7e9ea3c1b8b55d3a06aaa9ce36e154e`; verdict 255/255 PASS, zero missing. Original corpus, fixtures, and deadlines were retained.
