package oracle

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The recognizer depends on goparser, the ANTLR runtime and the standard
// library only: never sh/syntax, sh/gosource or go/parser.
func TestIndependentImports(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", ".").Output()
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		"bashpp-tests/tools/grammar-oracle/oracle":   true,
		"bashpp-tests/tools/grammar-oracle/goparser": true,
		"github.com/antlr4-go/antlr/v4":              true,
		"golang.org/x/exp/constraints":               true, // antlr4-go's own dependencies
		"golang.org/x/exp/slices":                    true,
	}
	for _, p := range strings.Fields(string(out)) {
		if !allowed[p] {
			t.Errorf("oracle depends on %s", p)
		}
	}
	std, err := exec.Command("go", "list", "-deps", "-f", "{{if .Standard}}{{.ImportPath}}{{end}}", ".").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range strings.Fields(string(std)) {
		if strings.HasPrefix(p, "go/") {
			t.Errorf("oracle depends on %s", p)
		}
	}
}

// Checkpoint 1 feasibility: one Tour of Go unit through the generated parser.
func TestTourUnit(t *testing.T) {
	src, err := os.ReadFile("../../../tour/_content/tour/basics/imports.bsh")
	if err != nil {
		t.Fatal(err)
	}
	if r := Recognize("unit", "sourceFile", src); r.Verdict != Accept {
		t.Fatalf("imports.bsh: %+v", r)
	}
}

func TestRejectPositions(t *testing.T) {
	for _, c := range []struct {
		src                    string
		offset, line, col, chr int
	}{
		{"package p\n\nfunc f() {\n\tx := (1\n}\n", 30, 4, 9, 9}, // missing ')' at the EOS newline
		{"package p\nvar s = \"é\" @\n", 23, 2, 14, 13},          // lexer: '@' after a 2-byte rune
		{"package p\nfunc f() {}}\n", 21, 2, 12, 12},             // extraneous '}' (recovered, still a reject)
		{"package p\nvar x = 1 +\n", 20, 2, 11, 11},              // no viable alternative from '+'
		{"package p\r\nvar x = )\r\n", 19, 2, 9, 9},              // CR stays in the line
	} {
		r := Recognize("unit", "sourceFile", []byte(c.src))
		if r.Verdict != Reject || len(r.Diagnostics) == 0 {
			t.Errorf("%q: %+v, want reject", c.src, r)
			continue
		}
		d := r.Diagnostics[0]
		if d.Offset != c.offset || d.Line != c.line || d.Col != c.col || d.Char != c.chr {
			t.Errorf("%q: first error %+v, want offset %d %d:%d char %d", c.src, d, c.offset, c.line, c.col, c.chr)
		}
	}
}

func TestUnknownEntry(t *testing.T) {
	if r := Recognize("mixed", "sourceFile", nil); r.Verdict != Error {
		t.Fatalf("%+v", r)
	}
}
