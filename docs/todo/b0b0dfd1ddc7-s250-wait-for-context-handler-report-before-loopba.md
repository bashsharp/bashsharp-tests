---
id: b0b0dfd1ddc7
kind: bug
title: S250 wait for context handler report before loopback termination
seq: 95
status: done
priority: p0
labels:
    - windows
    - go-by-example
    - adapter
created: 2026-09-23T18:29:26.375286Z
assignee: codex-s250
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
closed: 2026-09-23T20:27:23.813703Z
closed_by: codex-s250
---

Windows final GBE row14 examples/context/context.go: native/compiled stdout 30 bytes, interpreted empty, all terminated with 0xC000013A. In tools/go-by-example/gbe/gate.go, drive(context/) returns immediately after sending GET; the adapter then signals TERM before the interpreted HTTP handler can print. Make the context adapter observe the handler cancellation report before TERM, using the existing bounded deadline and original source/comparator. Add focused adapter control and rerun exact Windows row, then full affected gate. Preserve 20s row limit and fixture bytes.

## Diagnostic, 2026-09-23

Isolated patch `2cd869b` waits for the handler-start report before closing the client and for the handler-end report before signaling the server. The original source hash remains `7767ac607b8d097a1eff611b26968820b1d15bf243a39a1bd8ef8baa91d44182`; candidate manifest SHA-256 is `f5795676c1b7ee25752b351bce5585fea83273bd17d8d930a8b95bd5e00c82f3` (Bashy `f93816e`, sh `69eda1b`). On Windows, the focused adapter runtime test passed and an authenticated one-row diagnostic showed identical oracle/interpreted/compiled stdout: `server: hello handler started`, `server: context canceled`, `server: hello handler ended`. The row still fails effects because the Windows-terminated interpreted process retains its `.bashpp-eval-*` helper directory and `bashpp-session-*.go.bin` inside the observed temp root; that cleanup is tracked separately as sh Story #150. Diagnostic progress SHA-256: `c25d816bbdf71fffcb9c4d9620a465c42430da48b6720d613ef7c0533a4d01ad`. The one-row diagnostic intentionally reports missing attempts for the other 252 observations. Keep #95 open until the unchanged full row passes after #150.

## Final Windows acceptance, 2026-09-23

The final full Windows gate passed `examples/context/context.go` in all three modes with the original 20-second row limit. Adapter and temp-effect lifecycle controls are included in the retained gate evidence.

Final Windows candidate: Bashy `e48bd7df29a496692f22ebb72520f7f32c932697`, sh `0736c52ec82d39922359548f0496e44ec583c6ba`, Yoke `61c589775457cb2d3cb4fab4288a4e5512ce1616`, binary SHA-256 `d758e09dfd0152b428e1d07db79a7285f4cbf5e43649468c0e07097875d5b8b5`. Gate-time tests `eda9a22856288ebc239cb177c2d206cf69d51da4`; reviewed evidence anchor `019193dd1996420b10fee0413e6fb94bf7c52a5f`. The retained `C:\Users\noviadmin\s250-327-final3\evidence\gbe-final3-green-255.jsonl.pass` has SHA-256 `7614c6adb597b2cb2a5024bc88ce5e00f158e8a18ede198584ab7574c872148f` and independently verified root `6867eacc716b234cd30f830d9ae01b93c7e9ea3c1b8b55d3a06aaa9ce36e154e`; verdict 255/255 PASS, zero missing. Original corpus, fixtures, and deadlines were retained.
