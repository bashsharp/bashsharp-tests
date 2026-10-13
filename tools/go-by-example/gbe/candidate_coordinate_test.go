// Sprint: #434; Story: Separate authenticated build toolchain from fixed corpus oracle; Story-ID: 30f81908446b
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewedSecurityBuildCoordinate(t *testing.T) {
	goos, goarch := hostIdentity()
	oracle := &Toolchain{Version: "go1.27.1", Identity: "go version go1.27.1 " + goos + "/" + goarch}
	row := &Reviewed{Fields: map[string]string{
		"go_identity":  "go version go1.27.2 " + goos + "/" + goarch,
		"build_recipe": "GOTOOLCHAIN=go1.27.2 make build-bashy",
	}}
	if err := authenticateBuildCoordinate(row, oracle); err != nil {
		t.Fatalf("reviewed Go1.27.2 build against Go1.27.1 oracle: %v", err)
	}
	for name, edit := range map[string]func(){
		"unreviewed patch": func() {
			row.Fields["go_identity"] = strings.ReplaceAll(row.Fields["go_identity"], "go1.27.2", "go1.27.3")
			row.Fields["build_recipe"] = "GOTOOLCHAIN=go1.27.3 make build-bashy"
		},
		"forged recipe":  func() { row.Fields["build_recipe"] = "GOTOOLCHAIN=go1.27.1 make build-bashy" },
		"wrong platform": func() { row.Fields["go_identity"] = "go version go1.27.2 fake/arch" },
	} {
		t.Run(name, func(t *testing.T) {
			originalIdentity, originalRecipe := row.Fields["go_identity"], row.Fields["build_recipe"]
			defer func() { row.Fields["go_identity"], row.Fields["build_recipe"] = originalIdentity, originalRecipe }()
			edit()
			if err := authenticateBuildCoordinate(row, oracle); err == nil {
				t.Fatal("accepted mismatched build coordinate")
			}
		})
	}
}

func TestReviewedSecurityBuildUsesIndependentPin(t *testing.T) {
	goos, goarch := hostIdentity()
	oracle := &Toolchain{Version: "go1.27.1", Identity: "go version go1.27.1 " + goos + "/" + goarch, GoSHA256: strings.Repeat("1", 64)}
	row := &Reviewed{Fields: map[string]string{
		"go_identity":  "go version go1.27.2 " + goos + "/" + goarch,
		"build_recipe": "GOTOOLCHAIN=go1.27.2 make build-bashy",
	}}
	wantGo := strings.Repeat("2", 64)
	wantArchive := strings.Repeat("3", 64)
	table := filepath.Join(t.TempDir(), "build-toolchains.tsv")
	line := strings.Join([]string{goos, goarch, "go1.27.2", row.Fields["go_identity"], wantGo,
		"go1.27.2." + goos + "-" + goarch + ".archive", wantArchive,
		"https://go.dev/dl/?mode=json&include=all"}, "\t") + "\n"
	if err := os.WriteFile(table, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := reviewedBuildToolchainAt(table, row, oracle)
	if err != nil {
		t.Fatal(err)
	}
	if got.GoSHA256 != wantGo || got.DistributionSHA256 != wantArchive {
		t.Fatalf("build SDK pin = %#v, want independently reviewed checksums", got)
	}
}

func TestAuthenticatedSDKRejectsSameVersionNativeShapedWrongBytes(t *testing.T) {
	goroot := filepath.Join(t.TempDir(), "goroot")
	if err := os.MkdirAll(filepath.Join(goroot, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	wrong := make([]byte, 64)
	copy(wrong, []byte("\x7fELF"))
	wrong[4] = 2
	goBinary := goExecutable(goroot)
	if err := os.WriteFile(goBinary, wrong, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goroot, "VERSION"), []byte("go1.27.2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pin := &Toolchain{Version: "go1.27.2", GoSHA256: strings.Repeat("a", 64)}
	err := authenticateToolchainRoot(goroot, pin)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("error = %v, want same-version native-shaped wrong bytes rejected by reviewed checksum", err)
	}
}

func TestSecurityBuildPayloadMetadata(t *testing.T) {
	goos, goarch := hostIdentity()
	module := filepath.Join(os.Getenv("HOME"), "go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.2."+goos+"-"+goarch, "bin", "go")
	if goos == "windows" {
		module += ".exe"
	}
	if _, err := os.Stat(module); err != nil {
		t.Skip("Go1.27.2 SDK is not installed")
	}
	row := &Reviewed{Fields: map[string]string{"go_identity": "go version go1.27.2 " + goos + "/" + goarch}}
	oracle := &Toolchain{Version: "go1.27.1", Identity: "go version go1.27.1 " + goos + "/" + goarch}
	row.Fields["build_recipe"] = "GOTOOLCHAIN=go1.27.2 make build-bashy"
	if err := authenticateBuildCoordinate(row, oracle); err != nil {
		t.Fatalf("reviewed build against unchanged oracle: %v", err)
	}
	if err := authenticateBuildPayload(module, row); err != nil {
		t.Fatalf("real Go1.27.2 binary: %v", err)
	}
	row.Fields["go_identity"] = "go version go1.27.1 " + goos + "/" + goarch
	if err := authenticateBuildPayload(module, row); err == nil {
		t.Fatal("accepted stale reviewed build identity")
	}
	if err := authenticateBuildPayload(filepath.Join(t.TempDir(), "missing"), row); err == nil {
		t.Fatal("accepted missing build metadata")
	}
}
