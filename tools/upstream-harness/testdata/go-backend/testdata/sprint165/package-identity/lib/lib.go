// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// The full three-role shape: an ordinary file importing the tested
// package's own internal sibling, an in-package test file and an external
// test file.
package lib

import "example.com/pkgid/lib/internal/deep"

// Word returns the internal package's word.
func Word() string { return deep.Word() }
