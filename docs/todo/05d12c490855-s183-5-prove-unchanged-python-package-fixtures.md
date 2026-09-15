---
id: 05d12c490855
kind: task
title: S183.5 — prove unchanged Python package fixtures
seq: 80
status: done
priority: p1
created: 2026-09-15T00:07:55.463274Z
assignee: codex-gpt5.6-sol
sprint: 183
closed: 2026-09-15T02:27:47.916026Z
closed_by: codex-gpt5.6-sol
---

Add the smallest conformance gate and fixtures for direct imports against unchanged `/Users/qiangli/projects/poc/nanochat` at dc54a1a3077cab11d68fac4c5d1cd5c51f5d8c7a and `/Users/qiangli/projects/poc/mini-swe-agent` at 04d809ceab9df28f9adaed044884180159172930. Prove nanochat.execution.execute_code success/stdout plus kwargs/errors/stale handles, and minisweagent.agents.get_agent_class("default").__name__ without network or credentials. Missing optional checkouts may skip locally but sprint closure must run both and may not count skip. Preserve start-site/POSIX gates. Depends on S183.4. Sprint: #183.
