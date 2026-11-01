//go:build !windows

package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidatePayloadLayouts(t *testing.T) {
	for _, split := range []bool{false, true} {
		t.Run(fmt.Sprint("split=", split), func(t *testing.T) {
			launcher := filepath.Join(t.TempDir(), "bashy")
			if err := os.WriteFile(launcher, []byte("native candidate"), 0755); err != nil {
				t.Fatal(err)
			}
			payload := launcher
			if split {
				payload += ".real"
				if err := os.WriteFile(payload, []byte("separate payload"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			bytes, err := os.ReadFile(payload)
			if err != nil {
				t.Fatal(err)
			}
			reviewed := fmt.Sprintf("%x", sha256.Sum256(bytes))
			if got := candidatePayloadPath(launcher); got != payload {
				t.Fatalf("payload=%s want %s", got, payload)
			}
			if _, err := authenticateFile(candidatePayloadPath(launcher), reviewed); err != nil {
				t.Fatal(err)
			}
			if _, err := authenticateFile(candidatePayloadPath(launcher), fmt.Sprintf("%064d", 1)); err == nil {
				t.Fatal("incorrect reviewed hash accepted")
			}
			if err := os.WriteFile(payload, []byte("tampered"), 0755); err != nil {
				t.Fatal(err)
			}
			if _, err := authenticateFile(candidatePayloadPath(launcher), reviewed); err == nil {
				t.Fatal("tampered payload accepted")
			}
			if err := os.Remove(payload); err != nil {
				t.Fatal(err)
			}
			if _, err := authenticateFile(candidatePayloadPath(launcher), reviewed); err == nil {
				t.Fatal("missing payload accepted")
			}
		})
	}
}

func TestReviewedNativeCandidateStillRequiresExactIdentity(t *testing.T) {
	goos, goarch := hostIdentity()
	manifest := strings.Repeat("1", 64)
	payload := strings.Repeat("2", 64)
	table := filepath.Join(t.TempDir(), "candidates.tsv")
	row := strings.Join([]string{goos, goarch, manifest, payload, payload, "gosource-v1",
		"go version go1.27.2 " + goos + "/" + goarch, "GOTOOLCHAIN=go1.27.2 make build-bashy",
		"bashy=" + strings.Repeat("a", 40) + ";sh=" + strings.Repeat("b", 40)}, "\t") + "\n"
	if err := os.WriteFile(table, []byte(row), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := reviewedCandidate(table, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := reviewedCandidate(table, strings.Repeat("3", 64)); err == nil {
		t.Fatal("unknown manifest accepted")
	}
}

func TestDanglingSplitPayloadNeverFallsBack(t *testing.T) {
	launcher := filepath.Join(t.TempDir(), "bashy")
	bytes := []byte("native executable")
	if err := os.WriteFile(launcher, bytes, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("absent", launcher+".real"); err != nil {
		t.Fatal(err)
	}
	if got := candidatePayloadPath(launcher); got != launcher+".real" {
		t.Fatal("dangling payload fell back to launcher")
	}
	if _, err := authenticateFile(candidatePayloadPath(launcher), fmt.Sprintf("%x", sha256.Sum256(bytes))); err == nil {
		t.Fatal("dangling payload accepted")
	}
}
