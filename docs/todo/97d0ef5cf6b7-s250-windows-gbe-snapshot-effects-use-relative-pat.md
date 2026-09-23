---
id: 97d0ef5cf6b7
kind: bug
title: S250 Windows GBE snapshot effects use relative paths
seq: 100
status: done
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T19:36:49.4937Z
assignee: codex-s250
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
closed: 2026-09-23T20:27:24.520022Z
closed_by: codex-s250
---

Final authenticated Windows GBE defer row writes identical tmp/defer.txt content in all modes, but effect deltas contain mode-specific absolute C:\\...\\run\\{mode}\\tmp\\defer.txt and interpreted/compiled fail_effects. corpus.go snapshot uses strings.TrimPrefix(path, root+"/"); filepath.WalkDir on Windows yields backslashes, so relative-key derivation fails. Replace with filepath.Rel and filepath.ToSlash, preserve all effect entries and strict digest comparison, verify exact unchanged row under original 60s plus full 255/independent validator. Do not change source corpus or deadline.

## Final Windows acceptance, 2026-09-23

The final full Windows gate passed both `examples/defer/defer.go` and `examples/writing-files/writing-files.go` in all three modes. Retained effects use relative `tmp/` keys and preserve each file content hash; the earlier focused three-row replay passed 9/9.

Final Windows candidate: Bashy `e48bd7df29a496692f22ebb72520f7f32c932697`, sh `0736c52ec82d39922359548f0496e44ec583c6ba`, Yoke `61c589775457cb2d3cb4fab4288a4e5512ce1616`, binary SHA-256 `d758e09dfd0152b428e1d07db79a7285f4cbf5e43649468c0e07097875d5b8b5`. Gate-time tests `eda9a22856288ebc239cb177c2d206cf69d51da4`; reviewed evidence anchor `019193dd1996420b10fee0413e6fb94bf7c52a5f`. The retained `C:\Users\noviadmin\s250-327-final3\evidence\gbe-final3-green-255.jsonl.pass` has SHA-256 `7614c6adb597b2cb2a5024bc88ce5e00f158e8a18ede198584ab7574c872148f` and independently verified root `6867eacc716b234cd30f830d9ae01b93c7e9ea3c1b8b55d3a06aaa9ce36e154e`; verdict 255/255 PASS, zero missing. Original corpus, fixtures, and deadlines were retained.
