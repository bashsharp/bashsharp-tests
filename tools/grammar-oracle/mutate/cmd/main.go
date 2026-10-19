// Command cmd materializes the frozen derived input set under -out (as if
// -out were the bashsharp-tests root), so gate.sh can resolve derived.tsv
// sources without tracked bytes. It fails if derived.tsv is stale.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bashpp-tests/tools/grammar-oracle/mutate"
)

func main() {
	root := flag.String("root", "", "bashsharp-tests root")
	out := flag.String("out", "", "directory to materialize into")
	flag.Parse()
	if *root == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "-root and -out are required")
		os.Exit(2)
	}
	if err := run(*root, *out); err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
}

func run(root, out string) error {
	parents, err := mutate.Parents(root)
	if err != nil {
		return err
	}
	rows, err := mutate.Generate(parents)
	if err != nil {
		return err
	}
	want, err := os.ReadFile(filepath.Join(root, "docs/grammar-oracle/derived.tsv"))
	if err != nil {
		return err
	}
	if !bytes.Equal(mutate.Render(rows), want) {
		return fmt.Errorf("docs/grammar-oracle/derived.tsv is stale")
	}
	return mutate.Materialize(out, rows)
}
