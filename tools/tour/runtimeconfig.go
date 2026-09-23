package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// provisionManagedGo constructs yoke/binmgr's already-installed Go fast path.
// The cache is executor-owned and shared by all three modes, while the target
// remains the exact SDK authenticated by identity and digest before this call.
func provisionManagedGo(cache, goroot, version, digest string) (string, error) {
	release := strings.TrimPrefix(version, "go")
	if release == "" || release == version {
		return "", fmt.Errorf("invalid pinned Go version %q", version)
	}
	goBinary := goExecutable(goroot)
	if got := shaFile(goBinary); got != digest {
		return "", fmt.Errorf("authenticated Go binary checksum changed: got %s", got)
	}
	versionBytes, err := os.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil || strings.TrimSpace(firstLine(string(versionBytes))) != version {
		return "", fmt.Errorf("authenticated GOROOT version changed")
	}
	link := filepath.Join(cache, "go", release, "go")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return "", err
	}
	if err := linkSDK(link, goroot); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(link)
	want, wantErr := filepath.EvalSymlinks(goroot)
	if err != nil || wantErr != nil || resolved != want {
		return "", fmt.Errorf("managed Go SDK target mismatch")
	}
	managedGo := goExecutable(link)
	if got := shaFile(managedGo); got != digest {
		return "", fmt.Errorf("managed Go binary checksum mismatch: got %s", got)
	}
	return managedGo, nil
}
