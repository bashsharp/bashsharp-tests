# Grammar sources — provenance and notices

`upstream/` holds unmodified files from
[antlr/grammars-v4](https://github.com/antlr/grammars-v4) `golang/` at the
commit pinned in `docs/grammar-oracle/pin.tsv`, with their SHA-256 digests.
They are BSD-3-Clause: the `[The "BSD licence"]` header at the top of
`GoLexer.g4` and `GoParser.g4` is the notice (the repository has no top-level
LICENSE; `golang/README.md` states BSD-3). Keep those headers on any derived
file. `upstream/go.mod` is a local fence only, so the go tool never tries to
build the pristine `Go/go_parser_base.go` on its own.

Do not edit `upstream/`. Local patches (the Go-target transform, Go 1.27
reconciliation, Bash# composition) are made from checkpoint 2 on in sibling
files and each one is recorded.
