// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
package main

import "testing"

func TestWordInMainPackage(t *testing.T) {
	if got := word(); got != "deep" {
		t.Fatalf("word() = %q, want %q", got, "deep")
	}
}
