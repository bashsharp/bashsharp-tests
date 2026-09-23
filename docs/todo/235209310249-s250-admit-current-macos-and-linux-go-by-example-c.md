---
id: "235209310249"
kind: bug
title: S250 admit current macOS and Linux Go by Example candidates
seq: 89
status: assigned
priority: p0
created: 2026-09-23T09:43:52.988025Z
weave: 18
assignee: qiangli
sprint: 250
sprint_id: c912e608-edfe-59b8-bd36-a98f6dad1634
sprint_title: Validate Go by Example, Go Tour and BashSharp Tour on three hosts
---

The current clean Bashy candidate on macOS cannot start Go by Example because its manifest digest is absent from the reviewed candidates table; the Linux baseline used an older authenticated candidate. Review exact current source/binary manifests for macOS arm64 and Linux amd64, register only authenticated rows, then run full Go by Example 255 on both hosts. Preserve corpus and limits. If a current candidate is red, report exact case and create a separate root-cause repair story. Private Sprint 250 Story #676 holds the raw evidence.
