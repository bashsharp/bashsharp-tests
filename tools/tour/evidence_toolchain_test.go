package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvidenceToolchainSelectsHostInsteadOfFirstRow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "toolchain.tsv")
	if err := os.WriteFile(path, []byte("# host pins\ndarwin\tarm64\tgo1.27.1\tgo version go1.27.1 darwin/arm64\tdarwin-digest\nlinux\tx86_64\tgo1.27.1\tgo version go1.27.1 linux/amd64\tlinux-digest\n"), 0600); err != nil {
		t.Fatal(err)
	}
	row := evidenceToolchainRow(path, "linux", "x86_64")
	if len(row) != 5 || row[3] != "go version go1.27.1 linux/amd64" {
		t.Fatalf("wrong host receipt: %v", row)
	}
	darwin := evidenceToolchainRow(path, "darwin", "arm64")
	if len(darwin) != 5 || darwin[3] != "go version go1.27.1 darwin/arm64" {
		t.Fatalf("wrong Darwin host receipt: %v", darwin)
	}
	if row := evidenceToolchainRow(path, "linux", "arm64"); row != nil {
		t.Fatalf("unreviewed platform fell back to %v", row)
	}
}

func TestEvidenceToolchainSelectsRecordedIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "toolchain.tsv")
	if err := os.WriteFile(path, []byte("darwin\tarm64\tgo1.27.1\tgo version go1.27.1 darwin/arm64\tdarwin-digest\nlinux\tx86_64\tgo1.27.1\tgo version go1.27.1 linux/amd64\tlinux-digest\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, identity, digest string
	}{
		{"linux", "go version go1.27.1 linux/amd64", "linux-digest"},
		{"darwin", "go version go1.27.1 darwin/arm64", "darwin-digest"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := evidenceToolchainIdentityRow(path, tt.identity)
			if row == nil || field(row, 3) != tt.identity || field(row, 4) != tt.digest {
				t.Fatalf("recorded identity was not authenticated: %v", row)
			}
		})
	}
	for _, identity := range []string{"", "go version go1.27.1 linux/arm64", "go version go1.27.0 linux/amd64"} {
		if row := evidenceToolchainIdentityRow(path, identity); row != nil {
			t.Fatalf("missing or mismatched identity %q selected %v", identity, row)
		}
	}
}

func TestEvidenceToolchainRejectsMissingOrMismatchedDigest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "toolchain.tsv")
	if err := os.WriteFile(path, []byte("linux\tx86_64\tgo1.27.1\tgo version go1.27.1 linux/amd64\tlinux-digest\n"), 0600); err != nil {
		t.Fatal(err)
	}
	row := evidenceToolchainIdentityRow(path, "go version go1.27.1 linux/amd64")
	if row == nil {
		t.Fatal("matching identity was not selected")
	}
	for _, digest := range []string{"", "darwin-digest"} {
		if evidenceToolchainDigestMatches(row, digest) {
			t.Fatalf("missing or mismatched digest %q authenticated", digest)
		}
	}
	if !evidenceToolchainDigestMatches(row, "linux-digest") {
		t.Fatal("matching digest was rejected")
	}
}
