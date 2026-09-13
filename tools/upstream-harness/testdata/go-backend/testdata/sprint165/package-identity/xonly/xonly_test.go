// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// cmd/internal/testdir's shape: a package whose tests are all external.
// cmd/go builds no in-package variant at all; the one file is an xtest file
// and forms the external test package by that role. Its import of the
// tested package's own internal sibling is granted to example.com/pkgid/xonly
// (cmd/go loads an xtest's imports with the tested package as the importer),
// never to example.com/pkgid/xonly.test.
package xonly_test

import (
	"testing"

	"example.com/pkgid/xonly/internal/deep"
)

func TestOnlyExternal(t *testing.T) {
	if got := deep.Word(); got != "deep" {
		t.Fatalf("deep.Word() = %q, want %q", got, "deep")
	}
}
