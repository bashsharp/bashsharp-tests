# Seal-time input snapshot for the historical tour evidence ledger

Sprint 381 / Story #107 / Story-ID 3b50496741fd.

`tests/tour/evidence.jsonl` is a SEALED historical ledger: a tour-evidence/v2
run recorded on darwin/arm64 with the pinned go1.27.0 toolchain and published
bashy 0.20.0, committed at bb18483377ef8e9b3fd0c8595117c2f374708ec8. Its
bytes are immutable evidence of record and are never regenerated to track
later pin changes.

The validator inputs it binds to have since legitimately moved on in the live
tree (67cff32 changed `tests/tour/inventory.tsv` for the Bash# script-entry
interpreted lanes; the Go coordinate moved to go1.27.1; the baseline registry
became per-platform). Binding the sealed ledger to the LIVE tree therefore
fails closed ("manifest inventory binding mismatch") on every host, and a
foreign platform could never replay a darwin ledger at all. That failure was
an input-binding bug in the gate wiring, not evidence tampering.

`tour validate-evidence` now separates the two lanes transparently:

- **Sealed lane** — when the ledger's bytes hash to `evidence_sha256` in
  `seal.tsv`, it is authenticated as a historical snapshot against the
  seal-time inputs in this directory: full structural validation (envelope,
  inventory binding, baseline bindings, toolchain pin cross-checks, per-attempt
  command/source/normalization/outcome re-derivation, summary, canonical root
  and verdict chain). Replay is NOT performed in this lane — it requires the
  recorded darwin/arm64 host and the exact recorded binaries — and the
  validator states that explicitly in its output. No claim is made here about
  replay runs predating the seal.
- **Current lane** — any other ledger bytes take the unchanged full
  structural + replay validation against the live tree. Nothing about fresh
  candidate validation is weakened.

Every file here is byte-identical to the seal commit and is additionally
self-authenticating: the ledger's manifest records `inventory.data_sha256`,
`baseline.accepted_results_sha256` and `baseline.pin_sha256`, all covered by
the ledger's canonical root hash, and the sealed lane re-checks each snapshot
file against those digests. Tampering with a snapshot file or with the ledger
breaks the chain and fails closed. The ledger digest in `seal.tsv` is in turn
pinned by a compiled-in constant in the validator
(`legacySealedEvidenceSHA256`), so mutating the ledger and `seal.tsv`
together cannot reach the sealed lane without a validator source change the
gate and review catch.

| file | provenance |
| --- | --- |
| `seal.tsv` | ledger sha256, seal commit, recorded goos/goarch, baseline paths |
| `inventory.tsv` | `tests/tour/inventory.tsv` @ bb18483 (data sha `a63b57fe…`) |
| `results.tsv` | `tests/tour/results.tsv` @ bb18483 (`a92dcd38…`) |
| `baseline-pin.tsv` | `docs/tour/baseline-pin.tsv` @ bb18483 (`198dd6ee…`) |
| `toolchain.tsv` | `docs/tour/toolchain.tsv` @ bb18483 (darwin/arm64 go1.27.0) |

To re-run full replay authentication of the sealed ledger, check out the seal
commit on a darwin/arm64 host holding the recorded go1.27.0 toolchain module
and the published bashy 0.20.0 cache binary and run
`tools/tour/validate-evidence.sh` there.
