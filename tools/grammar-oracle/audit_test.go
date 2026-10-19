package grammaroracle

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bashpp-tests/tools/grammar-oracle/oracle"
)

// TestPredicateAudit runs the authored R6 audit units in testdata/go through
// the oracle (CONTRACT.md, "Go predicate audit"). A neg-* file must be
// rejected, every other file accepted, unless findings.tsv has an open row
// for it (audit/<file>). Each expectation is the gcsyntax
// (go1.27) syntax verdict; go/parser, which is never the oracle, cross-checks
// the authored expectation here, outside package oracle.
func TestPredicateAudit(t *testing.T) {
	files, err := filepath.Glob("testdata/go/*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no audit units: %v", err)
	}
	known := findings(t)
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		want := oracle.Accept
		if strings.HasPrefix(filepath.Base(f), "neg-") {
			want = oracle.Reject
		}
		_, perr := parser.ParseFile(token.NewFileSet(), f, src, parser.SkipObjectResolution)
		if (perr == nil) != (want == oracle.Accept) {
			t.Errorf("%s: authored %s, go/parser says %v", f, want, perr)
		}
		checkLedger(t, known, "audit/"+filepath.Base(f), want, oracle.Recognize("unit", "sourceFile", src))
	}
	checkLedgerSeen(t, known, true)
}
