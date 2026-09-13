// Copyright 2026 The bashpp-tests Authors. All rights reserved.
// Sprint: #162; Story: S162.0; Story-ID: cda64bde8fea
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixture has two package files because an overlay that maps only a
// convenient entry file can otherwise look plausible while cmd/go compiles an
// original sibling natively.
func TestOverlayProofRequiresEveryPackageFile(t *testing.T) {
	dir := t.TempDir()
	originals := []string{filepath.Join(dir, "one.go"), filepath.Join(dir, "two_test.go")}
	generated := []string{filepath.Join(dir, "one.generated.go"), filepath.Join(dir, "two_test.generated.go")}
	for i := range originals {
		if err := os.WriteFile(originals[i], []byte("package tiny\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(generated[i], []byte("// generated\npackage tiny\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	overlayPath := filepath.Join(dir, "overlay.json")
	overlayData, err := json.Marshal(struct {
		Replace map[string]string `json:"Replace"`
	}{Replace: map[string]string{originals[0]: generated[0], originals[1]: generated[1]}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overlayPath, overlayData, 0o600); err != nil {
		t.Fatal(err)
	}
	tracePath := filepath.Join(dir, "go-test-n.trace")
	trace := []byte(strings.Join(generated, "\n"))
	if err := os.WriteFile(tracePath, trace, 0o600); err != nil {
		t.Fatal(err)
	}
	p := planRecord{Package: "example/tiny", Overlay: overlayPath, Artifacts: generated}
	p.Program.Packages = []struct {
		Path  string
		Files []string
	}{{Path: "example/tiny", Files: originals}}
	proof := overlayProof{Schema: schema, Kind: "overlay-proof", Package: p.Package, GoTool: "/pinned/go", Overlay: fileProof{Path: overlayPath, SHA256: digest(overlayData)}, Trace: fileProof{Path: tracePath, SHA256: digest(trace)}, CompilerArgv: []string{"/pinned/go", "test", "-overlay=" + overlayPath, p.Package}}
	for i := range originals {
		data, err := os.ReadFile(generated[i])
		if err != nil {
			t.Fatal(err)
		}
		proof.Files = append(proof.Files, struct {
			Original  string `json:"original"`
			Generated string `json:"generated"`
			SHA256    string `json:"sha256"`
		}{originals[i], generated[i], digest(data)})
	}
	if err := verifyOverlay(p, []overlayProof{proof}, p.Package); err != nil {
		t.Fatalf("verifyOverlay: %v", err)
	}
	proof.Files = proof.Files[:1]
	if err := verifyOverlay(p, []overlayProof{proof}, p.Package); err == nil || !strings.Contains(err.Error(), "cover every generated") {
		t.Fatalf("missing sibling error = %v", err)
	}
}

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// The TestMain fact (D8): the plan must declare cmd/go's testmain identity
// (<pkg>.test) and assert --go-test-main on the Bash++ invocation, in the
// interpreted argv and inside the compiled transpile script alike.
func TestVerifyIdentityRequiresTheTestMainFact(t *testing.T) {
	pkg, tool := "cmd/compile/internal/abt", "/pinned/bashy"
	interpreted := planRecord{ImportPath: pkg + ".test", TestMain: true, Argv: []string{tool, "--bashpp", "--source=go", "--go-import-path", pkg + ".test", "--go-test-main", "--go-package", pkg + "=/src/a.go", "--go-file", "/tmp/_testmain.go"}}
	if err := verifyIdentity(interpreted, pkg, "interpreted", tool); err != nil {
		t.Fatalf("interpreted: %v", err)
	}
	script := "set -e\n'" + tool + "' 'transpile' '--bashpp' '--source=go' '--go-import-path' '" + pkg + ".test' '--go-test-main' '--go-library' '/tmp/lib' '--go-file' '/src/a.go' > '/tmp/out'\nexec env -u BASHPP_GOTEST_BACKEND '/pinned/go' 'test' '-overlay=/tmp/overlay.json' '" + pkg + "'"
	compiled := planRecord{ImportPath: pkg + ".test", TestMain: true, Argv: []string{"/bin/sh", "-c", script}}
	if err := verifyIdentity(compiled, pkg, "compiled", tool); err != nil {
		t.Fatalf("compiled: %v", err)
	}
	for name, tt := range map[string]struct {
		mode   string
		mutate func(p *planRecord)
		want   string
	}{
		"fact not declared":        {"interpreted", func(p *planRecord) { p.TestMain = false }, "TestMain fact"},
		"identity is the package":  {"interpreted", func(p *planRecord) { p.ImportPath = pkg }, "testmain identity"},
		"flag missing from argv":   {"interpreted", func(p *planRecord) { p.Argv = append(p.Argv[:5:5], p.Argv[6:]...) }, "--go-test-main"},
		"flag on another identity": {"interpreted", func(p *planRecord) { p.Argv[4] = pkg }, "--go-test-main"},
		"flag missing from script": {"compiled", func(p *planRecord) { p.Argv[2] = strings.Replace(p.Argv[2], " '--go-test-main'", "", 1) }, "testmain identity and fact"},
		"script identity is dotted": {"compiled", func(p *planRecord) {
			p.Argv[2] = strings.Replace(p.Argv[2], "'"+pkg+".test'", "'example.com/m.test'", 1)
		}, "testmain identity and fact"},
		"not one shell script": {"compiled", func(p *planRecord) { p.Argv = []string{tool, "transpile"} }, "one /bin/sh -c script"},
	} {
		t.Run(name, func(t *testing.T) {
			p := interpreted
			if tt.mode == "compiled" {
				p = compiled
			}
			p.Argv = append([]string(nil), p.Argv...)
			tt.mutate(&p)
			if err := verifyIdentity(p, pkg, tt.mode, tool); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
