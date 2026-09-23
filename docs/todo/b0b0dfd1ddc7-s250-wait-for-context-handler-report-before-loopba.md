---
id: b0b0dfd1ddc7
kind: bug
title: S250 wait for context handler report before loopback termination
seq: 95
status: todo
priority: p0
labels:
    - windows
    - go-by-example
    - adapter
created: 2026-09-23T18:29:26.375286Z
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
---

Windows final GBE row14 examples/context/context.go: native/compiled stdout 30 bytes, interpreted empty, all terminated with 0xC000013A. In tools/go-by-example/gbe/gate.go, drive(context/) returns immediately after sending GET; the adapter then signals TERM before the interpreted HTTP handler can print. Make the context adapter observe the handler cancellation report before TERM, using the existing bounded deadline and original source/comparator. Add focused adapter control and rerun exact Windows row, then full affected gate. Preserve 20s row limit and fixture bytes.
