package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcceptedPlatformSelectsAndAuthenticatesExactPlatform(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "docs/tour"), 0o755)
	os.MkdirAll(filepath.Join(root, "tests/tour"), 0o755)
	write := func(rel, body string) string {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return sha256hex([]byte(body))
	}
	darwinResults := write("tests/tour/results.darwin-arm64.tsv", "darwin\n")
	darwinPin := write("docs/tour/baseline-pin.darwin-arm64.tsv", "darwin-pin\n")
	linuxResults := write("tests/tour/results.linux-x86_64.tsv", "linux\n")
	linuxPin := write("docs/tour/baseline-pin.linux-x86_64.tsv", "linux-pin\n")
	index := strings.Join([]string{
		"darwin\tarm64\ttests/tour/results.darwin-arm64.tsv\t" + darwinResults + "\tdocs/tour/baseline-pin.darwin-arm64.tsv\t" + darwinPin + "\tgo version go1.27.0 darwin/arm64\t" + strings.Repeat("a", 64),
		"linux\tx86_64\ttests/tour/results.linux-x86_64.tsv\t" + linuxResults + "\tdocs/tour/baseline-pin.linux-x86_64.tsv\t" + linuxPin + "\tgo version go1.27.1 linux/amd64\t" + strings.Repeat("b", 64),
	}, "\n") + "\n"
	write(acceptedPlatformsPath, index)

	got, err := acceptedPlatform(root, "linux", "x86_64")
	if err != nil {
		t.Fatal(err)
	}
	if got.Results != "tests/tour/results.linux-x86_64.tsv" {
		t.Fatalf("selected %q", got.Results)
	}
	if _, err := acceptedPlatform(root, "linux", "arm64"); err == nil {
		t.Fatal("wrong platform fell back")
	}
	for _, unsafe := range []string{"../results.tsv", "tests//tour/results.tsv", "tests\\tour\\results.tsv", "C:/results.tsv", "/tmp/results.tsv"} {
		badIndex := strings.Replace(index, "tests/tour/results.linux-x86_64.tsv", unsafe, 1)
		write(acceptedPlatformsPath, badIndex)
		if _, err := acceptedPlatform(root, "linux", "x86_64"); err == nil {
			t.Errorf("accepted unsafe path %q", unsafe)
		}
	}
	write(acceptedPlatformsPath, index)
	write("tests/tour/results.linux-x86_64.tsv", "forged\n")
	if _, err := acceptedPlatform(root, "linux", "x86_64"); err == nil {
		t.Fatal("tampered platform observation accepted")
	}
}
