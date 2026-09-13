// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
package lib

import "testing"

func TestWordInPackage(t *testing.T) {
	if got := Word(); got != "deep" {
		t.Fatalf("Word() = %q, want %q", got, "deep")
	}
}
