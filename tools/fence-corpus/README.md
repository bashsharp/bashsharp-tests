# Shared fenced-example gate

Sprint #562, Story #3047 (`98e7589729ed`).

```sh
BASHY_BIN=/absolute/path/to/bashy sh tools/fence-corpus/check.sh \
  --corpus python-by-example --corpus examples-fenced
sh tools/fence-corpus/validate-inventory.sh python-by-example
sh tools/fence-corpus/selftest.sh
```

`check.sh` requires at least one `--corpus`. Each manifest row selects `run`,
`compiled`, or `build`; `--mode` filters those rows, and `--case ID` selects an ID.
Filters never waive committed count floors: a small sample of a larger corpus
can print PASS for its selected cases and still fail the aggregate ratchet.
No selected cases is a failure. Every row is structurally checked before
filtering, including the ban on xfail and the presence of expected output.

Each case gets a fresh copy of its source directory, a private HOME, TMPDIR,
Go build cache and work directory. Its working directory is that copy, so
relative fixture assets work and writes do not modify the corpus. Symlink assets
are rejected so they cannot redirect writes outside the copy. Capture uses
files and compares bytes, including NULs and final newlines. The first line of
`name.expected` must be `# rc=N` followed by a newline; the remaining bytes are
stdout. `name.stderr`, when present, is raw stderr; otherwise stderr must be
empty. The manifest rc and expected header must agree. Timeouts, launch errors,
and surviving descendants fail even if an expected rc happens to match.

An optional `name.compare` contains exactly one GBE comparator name from
`NormalizerNames` in `tools/go-by-example/gbe/normalizer.go`, excluding the
`none` sentinel. Both expected and actual streams pass through the existing
GBE normalizer with its stream-specific rules; unknown names and invalid shapes
fail. These are the GBE example-specific licences, not general output sorting
or number removal. In particular, `duration` permits Go test elapsed-time
variation but preserves ordinary printed durations; `panic_trace` preserves
the panic body on stderr while ignoring the stack. Exit status is always checked
independently and cannot be waived by a comparator.

The default **20-second wall-time budget** covers all executable stages of a
case together. `FENCE_TIMEOUT` may only shorten it. This is the existing GBE
process supervisor, including its bounded cleanup grace, not a new timeout
implementation. Temporary case directories and captured files are removed
when the case finishes; mismatch/stage diagnostics are printed.

## Toolchains and compiled cases

`build.sh` links the existing `tools/go-by-example/gbe` package into a private
ignored build tree under `tools/fence-corpus/.cache`. It reuses the existing
GBE build script, adding only an output file for the authenticated SDK path.
The original source files and three-mode gate are unchanged. The adapter calls
GBE's `capture`, `toolchainPin`, `resolveToolchain`, `runEnv`, `buildEnv`, and
managed SDK cache helpers directly. The SDK is pinned by
`docs/go-by-example/toolchain.tsv`, including its binary checksum. The existing
build resolver may locate that SDK through a bootstrap Go installation; cases
never execute an arbitrary Go toolchain from PATH. Missing SDKs are fatal.
The SDK is required even for run-mode checks because it builds the supervisor.

Run mode invokes `$BASHY_BIN --bashsharp FILE`. Compiled mode invokes:

```text
$BASHY_BIN transpile --bashsharp FILE -o generated/main.go
<pinned SDK>/bin/go build -o artifact .
artifact
```

Set `FENCE_SH_MODULE` to the **candidate's** existing `mvdan.cc/sh/v3` lowering
runtime module directory for compiled or build cases. The generated module uses the same
runtime replacement pattern as GBE, with module downloads and toolchain
switching disabled. It does not use the Go-source importer. The runner never
edits or provisions that runtime checkout.

Build mode first invokes `$BASHY_BIN check --bashsharp FILE`, then uses the
same transpile and native build stages as compiled mode. Every stage must
succeed; neither the source program nor the artifact is executed. Its manifest
rc must be 0 and its `.expected` must contain only `# rc=0` plus a newline.
It has its own `build` counts rows and supports `--mode build`.

Before any case runs, the runner unions selected rows' comma/space-separated
`needs` and calls a small fence for each language through the candidate bashy.
This uses bashy's own managed provisioning, including transitive tools, rather
than host executables. Unknown needs and provisioning failures fail the run;
there are no skips. Provisioning has a separate 120-second deadline per language;
the case deadline remains at most 20 seconds.

Each run removes `tools/fence-corpus/.cache/run-<id>`, including its
provision log and default toolchains, after restoring owner write permissions.
Cleanup applies to success, failure, and trapped INT/TERM/HUP signals. On a
signal, the active bounded stage finishes or times out under GBE supervision
before cleanup, so its children are reaped before their files are removed.
Set `FENCE_TOOLCHAIN_DIR` to retain and reuse a dedicated toolchain directory
(concurrent reuse fails with a busy-directory error). Its owner write bits are
restored at run end, and its latest `provision.log` is retained there. Temporary
run scratch is still removed. Symlink targets are never traversed or chmodded.
`FENCE_BIN_CACHE` is no longer consumed. The runner owns the `bin`, `python`, and
`uv` subdirectories and removes their write bits during case execution; symlink
targets such as the authenticated Go SDK are never chmodded. Case HOME, TMPDIR,
working directory, build cache, and uv scratch cache remain fresh (uv writes
cache metadata even during interpreter discovery). These permissions and offline
settings are isolation for trusted corpus programs, not an OS security sandbox.

