package main

import (
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
)

// Sprint: #376; Story: #1553; Story-ID: 6fbca78eeae0
// Go requires a .go staging name, but the committed input is a .bsh file.
// Rewrite that exact file's compiler debug identity, preserving every source
// byte and line number. No output normalization or directory-wide rewrite is
// involved: unrelated source names and wrong positions remain observable.
func oracleBuildCommand(goBinary, sourceDir, sourceName, output string, testRow bool) []string {
	staged := upstreamProgramPath(sourceName)
	if testRow {
		staged = "main_test.go"
	}
	rewrite := filepath.Join(sourceDir, staged) + "=>" + sourceName
	// The compiler resolves its working directory through symlinks (notably
	// /var on macOS). Include that exact spelling, without widening the rule.
	if real, err := filepath.EvalSymlinks(sourceDir); err == nil && real != sourceDir {
		rewrite += ";" + filepath.Join(real, staged) + "=>" + sourceName
	}
	flag := "-gcflags=-trimpath " + strconv.Quote(rewrite)
	if testRow {
		return []string{goBinary, "test", "-c", flag, "-o", output, "."}
	}
	return []string{goBinary, "build", flag, "-o", output, "."}
}

func validOracleBuildCommand(argv []string, sourceDir, sourceName string, testRow bool) bool {
	if len(argv) == 0 || filepath.Base(sourceDir) != "oracle" || filepath.Base(filepath.Dir(sourceDir)) != "src" {
		return false
	}
	output := filepath.Join(filepath.Dir(filepath.Dir(sourceDir)), "bin", "oracle")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	return slices.Equal(argv, oracleBuildCommand(argv[0], sourceDir, sourceName, output, testRow))
}
