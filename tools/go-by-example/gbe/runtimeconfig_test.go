package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func managedToolTestContext(t *testing.T) *toolchainContext {
	return managedToolTestContextVersion(t, "go1.27.1")
}

func managedToolTestContextVersion(t *testing.T, version string) *toolchainContext {
	t.Helper()
	root := filepath.Join(t.TempDir(), "goroot")
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	goBinary := filepath.Join(root, "bin", "go")
	if err := os.WriteFile(goBinary, []byte("authenticated-go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte(version+"\ntime 2026-09-01T00:00:00Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return &toolchainContext{goroot: root, goBinary: goBinary, goVersion: version, goSHA256: sha(goBinary)}
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

func TestProvisionManagedToolCachesPreparesSecurityBuildBeforeBaseline(t *testing.T) {
	oracle := managedToolTestContextVersion(t, "go1.27.1")
	build := managedToolTestContextVersion(t, "go1.27.2")
	cache := filepath.Join(t.TempDir(), "managed-tools")
	if err := provisionManagedToolCaches(cache, oracle, build); err != nil {
		t.Fatal(err)
	}
	baseline, err := snapshot(cache)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"1.27.1", "1.27.2"} {
		path := filepath.Join(cache, "go", version, "go", "bin", "go")
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("SDK %s was not prepared before baseline: %v", version, err)
		}
	}
	after, err := snapshot(cache)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(baseline, after) {
		t.Fatal("prepared SDK cache mutated after baseline")
	}
}

func TestProvisionManagedToolCachesRejectsForgedSecurityBuildSource(t *testing.T) {
	build := managedToolTestContextVersion(t, "go1.27.2")
	if err := os.WriteFile(filepath.Join(build.goroot, "VERSION"), []byte("go1.27.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := provisionManagedToolCaches(filepath.Join(t.TempDir(), "managed-tools"), build)
	if err == nil || !strings.Contains(err.Error(), "GOROOT version changed") {
		t.Fatalf("error = %v, want forged SDK identity rejection", err)
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

func TestInitializeManagedToolCaches(t *testing.T) {
	for _, scenario := range []string{"success", "failed-init", "bad-installed-binary", "bad-installed-version", "bad-original", "runner-error"} {
		t.Run(scenario, func(t *testing.T) {
			tc := managedToolTestContext(t)
			cache := filepath.Join(t.TempDir(), "managed-tools")
			if err := provisionManagedToolCaches(cache, tc); err != nil {
				t.Fatal(err)
			}
			if scenario == "bad-original" {
				if err := os.WriteFile(tc.goBinary, []byte("tampered"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			runner := func(argv []string, cwd, prefix string, env map[string]string, timeout Number, stdin string) (*Object, error) {
				calls++
				if !reflect.DeepEqual(argv, []string{"candidate", "go", "version"}) || env["BASHY_GO_VERSION"] != "1.27.1" || env["BASHY_BIN_CACHE"] != cache || timeout.AsFloat() != 180 {
					t.Fatalf("bad initialization contract: %v %v", argv, env)
				}
				if scenario == "runner-error" {
					return nil, os.ErrPermission
				}
				code := 0
				if scenario == "failed-init" {
					code = 1
				}
				// Model producer publication without modifying the authenticated source.
				root := filepath.Join(cache, "go", "1.27.1", "go")
				if err := os.Remove(root); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(root, "bin"), 0755); err != nil {
					t.Fatal(err)
				}
				binary := []byte("authenticated-go")
				version := []byte("go1.27.1\n")
				if scenario == "bad-installed-binary" {
					binary = []byte("forged")
				}
				if scenario == "bad-installed-version" {
					version = []byte("go1.27.2\n")
				}
				if err := os.WriteFile(goExecutable(root), binary, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "VERSION"), version, 0644); err != nil {
					t.Fatal(err)
				}
				return Obj("spawned", true, "state", "exited", "exit", Int(int64(code)), "signal", nil), nil
			}
			logs := filepath.Join(t.TempDir(), "init")
			err := initializeManagedToolCaches("candidate", cache, logs, map[string]string{"HOME": "isolated"}, runner, tc, tc)
			if (err == nil) != (scenario == "success") {
				t.Fatalf("error = %v", err)
			}
			if scenario == "bad-original" {
				if calls != 0 {
					t.Fatal("executed with unauthenticated original SDK")
				}
				return
			}
			if calls != 1 {
				t.Fatalf("initializations = %d", calls)
			}
			data, readErr := os.ReadFile(filepath.Join(logs, "1.27.1.initialization.json"))
			if readErr != nil || !strings.Contains(string(data), "changed_paths") {
				t.Fatalf("missing evidence: %s %v", data, readErr)
			}
		})
	}
}
