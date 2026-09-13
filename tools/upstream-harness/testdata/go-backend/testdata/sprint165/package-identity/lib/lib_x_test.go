// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
package lib_test

import (
	"testing"

	"example.com/pkgid/lib"
	"example.com/pkgid/lib/internal/deep"
)

func TestWordExternal(t *testing.T) {
	if lib.Word() != deep.Word() {
		t.Fatalf("lib.Word() = %q, deep.Word() = %q", lib.Word(), deep.Word())
	}
}
