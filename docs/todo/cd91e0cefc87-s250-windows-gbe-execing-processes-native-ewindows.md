---
id: cd91e0cefc87
kind: bug
title: S250 Windows GBE execing-processes native EWINDOWS parity
seq: 98
status: done
priority: p0
labels:
    - windows
    - go-by-example
created: 2026-09-23T18:36:51.468086Z
assignee: codex-s250
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
closed: 2026-09-23T20:27:24.227472Z
closed_by: codex-s250
---

Final Windows GBE row 23 execing-processes: the unchanged source calls LookPath("ls") and syscall.Exec. Native Go on Windows implements syscall.Exec as EWINDOWS, so installing ls cannot make this row execute Unix replacement semantics. Retained oracle and compiled traces panic on missing ls, while interpreted emits a serialized native *os/exec.Error handle. Establish exact native Windows oracle semantics with focused controls, then repair only the proven product or harness error-parity mechanism. Preserve source corpus, normalizer licenses, and original limits; verify focused authenticated row and full 255 gate.

2026-09-23 focused receipt: pinned MinGit tools make LookPath("ls") succeed. The unchanged source then reaches native Windows syscall.Exec, which returns EWINDOWS; oracle, interpreted, and compiled each panic with `not supported by windows` and exit 2. sh Stories #151 and #152 repair native error rendering and the scalar error-interface comparison. The tests harness applies the existing panic_trace normalization only to this Windows row, in both gate and independent validator, preserving panic body and exit status; authored classification.tsv remains unchanged. Focused normalizer controls pass on Go 1.27.1 and Windows cross-compilation succeeds.

The authenticated final candidate uses Bashy e48bd7d, sh 0736c52, and Yoke 61c5897 (Windows binary SHA-256 `d758e09dfd0152b428e1d07db79a7285f4cbf5e43649468c0e07097875d5b8b5`, manifest SHA-256 `99184a30ff9089c9d7d74dca2caed5d9a10a7d64337d6601691c3fce900666c3`). An exact bounded row-23 replay under the original 60-second limit gave oracle/interpreted/compiled `pass(complete,2)`; retained progress SHA-256 `37e142dd03b32764cfd1330c472276d2533183d492a6e036b258f63887c976a5`. Its diagnostic gate intentionally reports `missing-attempt evidence: expected 255 attempt records, got 3`; full 255 and independent validation remain the acceptance gate.

## Final Windows acceptance, 2026-09-23

The final full Windows gate passed `examples/execing-processes/execing-processes.go` in all three modes, each complete with native EWINDOWS panic exit 2 under the row-specific panic trace comparator. The earlier unchanged-source focused replay also passed 3/3.

Final Windows candidate: Bashy `e48bd7df29a496692f22ebb72520f7f32c932697`, sh `0736c52ec82d39922359548f0496e44ec583c6ba`, Yoke `61c589775457cb2d3cb4fab4288a4e5512ce1616`, binary SHA-256 `d758e09dfd0152b428e1d07db79a7285f4cbf5e43649468c0e07097875d5b8b5`. Gate-time tests `eda9a22856288ebc239cb177c2d206cf69d51da4`; reviewed evidence anchor `019193dd1996420b10fee0413e6fb94bf7c52a5f`. The retained `C:\Users\noviadmin\s250-327-final3\evidence\gbe-final3-green-255.jsonl.pass` has SHA-256 `7614c6adb597b2cb2a5024bc88ce5e00f158e8a18ede198584ab7574c872148f` and independently verified root `6867eacc716b234cd30f830d9ae01b93c7e9ea3c1b8b55d3a06aaa9ce36e154e`; verdict 255/255 PASS, zero missing. Original corpus, fixtures, and deadlines were retained.
