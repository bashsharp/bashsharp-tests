---
id: 30f81908446b
kind: bug
title: Separate authenticated build toolchain from fixed Go1.27.1 corpus oracle
seq: 108
status: done
priority: p1
created: 2026-10-13T07:40:19.154178Z
weave: 1
assignee: s434-contract
sprint: 434
sprint_id: f0fadef8-daec-5fc2-9c78-40979f01a0a7
sprint_title: bashy verified builds for binmgr tools and services (fork, build, sign, distribute)
closed: 2026-10-13T07:50:14.437207Z
resolution: fixed
closed_by: s434-contract-a
---

Operator decision: build/ship go1.27.2, conformance coordinate remains Go1.27.1 corpus. Current tools/go-by-example/gbe/candidate.go authenticateManifest rejects row go_identity != toolchain.tsv oracle identity; f21052e1 required normal gate cannot start. Implement narrow explicit reviewed build/oracle coordinate separation (not permissive mismatch bypass). Keep corpus/oracle table and full corpus coordinate unchanged. Authenticate candidate build Go identity via reviewed candidate metadata and real binary/build recipe; preserve manifest/launcher/payload hashes, exact clean repository revisions, review allowlist and runtime checks. Reject unreviewed build identities or forged/stale metadata. Add meaningful positive security-patch build against unchanged oracle test and tamper/mismatch negative coverage using existing gate tests. Document semantics. Coordinate with verification worker codex-gpt5.6-sol-w21, a remote macOS host exclusively its gate; only short unit tests on the development machine. No full barrier unless semantic delta. No Claude/no containers on the development machine. Read local CLAUDE/AGENTS. Commit with Sprint434/Story/Story-ID and send exact SHA + tests to manager s434-codex-manager for independent review, no main push.
