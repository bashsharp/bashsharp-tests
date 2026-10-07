---
id: 3b50496741fd
kind: test
title: 'S381 S8 test prep: authenticate Linux ARM64 Go 1.27.1 candidate and run full harness'
seq: 107
status: done
priority: p0
created: 2026-10-07T04:59:30.841005Z
assignee: claude-fable5
sprint: 381
sprint_id: 1a8fa6b8-96d8-5f96-bcfa-d01ecbb8005c
sprint_title: 'Bash# as the agent action language: evidence of record, uncovered gaps, first paired experiment'
closed: 2026-10-07T09:46:18.111155Z
closed_by: codex-gpt6-sol
---

Provision Linux ARM64 Go 1.27.1 toolchain pin and reviewed candidate metadata for exact S8 runtime commits sh 836fc0bc9, bashy bf6cac8, ycode 84ed245. Run unchanged 255 Go-by-Example differential, Go oracle, and remaining Bash# harness inside private Podman network namespace; retain raw logs and report complete verdict. No skips or gate weakening.


Resume correction 2026-10-07, manager codex-gpt6-sol: Read /Users/qiangli/projects/poc/dhnt/docs/evidence/sprint-381-execution.json and current plan handoff FIRST. Exact S8 runtime already authenticated remotely, heavy fullgate passes255GBE+35oracle+97nativeTour baseline then fails historical tests/tour/evidence.jsonl manifest inventory binding mismatch. Preserve original historical ledger BYTES. Do not regenerate it with release0.20 or relabel historical results current. Diagnose historical inventory pin drift and prepare minimal honest validator/input-binding repair (legitimate bug fix permitted; no weakened checks/skips). Separate authenticated historical replay from fresh S8 validation. Independent ClaudeSonnet5.5 diagnosis active; read manager inbox at each turn.

Prior Codex run bashsharp-tests#2 stopped, salvage commit803851697fbde1fdf2c88ce4de9cf55346c33843 in /Users/qiangli/.bashy/weave/bashsharp-tests-ad201858/workspaces/issue-2 contains LinuxARM64 auth registries+native baseline artifacts worth reviewing/reusing. Remote ~/s381/isolated/harness-run has private experimental edits, including regenerated historical ledger, so restore its original ledger from committed local candidate only after archiving private experimental ledger separately. Do not trust its contents as original. All raw failures preserved. Frozen auth in ~/s381/isolated/auth; remote stopped s381-isolated-final container/native volume s381-harness-evidence and localhost/s381-harness:arm64 image. Remote novidesign.local is manager-claimed; use ONLY that host for heavy suites. Do not build heavy suites locally or alter original root/subprojects. Manager runs full final gate. Deliver isolated reviewed candidate, meaningful focused checks, exact remote gate command, no push. Need rootcause report within5min and bounded30min implementation. Do not dump JSONL; inspect manifest and summary only. Existing current runtime gates remain reusable, no new runtime changes.
