---
id: 9b5fbd6d27e5
kind: task
title: 'S186.2 polyglot-gate: explicit .ts relative import row'
seq: 83
status: done
priority: p2
created: 2026-09-15T08:10:31.488605Z
assignee: transom
sprint: 186
closed: 2026-09-15T08:17:19.194184Z
closed_by: transom
---

One row: an ESM package.json + src/format.ts project in scratch; a ~~~ts fence importing ./src/format.ts runs on Node and returns 1.5k.
