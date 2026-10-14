---
id: 1c290bd38ff3
kind: task
title: Authenticate candidate SDK against independently reviewed bytes before GBE baseline
seq: 110
status: done
priority: p1
created: 2026-10-13T18:35:56.524839Z
weave: 3
assignee: codex-gpt5.6-sol
sprint: 434
sprint_id: f0fadef8-daec-5fc2-9c78-40979f01a0a7
sprint_title: bashy verified builds for binmgr tools and services (fork, build, sign, distribute)
closed: 2026-10-13T18:51:56.602324Z
closed_by: s434-codex-manager
---

Follow-up to aaae6f47b316 / weave 2. Reuse preserved worker commit b68282f7caae1d56057eb13d455e465c370acb4a from local workspace weave run 2 workspace; do not discard unique work. Review rejected self-pinning: discovered SDK hash plus version/native checks do not authenticate source. Compare candidate build SDK against independently reviewed checksum or authenticated official distribution before executing/linking it; fixed Go1.27.1 oracle remains unchanged. Add meaningful negative regression for same-version native-shaped wrong bytes, retain dual pre-baseline provisioning and cache binding. Short local unit tests only; no remote host or heavy gate. Keep private paths/hostnames out of public committed docs. Commit as actual worker with new story trailers; do not push or mutate main. Report exact trusted source and checksum provenance, commit and tests to s434-codex-manager. Initial review window 20m; updates every 3m. This is implementation follow-up, not permission to claim full Go conformance.
