package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func managedToolTestContext(t *testing.T) *toolchainContext {
	t.Helper()
	root := filepath.Join(t.TempDir(), "goroot")
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	goBinary := filepath.Join(root, "bin", "go")
	if err := os.WriteFile(goBinary, []byte("authenticated-go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte("go1.27.1\ntime 2026-09-01T00:00:00Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return &toolchainContext{goroot: root, goBinary: goBinary, goVersion: "go1.27.1", goSHA256: sha(goBinary)}
}

func TestProvisionManagedToolCacheConstructsAuthenticatedFastPath(t *testing.T) {
	tc := managedToolTestContext(t)
	cache := filepath.Join(t.TempDir(), "managed-tools")
	got, err := provisionManagedToolCache(cache, tc)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cache, "go", "1.27.1", "go", "bin", "go")
	if got != want || sha(got) != tc.goSHA256 {
		t.Fatalf("fast path = %q (%s), want %q (%s)", got, sha(got), want, tc.goSHA256)
	}
	link := filepath.Join(cache, "go", "1.27.1", "go")
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("SDK link is not a symlink: info=%v err=%v", info, err)
	}
}

func TestProvisionManagedToolCacheRejectsUnauthenticatedSDK(t *testing.T) {
	tests := []struct {
		name string
		edit func(*testing.T, *toolchainContext)
		want string
	}{
		{"version", func(t *testing.T, tc *toolchainContext) {
			if err := os.WriteFile(filepath.Join(tc.goroot, "VERSION"), []byte("go1.27.0\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "GOROOT version changed"},
		{"digest", func(t *testing.T, tc *toolchainContext) {
			if err := os.WriteFile(tc.goBinary, []byte("different-go"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, "checksum changed"},
		{"unpinned-version", func(t *testing.T, tc *toolchainContext) { tc.goVersion = "1.27.1" }, "invalid pinned Go version"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := managedToolTestContext(t)
			test.edit(t, tc)
			_, err := provisionManagedToolCache(filepath.Join(t.TempDir(), "cache"), tc)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want marker %q", err, test.want)
			}
		})
	}
}
