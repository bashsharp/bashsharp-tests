package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureSDK(t *testing.T) (string, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "sdk")
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
	return root, shaFile(goBinary)
}

func TestProvisionManagedGoConstructsAuthenticatedCache(t *testing.T) {
	root, digest := fixtureSDK(t)
	cache := filepath.Join(t.TempDir(), "managed-tools")
	got, err := provisionManagedGo(cache, root, "go1.27.1", digest)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cache, "go", "1.27.1", "go", "bin", "go")
	if got != want || shaFile(got) != digest {
		t.Fatalf("managed Go = %q, want authenticated %q", got, want)
	}
	if info, err := os.Lstat(filepath.Dir(filepath.Dir(got))); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("SDK cache entry is not a symlink: info=%v err=%v", info, err)
	}
}

func TestProvisionManagedGoRejectsChangedAuthentication(t *testing.T) {
	for _, test := range []struct {
		name, want string
		change     func(string)
	}{
		{"digest", "checksum changed", func(root string) { _ = os.WriteFile(filepath.Join(root, "bin", "go"), []byte("changed"), 0o755) }},
		{"version", "GOROOT version changed", func(root string) { _ = os.WriteFile(filepath.Join(root, "VERSION"), []byte("go1.27.0\n"), 0o644) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, digest := fixtureSDK(t)
			test.change(root)
			_, err := provisionManagedGo(filepath.Join(t.TempDir(), "cache"), root, "go1.27.1", digest)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}
