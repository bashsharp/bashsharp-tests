---
id: 1bd89e54e114
kind: bug
title: Bound retained per-case compiler caches in corpus diagnostic evidence
seq: 115
status: todo
priority: p0
created: 2026-11-06T02:30:50.368153Z
assignee: opencode-glm-5.3
sprint: 578
sprint_id: 70ba48be-af89-5ab1-8a8d-42a15fc1ddbe
sprint_title: 'S562 follow-up: remaining fence corpus work'
---

Source inspection found tools/fence-corpus/adapter.go retains full work directories for compiler diagnostic, semantic and property-table evidence, while runtime_cache.go copies the entire authenticated runtime-gocache into each work/cache. Ordinary cases clean their work. Repeated evidence cases can therefore multiply disposable compiler-cache bytes; this is a concrete mechanism, not yet proof of the old guest574GB cause. Required test environment ceiling20GB, initial8GB. Measure a representative case and retain raw stdout/stderr/status, source/provider/binary hashes and all required proof artifacts. After final receipt and unchanged-input verification, reclaim only proven disposable per-case build caches; never share candidate-writable caches or discard referenced native/worker evidence. Include independently checked cleanup and space accounting, preserve deadlines/modes/oracles/inventory, and verify representative diagnostic/semantic red/green plus later full retained corpora. No change to product semantics or blanket evidence deletion.

Carried from S562 (operator-approved early close 2026-11-06); deferral is not PASS. Run/branch/evidence references: weave run 54.
