module bashpp-tests/tools/grammar-oracle

// Test-only module, built with GOWORK=off. The toolchain line is exact, like
// tools/go-oracle, so the oracle is never rebuilt by a different compiler than
// the pinned one. github.com/antlr4-go/antlr/v4 is the only dependency
// (docs/grammar-oracle/pin.tsv); production modules never import this one.
go 1.27

toolchain go1.27.2

require github.com/antlr4-go/antlr/v4 v4.13.1

require golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
