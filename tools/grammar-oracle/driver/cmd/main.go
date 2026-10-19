package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bashpp-tests/tools/grammar-oracle/driver"
)

func main() {
	root := flag.String("root", "", "bashsharp-tests root")
	sh := flag.String("sh-root", "", "read-only sh checkout")
	tool := flag.String("toolchain", "go1.27.2", "GOTOOLCHAIN value")
	derived := flag.String("derived-root", "", "root under which derived.tsv tests: sources are materialized (default -root)")
	verify := flag.Bool("verify", false, "verify committed comparison records")
	flag.Parse()
	if *root == "" {
		fmt.Fprintln(os.Stderr, "-root is required")
		os.Exit(2)
	}
	r, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	compare := filepath.Join(r, "docs/grammar-oracle/results/compare.jsonl")
	if *verify {
		err = driver.Verify(r, compare)
		if err != nil {
			fmt.Fprintln(os.Stderr, "FAIL:", err)
			os.Exit(1)
		}
		fmt.Println("PASS: grammar-oracle evidence verified")
		return
	}
	sum, err := driver.Run(driver.Options{Root: r, SHRoot: *sh, DerivedRoot: *derived, Toolchain: *tool, Timeout: 600 * time.Second, ComparePath: compare, ReportPath: filepath.Join(r, "docs/grammar-oracle/results/report.md"), FindingsPath: filepath.Join(r, "docs/grammar-oracle/findings.tsv")})
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(2)
	}
	t := sum.Tally
	if !sum.Pass {
		fmt.Fprintf(os.Stderr, "FAIL: grammar-oracle comparison: %d required disagreements, %d missing/errors; reported separately: %d probe, %d planned disagreements, %d known divergences; %d compared records\n", t.Required, t.Missing, t.Probes, t.Planned, t.Known, len(sum.Records))
		os.Exit(1)
	}
	if err = driver.Verify(r, compare); err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
	fmt.Printf("PASS: grammar-oracle comparison, %d records: %d required disagreements, %d missing/errors; reported separately: %d probe, %d planned disagreements, %d known divergences\n", len(sum.Records), t.Required, t.Missing, t.Probes, t.Planned, t.Known)
}
