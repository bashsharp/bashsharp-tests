# Full harness fixture contracts

`run.sh` keeps every discovered fixture in its denominator. Lowering fixtures
use the existing `docs/lowering/go-profile-cases.tsv` and
`profile-additional.tsv` contracts: exact numeric status, stdout and stderr,
including expected rejections. They execute from their declared fixture root
so source locations are compared unchanged. A nonzero exit alone does not
prove a contracted rejection.

Action annotations accept Go `//` comments or shell `#` comments.
`# errorcheck` fixtures must reject and match their `# ERROR` diagnostics.
The negative Python fixture requires an `AttributeError` naming the requested
attribute; a missing package does not satisfy it.

Provision the external fixture environments described by
`tools/python-package-gate.sh` and `tools/sprint184-typescript-gate.sh` before
running. Expose the pinned Python packages through `PYTHONPATH` and
`BASHPP_PYTHON`; configure Node, `BASHPP_TYPESCRIPT_MODULE`, `BASHPP_BUN`, and
`OPENCODE_ROOT`. The shared `tools/opencode-fixture.sh` uses the pinned package
directory and stdin, verifies its output, and checks that its checkout remains
unchanged. The harness neither installs packages nor skips missing dependencies.

Run `ruby tests/lowering/interpreted_contract_test.rb` to check grading against
wrong statuses, missing diagnostics, wrong source locations and extra output.
