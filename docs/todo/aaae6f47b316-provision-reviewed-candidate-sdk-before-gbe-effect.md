---
id: aaae6f47b316
kind: bug
title: Provision reviewed candidate SDK before GBE effect baseline
seq: 109
status: done
priority: p1
created: 2026-10-13T18:24:53.210918Z
assignee: muse-spark1.3
sprint: 434
sprint_id: f0fadef8-daec-5fc2-9c78-40979f01a0a7
sprint_title: bashy verified builds for binmgr tools and services (fork, build, sign, distribute)
closed: 2026-10-13T19:51:23.114465Z
closed_by: s434-muse-manager
---

S434 normal Go1.27.2 security-build gate is blocked: gate.go provisionsManagedToolCache only for Go1.27.1 oracle, then reviewed Go1.27.2 candidate materializes its SDK in BASHY_BIN_CACHE during first measured row, causing two input_mutation records. Fix harness preparation to provision both authenticated oracle SDK and separately reviewed candidate build SDK before input/effect baseline. Preserve fixed Go1.27.1 corpus/oracle, candidate authentication and checksum/source binding; reject unreviewed/forged SDK identity; do not ignore cache mutations or weaken measurement. Read runtimeconfig.go and candidate.go. Add focused regression tests proving security-build SDK setup before measurement and invalid identity rejection; reuse existing auth/provision mechanisms. Do not modify corpora, results, port behavior, candidate binaries, or unrelated docs. Worker-owned commit with this story trailers; no main push or parent pin. Short focused tests only on development machine, heavy/native gate later on claimed remote host. ACK manager s434-codex-manager; report concrete progress every3m; expected20min.

MANAGER REVIEW REQUIRED: preserve/reuse worker b68282f7caae1d56057eb13d455e465c370acb4a from existing weave issue-2; do not start over. Initial patch was NOT accepted: reviewedBuildToolchain hashes the discovered SDK itself and returns that digest as trusted GoSHA256. Identity/VERSION/native shape are self-asserted, not independent byte authentication. Bind SDK to reviewed exact digest or independently authenticated official release source. Keep Go1.27.1 oracle pin unchanged. Add meaningful negative for same-version native executable with unexpected bytes and positive real authenticated Go1.27.2 path. Do not weaken baseline/authentication. Commit corrected worker change under this same story; manager runs independent tests then normal merge. ACK this exact review concern, not original task only.
