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
