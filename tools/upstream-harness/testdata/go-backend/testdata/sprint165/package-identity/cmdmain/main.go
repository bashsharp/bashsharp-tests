// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// cmd/compile's shape: a main package that is the parent of its own
// internal tree, with an in-package test. cmd/go recompiles it as the test
// variant under its own identity (example.com/pkgid/cmdmain); the testmain
// identity (example.com/pkgid/cmdmain.test) is outside that tree and would be
// refused the import.
package main

import (
	"fmt"

	"example.com/pkgid/cmdmain/internal/deep"
)

func word() string { return deep.Word() }

func main() { fmt.Println(word()) }
