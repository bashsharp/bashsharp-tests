package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvidenceToolchainSelectsHostInsteadOfFirstRow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "toolchain.tsv")
	if err := os.WriteFile(path, []byte("# host pins\ndarwin\tarm64\tgo1.27.0\tdarwin identity\ndarwin\tx86_64\tgo1.27.0\tintel identity\nlinux\tx86_64\tgo1.27.0\tlinux identity\n"), 0600); err != nil {
		t.Fatal(err)
	}
	row := evidenceToolchainRow(path, "linux", "x86_64")
	if len(row) != 4 || row[3] != "linux identity" {
		t.Fatalf("wrong host receipt: %v", row)
	}
	if row := evidenceToolchainRow(path, "linux", "arm64"); row != nil {
		t.Fatalf("unreviewed platform fell back to %v", row)
	}
}
