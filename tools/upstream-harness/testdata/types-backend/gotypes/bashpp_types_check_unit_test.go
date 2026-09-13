package types_test

import (
	"go/token"
	"testing"

	. "go/types"
)

func gotypesFixture() (*token.FileSet, map[string]*token.File) {
	fset := token.NewFileSet()
	src := []byte("package p\nvar y int\nvar x int\nvar x int\n")
	file := fset.AddFile("file.go", -1, len(src))
	file.SetLinesForContent(src)
	return fset, map[string]*token.File{"file.go": file}
}

func TestBashppParseGotypesDropsSecondaryDiagnostics(t *testing.T) {
	fset, known := gotypesFixture()
	errs, unparsed := bashppParseGotypesDiagnostics(fset, known, "file.go:4:5: x redeclared in this block\nfile.go:3:5: \tother declaration of x\n")
	if unparsed != 0 {
		t.Fatalf("unparsed = %d, want 0", unparsed)
	}
	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1: %v", len(errs), errs)
	}
	err := errs[0].(Error)
	if got := fset.Position(err.Pos); got.Filename != "file.go" || got.Line != 4 || got.Column != 5 {
		t.Fatalf("position = %v, want file.go:4:5", got)
	}
	if err.Msg != "x redeclared in this block" {
		t.Fatalf("Msg = %q", err.Msg)
	}
}

func TestBashppParseGotypesLeadingTabFirstIsUnparsed(t *testing.T) {
	fset, known := gotypesFixture()
	errs, unparsed := bashppParseGotypesDiagnostics(fset, known, "file.go:3:5: \tother declaration of x\n")
	if len(errs) != 0 || unparsed != 1 {
		t.Fatalf("errs, unparsed = %d, %d; want 0, 1", len(errs), unparsed)
	}
}

func TestBashppParseGotypesKeepsIndependentPrimaries(t *testing.T) {
	fset, known := gotypesFixture()
	errs, unparsed := bashppParseGotypesDiagnostics(fset, known, "file.go:3:5: first primary\nfile.go:4:5: second primary\n")
	if unparsed != 0 {
		t.Fatalf("unparsed = %d, want 0", unparsed)
	}
	if len(errs) != 2 {
		t.Fatalf("len(errs) = %d, want 2: %v", len(errs), errs)
	}
}

func TestBashppParseGotypesDropsTwoSecondaryDiagnostics(t *testing.T) {
	fset, known := gotypesFixture()
	errs, unparsed := bashppParseGotypesDiagnostics(fset, known, "file.go:4:5: x redeclared in this block\nfile.go:3:5: \tother declaration of x\nfile.go:2:5: \tprevious case\n")
	if unparsed != 0 {
		t.Fatalf("unparsed = %d, want 0", unparsed)
	}
	if len(errs) != 1 {
		t.Fatalf("len(errs) = %d, want 1: %v", len(errs), errs)
	}
}

// Sprint: #154; Story: S154.1; Story-ID: 29abb27c8659
// gc's own shape for a sub-error: a TAB-prefixed line carrying its position
// is the secondary go/types ignores — never an error, never unattributed.
func TestBashppParseGotypesDropsGcShapedContinuation(t *testing.T) {
	fset, known := gotypesFixture()
	errs, unparsed := bashppParseGotypesDiagnostics(fset, known, "file.go:4:5: x redeclared in this block\n\tfile.go:3:5: other declaration of x\n")
	if unparsed != 0 || len(errs) != 1 {
		t.Fatalf("errs, unparsed = %d, %d; want 1, 0: %v", len(errs), unparsed, errs)
	}
	if errs, unparsed := bashppParseGotypesDiagnostics(fset, known, "\tfile.go:3:5: other declaration of x\n"); len(errs) != 0 || unparsed != 1 {
		t.Fatalf("leading continuation: errs, unparsed = %d, %d; want 0, 1", len(errs), unparsed)
	}
}

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// An unpositioned TAB line is the primary's own continuation (go/types joins
// sub-errors without a position into one Msg, and the ERROR comments match
// against that joined message); a positioned TAB line, in either the go/types
// (": \t") or the gc ("\t<pos>: ") shape, stays the secondary upstream ignores.
func TestBashppParseGotypesFoldsUnpositionedContinuation(t *testing.T) {
	fset, known := gotypesFixture()
	errs, unparsed := bashppParseGotypesDiagnostics(fset, known, "file.go:4:5: not enough arguments in call to f\n\thave ()\n\twant (int)\nfile.go:3:5: \tother declaration of x\n\tfile.go:2:5: previous case\n")
	if unparsed != 0 || len(errs) != 1 {
		t.Fatalf("errs, unparsed = %d, %d; want 1, 0: %v", len(errs), unparsed, errs)
	}
	if got := errs[0].(Error).Msg; got != "not enough arguments in call to f\n\thave ()\n\twant (int)" {
		t.Fatalf("Msg = %q", got)
	}
}

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// Mixed continuations: unpositioned lines fold into the primary they follow
// even across a dropped positioned secondary, a second primary starts a new
// Msg, and a method-signature detail folds like have/want does.
func TestBashppParseGotypesFoldsMixedContinuations(t *testing.T) {
	fset, known := gotypesFixture()
	out := "file.go:2:5: cannot use x (variable of type T) as I value in assignment: T does not implement I (wrong type for method M)\n" +
		"\t\thave M(int)\n" +
		"\t\twant M(string)\n" +
		"\tfile.go:3:5: other declaration of x\n" +
		"\tsee also\n" +
		"file.go:4:5: x redeclared in this block\n" +
		"file.go:3:5: \tother declaration of x\n" +
		"\thave ()\n"
	errs, unparsed := bashppParseGotypesDiagnostics(fset, known, out)
	if unparsed != 0 || len(errs) != 2 {
		t.Fatalf("errs, unparsed = %d, %d; want 2, 0: %v", len(errs), unparsed, errs)
	}
	want := []string{
		"cannot use x (variable of type T) as I value in assignment: T does not implement I (wrong type for method M)\n\t\thave M(int)\n\t\twant M(string)\n\tsee also",
		"x redeclared in this block\n\thave ()",
	}
	for i, w := range want {
		if got := errs[i].(Error).Msg; got != w {
			t.Fatalf("errs[%d].Msg = %q, want %q", i, got, w)
		}
	}
	if got := fset.Position(errs[1].(Error).Pos); got.Line != 4 || got.Column != 5 {
		t.Fatalf("errs[1] position = %v, want file.go:4:5", got)
	}
}

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// An unpositioned continuation with no primary to belong to is unattributed,
// exactly as a leading positioned one is.
func TestBashppParseGotypesLeadingUnpositionedContinuationIsUnparsed(t *testing.T) {
	fset, known := gotypesFixture()
	errs, unparsed := bashppParseGotypesDiagnostics(fset, known, "\thave ()\n\twant (int)\nfile.go:4:5: not enough arguments in call to f\n")
	if len(errs) != 1 || unparsed != 2 {
		t.Fatalf("errs, unparsed = %d, %d; want 1, 2: %v", len(errs), unparsed, errs)
	}
	if got := errs[0].(Error).Msg; got != "not enough arguments in call to f" {
		t.Fatalf("Msg = %q", got)
	}
}
