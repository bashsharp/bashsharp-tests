//go:build !windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func tourCandidateFixture(t *testing.T) (string, map[string]any) {
	t.Helper()
	dir := t.TempDir()
	repo := filepath.Join(dir, "source")
	if err := os.Mkdir(repo, 0755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "core.hooksPath=/dev/null", "commit", "-q", "--allow-empty", "-m", "fixture"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	rev, ok := gitOutput(repo, "rev-parse", "HEAD")
	if !ok {
		t.Fatal("fixture revision unavailable")
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "bashy")
	if err := os.WriteFile(bin, data, 0755); err != nil {
		t.Fatal(err)
	}
	return bin, map[string]any{"launcher_sha256": shaFile(bin), "payload_sha256": shaFile(bin), "frontend_version": "gosource-v1", "build_recipe": "authored fixture; never executed", "repositories": []any{map[string]any{"path": repo, "commit": strings.TrimSpace(rev)}}}
}

func TestTourNativeCandidateAuthentication(t *testing.T) {
	bin, manifest := tourCandidateFixture(t)
	accepted := func() {
		t.Helper()
		if _, err := AuthenticateCandidate(bin, manifest); err != nil {
			t.Fatal(err)
		}
	}
	rejected := func(label string) {
		t.Helper()
		if _, err := AuthenticateCandidate(bin, manifest); err == nil {
			t.Fatalf("%s accepted", label)
		}
	}
	write := func(path string, data []byte) {
		t.Helper()
		if err := os.WriteFile(path, data, 0755); err != nil {
			t.Fatal(err)
		}
	}
	remove := func(path string) {
		t.Helper()
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
	accepted()
	original := manifest["payload_sha256"]
	manifest["payload_sha256"] = strings.Repeat("0", 64)
	rejected("wrong payload hash")
	manifest["payload_sha256"] = original
	manifest["launcher_sha256"] = strings.Repeat("0", 64)
	rejected("wrong launcher hash")
	manifest["launcher_sha256"] = original
	repo := asMap(asList(manifest["repositories"])[0])
	rev := repo["commit"]
	repo["commit"] = strings.Repeat("a", 40)
	rejected("wrong repository revision")
	repo["commit"] = rev
	dirty := filepath.Join(asString(repo["path"]), "untracked")
	write(dirty, []byte("dirty"))
	rejected("dirty repository")
	remove(dirty)
	if err := os.Symlink("missing", bin+".real"); err != nil {
		t.Fatal(err)
	}
	rejected("dangling split payload")
	remove(bin + ".real")
	accepted()
	if err := os.Chmod(bin, 0644); err != nil {
		t.Fatal(err)
	}
	rejected("non-executable native payload")
	write(bin, []byte("#!/bin/sh\nexit 0\n"))
	if err := os.Chmod(bin, 0755); err != nil {
		t.Fatal(err)
	}
	rejected("tampered executable")
	manifest["launcher_sha256"], manifest["payload_sha256"] = shaFile(bin), shaFile(bin)
	rejected("shell launcher without Go payload")
	remove(bin)
	rejected("deleted executable")
}

func TestTourSplitCandidateAuthentication(t *testing.T) {
	bin, manifest := tourCandidateFixture(t)
	if err := os.WriteFile(bin+".real", []byte("historical separate payload"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest["payload_sha256"] = shaFile(bin + ".real")
	if _, err := AuthenticateCandidate(bin, manifest); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin+".real", []byte("tampered"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := AuthenticateCandidate(bin, manifest); err == nil {
		t.Fatal("tampered split payload accepted")
	}
	if err := os.Remove(bin + ".real"); err != nil {
		t.Fatal(err)
	}
	if _, err := AuthenticateCandidate(bin, manifest); err == nil {
		t.Fatal("deleted split payload accepted as launcher")
	}
}

func TestTourMigratedCandidateContract(t *testing.T) {
	root := sealTestRoot(t)
	contract := loadCandidateContract(filepath.Join(root, "docs/tour/candidate.tsv"))
	expected := strings.Fields("agent-bench appstore bashsharp bashsharp-tour bashy bonsai coreutils filebrowser gfy gotreesitter nadir outpost readline sh yamux ycode yoke")
	if len(contract) != len(expected) {
		t.Fatalf("components=%d want17", len(contract))
	}
	seen := map[string]bool{}
	for _, row := range contract {
		name := asString(row["component"])
		if seen[name] || !containsString(expected, name) {
			t.Fatalf("unexpected or duplicate component %q", name)
		}
		seen[name] = true
	}
}