Case environments are built from an explicit allowlist with empty PATH, fixed
locale/timezone, `BASHY_HINTS=off`, and managed toolchain paths. No caller agent,
session, proxy, or tool-version variables are inherited. `BASHY_OFFLINE=1`,
`UV_OFFLINE=1`, `UV_PYTHON_DOWNLOADS=never`, Go's disabled proxy/sumdb and Cargo's
offline setting prevent tool acquisition during cases. Python uses
`UV_PYTHON_INSTALL_DIR` and `UV_CACHE_DIR`: yoke's
`external/python.EnsureInterpreter` passes these to uv's managed-only
`python find/install` (unlike the independent `BASHY_BIN_CACHE`).

Failures show expected/actual rc and both streams, with up to 20 differing line
pairs per stream, capped at 200 bytes per displayed line. Quoted bytes expose
missing newlines and control characters. Provisioning failures print a bounded excerpt before cleanup; full logs are
retained only with an explicit `FENCE_TOOLCHAIN_DIR`.

## Tables and ratchets

TSV headers are required (a leading `# ` is also accepted). Empty fields are
invalid; use `-` where the contract permits an absent value.

| File | Header |
| --- | --- |
| cases.tsv | `id mode file needs rc xfail` |
| pin.tsv | `source upstream_repo commit code_license prose_license` |
| inventory.tsv | `source_ref disposition case_id reason` |
| counts.tsv | `mode platform cases pass` |

The displayed header fields are separated by literal tabs in files. Pin commits
must be full 40- or 64-digit hex object IDs. Inventory validation rejects
unknown dispositions, duplicate source references, dangling case IDs, and
cases without a `case` or `diagnostic` inventory row. Other dispositions use
`case_id=-`. Allowed dispositions are `case`, `diagnostic`, `prose`, `fragment`,
`interactive`, `os-only:<os>`, `env-dependent`, and `missing-feature:<short-name>`.
This validates accounting structure; upstream completeness remains the pinned
generator's responsibility.

`platform` is Go's `GOOS/GOARCH`, e.g. `darwin/arm64`, `linux/amd64`, or
`windows/amd64`. A selected mode needs a committed count row for the current
platform. Full runs also enforce floors for modes removed from the manifest.
Every case failure is fatal independently of the floors.

```sh
BASHY_BIN=/absolute/path/to/bashy sh tools/fence-corpus/check.sh \
  --corpus examples-fenced --update-counts
```

This explicit operation only raises/adds current-platform counts, preserves all
other platforms, refuses `--case`, and writes nothing if any corpus fails.
Review and commit the resulting TSV diff; normal checks never update counts.
An initial corpus can start with the counts header alone, then use this reviewed
operation to establish its first platform floor. Never lower floors to get a
passing gate.

## Harness and verification

The one new mandatory section in `harness/run.sh` invokes `aggregate.sh`.
It recursively discovers directories named `*-by-example`, `examples-fenced`,
or `tour-fenced`, excluding `.git`, `.cache`, `.agents`, and repository `tools`
and `docs` (which contain machinery and provenance with overlapping names).
Discovery does not depend on cases.tsv existing: a deleted manifest fails.
Each discovered corpus must pass both validation and execution. Missing runner
scripts and all failures propagate. `corpora.tsv` lists corpus names, comma-separated
OS names (`linux`, `darwin`, `windows`), and `pending` or `required` status.
Unlisted discovered corpora fail. A required corpus missing on a listed OS
fails even if no corpus directories remain. Pending rows print `PENDING <corpus>`
and permit an absent corpus; any discovered corpus still undergoes all checks.
All nine initial rows are pending; the manager promotes each to required when
its corpus lands. `aggregate.sh [ROOT]` supports
an explicit discovery root for fixture testing; the harness always uses the
repository root. Existing harness sections, including the Go three-mode gate,
are unchanged.

The selftest copies a tiny committed synthetic corpus and applies isolated
mutations to test PASS, stdout mismatch, final-newline mismatch, count drops,
non-decreasing updates, inventory errors, missing expected output, xfail,
timeouts, rc mismatch, missing runner/SDK/runtime, and fatal aggregate behavior.
It also verifies build-only hangs are never executed, failures in each build
stage, seven comparator licences (volatile changes, real differences, and rc
mismatches), strict stream routing for `panic_trace` and `closing_channel_order`,
unknown names, and run cleanup including interruption.
Its shell runner is a deliberate protocol stub, not a Bash# oracle. Compiled
and build selftesting stub the check/transpile protocol, then compile real Go
(and execute it only for compiled mode) with
the pinned SDK. `testdata/runtime/go.mod` is an empty test-only module identity.
The fixtures represent one synthetic source example, one case, and one `case`
disposition; no upstream examples were imported for this infrastructure story.
