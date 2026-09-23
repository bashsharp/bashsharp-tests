---
id: 830e60d27c55
kind: bug
title: S250 normalize Windows GBE ls total metadata summary
seq: 99
status: done
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T19:08:26.171358Z
assignee: codex-s250
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
closed: 2026-09-23T20:27:24.379647Z
closed_by: codex-s250
---

Final Windows GBE spawning-processes row 61 now executes all three modes with pinned MinGit tools and exit 0. Raw ls -a -l -h output lists identical entries, but its total summary is 0 for oracle/compiled and 4.0K for interpreted; this is directory block metadata. The existing file_metadata normalizer covers detailed metadata fields but leaves the total summary byte-exact. Add only a strict total-line normalization under the row’s existing file_metadata license, retaining entry names, order, command output and all other lines. Keep original source, adapters and limits. Verify focused row 61, independent validator, and full 255 gate.

## Final Windows acceptance, 2026-09-23

The final full Windows gate passed `examples/spawning-processes/spawning-processes.go` in all three modes, including the strict Windows `ls` metadata normalization.

Final Windows candidate: Bashy `e48bd7df29a496692f22ebb72520f7f32c932697`, sh `0736c52ec82d39922359548f0496e44ec583c6ba`, Yoke `61c589775457cb2d3cb4fab4288a4e5512ce1616`, binary SHA-256 `d758e09dfd0152b428e1d07db79a7285f4cbf5e43649468c0e07097875d5b8b5`. Gate-time tests `eda9a22856288ebc239cb177c2d206cf69d51da4`; reviewed evidence anchor `019193dd1996420b10fee0413e6fb94bf7c52a5f`. The retained `C:\Users\noviadmin\s250-327-final3\evidence\gbe-final3-green-255.jsonl.pass` has SHA-256 `7614c6adb597b2cb2a5024bc88ce5e00f158e8a18ede198584ab7574c872148f` and independently verified root `6867eacc716b234cd30f830d9ae01b93c7e9ea3c1b8b55d3a06aaa9ce36e154e`; verdict 255/255 PASS, zero missing. Original corpus, fixtures, and deadlines were retained.
