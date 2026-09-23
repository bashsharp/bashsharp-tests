---
id: f9c666fae020
kind: bug
title: S250 Windows Go by Example source map path separator mismatch
seq: 91
status: done
priority: p0
labels:
    - windows
    - go-by-example
    - source-map
created: 2026-09-23T16:56:23.89947Z
assignee: codex-s250
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
closed: 2026-09-23T20:27:23.677274Z
closed_by: codex-s250
---

Exact Windows arrays gate stages unchanged source at a mixed-separator path, then records that string as a source-map key. Bashy emits the native backslash spelling for the same source and validSourceMap compares keys byte-for-byte, so completed transpiles are rejected as invalid_source_map. In diagnostic rows, candidate 3a425685/source 9b23202e exits 0 at 54.8–57.3s with generated.go and map, yet compiled is unspawned. Canonicalize the staged input path before both the CLI argv and source records; preserve original source bytes, deadlines, and source-map content validation. Prove with exact Windows one-row, then full GBE 255 on final head. Distinct near-deadline transpile timing remains Story #327.

Focused proof: on noviwin1, the unchanged completed arrays artifacts at `C:\Users\noviadmin\s250-327\diag\arrays-pipe-lf.jsonl.work\000\compiled\transpile\generated.go{,.map}` passed the full `validSourceMap` digest/content/marker check when the record key used the map's canonical backslash source name, and failed with the gate's mixed-separator spelling. A diagnostic test executable is retained at `C:\Users\noviadmin\s250-327\diag\map-diag.test.exe`; its source was temporary and is not part of the delivery. The two-line harness fix canonicalizes both ordinary and generated test-driver input paths before argv and source records. `go test ./...` in the Go by Example harness passed on macOS. The exact Windows one-row still hit its unchanged 60-second transpile deadline in a separate run, so the gate remains red and Story #327 remains open for runtime diagnosis; do not claim #91 as a full-row pass yet.

## Final Windows acceptance, 2026-09-23

The unchanged Windows `arrays` row passed oracle, interpreted, and compiled at the original 60-second limit. The final full 255-attempt gate passed and its source-map records validated against the authenticated public candidate.

Final Windows candidate: Bashy `e48bd7df29a496692f22ebb72520f7f32c932697`, sh `0736c52ec82d39922359548f0496e44ec583c6ba`, Yoke `61c589775457cb2d3cb4fab4288a4e5512ce1616`, binary SHA-256 `d758e09dfd0152b428e1d07db79a7285f4cbf5e43649468c0e07097875d5b8b5`. Gate-time tests `eda9a22856288ebc239cb177c2d206cf69d51da4`; reviewed evidence anchor `019193dd1996420b10fee0413e6fb94bf7c52a5f`. The retained `C:\Users\noviadmin\s250-327-final3\evidence\gbe-final3-green-255.jsonl.pass` has SHA-256 `7614c6adb597b2cb2a5024bc88ce5e00f158e8a18ede198584ab7574c872148f` and independently verified root `6867eacc716b234cd30f830d9ae01b93c7e9ea3c1b8b55d3a06aaa9ce36e154e`; verdict 255/255 PASS, zero missing. Original corpus, fixtures, and deadlines were retained.
