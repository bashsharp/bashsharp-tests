package main

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strings"
)

// interpretedTestEntry is a generated driver artifact, not a replacement for
// the byte-pinned source copy. Insert only missing driver imports after the
// package clause and append its declarations. This keeps the test assertions
// unchanged and gives plain bashy one .bsh entry, with no CLI mode selectors.
func interpretedTestEntry(source []byte, driver string) (string, error) {
	fset := token.NewFileSet()
	src, err := goparser.ParseFile(fset, "source.bsh", source, 0)
	if err != nil {
		return "", err
	}
	drv, err := goparser.ParseFile(fset, "driver.go", driver, 0)
	if err != nil {
		return "", err
	}
	imports := map[string]bool{}
	for _, imp := range src.Imports {
		imports[imp.Path.Value] = true
	}
	var added strings.Builder
	for _, imp := range drv.Imports {
		if !imports[imp.Path.Value] {
			fmt.Fprintf(&added, "\nimport %s\n", imp.Path.Value)
		}
	}
	offset := fset.Position(src.Name.End()).Offset
	entry := string(source[:offset]) + added.String() + string(source[offset:])
	for _, decl := range drv.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			continue
		}
		start, end := fset.Position(decl.Pos()).Offset, fset.Position(decl.End()).Offset
		entry += "\n" + driver[start:end] + "\n"
	}
	return entry, nil
}

func interpretedCommand(bashy, entry string, args []string) []string {
	return append([]string{bashy, entry}, args...)
}
