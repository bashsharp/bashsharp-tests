---
id: f9c666fae020
kind: bug
title: S250 Windows Go by Example source map path separator mismatch
seq: 91
status: doing
priority: p0
labels:
    - windows
    - go-by-example
    - source-map
created: 2026-09-23T16:56:23.89947Z
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
---

Exact Windows arrays gate stages unchanged source at a mixed-separator path, then records that string as a source-map key. Bashy emits the native backslash spelling for the same source and validSourceMap compares keys byte-for-byte, so completed transpiles are rejected as invalid_source_map. In diagnostic rows, candidate 3a425685/source 9b23202e exits 0 at 54.8–57.3s with generated.go and map, yet compiled is unspawned. Canonicalize the staged input path before both the CLI argv and source records; preserve original source bytes, deadlines, and source-map content validation. Prove with exact Windows one-row, then full GBE 255 on final head. Distinct near-deadline transpile timing remains Story #327.

Focused proof: on noviwin1, the unchanged completed arrays artifacts at `C:\Users\noviadmin\s250-327\diag\arrays-pipe-lf.jsonl.work\000\compiled\transpile\generated.go{,.map}` passed the full `validSourceMap` digest/content/marker check when the record key used the map's canonical backslash source name, and failed with the gate's mixed-separator spelling. A diagnostic test executable is retained at `C:\Users\noviadmin\s250-327\diag\map-diag.test.exe`; its source was temporary and is not part of the delivery. The two-line harness fix canonicalizes both ordinary and generated test-driver input paths before argv and source records. `go test ./...` in the Go by Example harness passed on macOS. The exact Windows one-row still hit its unchanged 60-second transpile deadline in a separate run, so the gate remains red and Story #327 remains open for runtime diagnosis; do not claim #91 as a full-row pass yet.
