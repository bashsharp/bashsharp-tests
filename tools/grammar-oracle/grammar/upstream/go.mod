// Fence, not a module anyone builds: it keeps the pristine upstream
// Go/go_parser_base.go (which needs the generated parser to compile) out of
// the grammar-oracle module's ./... . The files here are byte-identical to
// antlr/grammars-v4 at the commit in docs/grammar-oracle/pin.tsv; local
// patches live outside this directory (checkpoint 2).
module bashpp-tests/tools/grammar-oracle/grammar/upstream

go 1.27
