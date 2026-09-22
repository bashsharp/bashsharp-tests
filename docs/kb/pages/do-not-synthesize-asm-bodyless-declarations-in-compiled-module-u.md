---
id: 01a0caaa-f5e6-7d3e-bb2a-1690ca74f905
seq: 1
form: page
type: lesson
title: Do not synthesize asm bodyless declarations in compiled module units
description: 'WHEN Bash++ compiled testdir module units include authenticated .s companions, copy the companion files but do not generate a separate bashpp_asmdecls.go: the transpiled unit already carries bodyless declarations with original //line positions, and an extra stub redeclares assembly functions such as issue74648.F or issue15609.jump.'
status: candidate
source:
    tool: codex-gpt-5.5-n
    host: dragon
    episode: weave-issue-14
created: "2026-09-22T19:49:56Z"
---
