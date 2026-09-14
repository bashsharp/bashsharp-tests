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
		Roles map[string]string
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
// The two identities (D8): the plan must declare cmd/go's testmain identity
// (<pkg>.test) with the TestMain fact and assert --go-test-main on the one
// Bash++ invocation that receives _testmain.go (the interpreted argv); the
// compiled transpile checks the library under the tested package's OWN
// identity (<pkg>) and never asserts the fact. The four interpreted
// negatives are the D8 set; the compiled negatives are the library's.
func TestVerifyIdentityRequiresTheTestMainFact(t *testing.T) {
	pkg, tool := "cmd/compile/internal/abt", "/pinned/bashy"
	interpreted := planRecord{ImportPath: pkg + ".test", TestMain: true, Argv: []string{tool, "--bashpp", "--source=go", "--go-import-path", pkg + ".test", "--go-test-main", "--go-package", pkg + "=/src/a.go", "--go-file", "/tmp/_testmain.go"}}
	if err := verifyIdentity(interpreted, pkg, "interpreted", tool); err != nil {
		t.Fatalf("interpreted: %v", err)
	}
	script := "set -e\n'" + tool + "' 'transpile' '--bashpp' '--source=go' '--go-import-path' '" + pkg + "' '--go-library' '/tmp/lib' '--go-file' '/src/a.go' > '/tmp/out'\nexec env -u BASHPP_GOTEST_BACKEND '/pinned/go' 'test' '-overlay=/tmp/overlay.json' '" + pkg + "'"
	compiled := planRecord{ImportPath: pkg + ".test", TestMain: true, Argv: []string{"/bin/sh", "-c", script}}
	compiled.Library.ImportPath = pkg
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
		"flag on another identity": {"interpreted", func(p *planRecord) { p.Argv[4] = pkg }, "identity other than the testmain's"},
		"second identity on the program": {"interpreted", func(p *planRecord) {
			p.Argv = append(p.Argv, "--go-import-path", pkg)
		}, "identity other than the testmain's"},
		"library under the testmain identity": {"compiled", func(p *planRecord) {
			p.Library.ImportPath = pkg + ".test"
		}, "tested package's own identity"},
		"fact asserted on the library": {"compiled", func(p *planRecord) {
			p.Library.TestMain = true
		}, "without the TestMain fact"},
		"script checks the library as the testmain": {"compiled", func(p *planRecord) {
			p.Argv[2] = strings.Replace(p.Argv[2], "'"+pkg+"' '--go-library'", "'"+pkg+".test' '--go-library'", 1)
		}, "tested package's own identity"},
		"script asserts the fact on the library": {"compiled", func(p *planRecord) {
			p.Argv[2] = strings.Replace(p.Argv[2], "'--go-library'", "'--go-test-main' '--go-library'", 1)
		}, "tested package's own identity"},
		"script asserts the fact after the library": {"compiled", func(p *planRecord) {
			p.Argv[2] = strings.Replace(p.Argv[2], " > '/tmp/out'", " '--go-test-main' > '/tmp/out'", 1)
		}, "asserts the TestMain fact on a library"},
		"script identity is dotted": {"compiled", func(p *planRecord) {
			p.Argv[2] = strings.Replace(p.Argv[2], "'"+pkg+"' '--go-library'", "'example.com/m' '--go-library'", 1)
		}, "tested package's own identity"},
		"not one shell script":             {"compiled", func(p *planRecord) { p.Argv = []string{tool, "transpile"} }, "one /bin/sh script"},
		"script file not the recorded one": {"compiled", func(p *planRecord) { p.Argv = []string{"/bin/sh", "/tmp/elsewhere.sh"}; p.Script = "/tmp/plan.sh" }, "one /bin/sh script"},
		"script file missing": {"compiled", func(p *planRecord) {
			p.Argv = []string{"/bin/sh", "/nonexistent/plan.sh"}
			p.Script = "/nonexistent/plan.sh"
		}, "cannot be read"},
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

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// The file roles form cmd/go's two test packages: the tested package's
// entry carries go and test files, the external test package's entry xtest
// files only, every file once with a role; a package whose tests are all
// external (cmd/internal/testdir) is one xtest entry and nothing else. In
// compiled mode the transpile hands every file under its role's flag.
func TestVerifyRolesFormTheTestPackages(t *testing.T) {
	type entry = struct {
		Path  string
		Files []string
		Roles map[string]string
	}
	pkg, dir := "cmd/compile/internal/abt", "/src/cmd/compile/internal/abt"
	full := planRecord{}
	full.Program.Packages = []entry{
		{pkg, []string{dir + "/avlint32.go", dir + "/avlint32_test.go"}, map[string]string{dir + "/avlint32.go": "go", dir + "/avlint32_test.go": "test"}},
		{pkg + "_test", []string{dir + "/x_test.go"}, map[string]string{dir + "/x_test.go": "xtest"}},
	}
	if err := verifyRoles(full, pkg, "interpreted"); err != nil {
		t.Fatalf("interpreted full shape: %v", err)
	}
	xonly := planRecord{}
	xonly.Program.Packages = []entry{{"cmd/internal/testdir_test", []string{"/src/cmd/internal/testdir/testdir_test.go"}, map[string]string{"/src/cmd/internal/testdir/testdir_test.go": "xtest"}}}
	if err := verifyRoles(xonly, "cmd/internal/testdir", "interpreted"); err != nil {
		t.Fatalf("interpreted external-tests-only shape: %v", err)
	}
	compiled := full
	compiled.Library.Roles = map[string]string{dir + "/avlint32.go": "go", dir + "/avlint32_test.go": "test", dir + "/x_test.go": "xtest"}
	compiled.Argv = []string{"/bin/sh", "-c", "'/pinned/bashy' 'transpile' '--bashpp' '--source=go' '--go-import-path' '" + pkg + "' '--go-library' '/tmp/lib' '--go-file' '" + dir + "/avlint32.go' '--go-test-file' '" + dir + "/avlint32_test.go' '--go-xtest-file' '" + dir + "/x_test.go' > '/tmp/out'"}
	if err := verifyRoles(compiled, pkg, "compiled"); err != nil {
		t.Fatalf("compiled full shape: %v", err)
	}
	for name, tt := range map[string]struct {
		mode   string
		mutate func(p *planRecord)
		want   string
	}{
		"test file relabelled as ordinary": {"interpreted", func(p *planRecord) {
			p.Program.Packages[0].Roles[dir+"/avlint32_test.go"] = "go"
		}, "handed as an ordinary file"},
		"external test file in the tested package": {"interpreted", func(p *planRecord) {
			p.Program.Packages[0].Files = append(p.Program.Packages[0].Files, dir+"/y_test.go")
			p.Program.Packages[0].Roles[dir+"/y_test.go"] = "xtest"
		}, "never forms the tested package"},
		"ordinary file in the external test package": {"interpreted", func(p *planRecord) {
			p.Program.Packages[1].Roles[dir+"/x_test.go"] = "go"
		}, "want xtest"},
		"role without a _test.go name": {"interpreted", func(p *planRecord) {
			p.Program.Packages[0].Roles[dir+"/avlint32.go"] = "test"
		}, "not a _test.go file"},
		"file handed twice": {"interpreted", func(p *planRecord) {
			p.Program.Packages[1].Files = append(p.Program.Packages[1].Files, dir+"/avlint32_test.go")
			p.Program.Packages[1].Roles[dir+"/avlint32_test.go"] = "xtest"
		}, "more than once"},
		"file without a role": {"interpreted", func(p *planRecord) {
			p.Program.Packages[0].Files = append(p.Program.Packages[0].Files, dir+"/z.go")
			p.Program.Packages[0].Roles[dir+"/z.go"] = "go"
			p.Program.Packages[0].Roles[dir+"/w.go"] = "go"
			p.Program.Packages[0].Files = p.Program.Packages[0].Files[:2]
		}, "roles for"},
		"transpile hands the xtest file as ordinary": {"compiled", func(p *planRecord) {
			p.Argv[2] = strings.Replace(p.Argv[2], "'--go-xtest-file'", "'--go-file'", 1)
		}, "does not hand"},
		"library role disagrees with the map": {"compiled", func(p *planRecord) {
			p.Library.Roles[dir+"/x_test.go"] = "test"
		}, "library records role"},
	} {
		t.Run(name, func(t *testing.T) {
			p := cloneRoles(full)
			if tt.mode == "compiled" {
				p = cloneRoles(compiled)
			}
			tt.mutate(&p)
			if err := verifyRoles(p, pkg, tt.mode); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

// cloneRoles deep-copies the package entries, their role maps and the
// library's, so one negative's mutation never leaks into the next.
func cloneRoles(p planRecord) planRecord {
	c := p
	c.Argv = append([]string(nil), p.Argv...)
	c.Program.Packages = nil
	for _, e := range p.Program.Packages {
		roles := map[string]string{}
		for k, v := range e.Roles {
			roles[k] = v
		}
		c.Program.Packages = append(c.Program.Packages, struct {
			Path  string
			Files []string
			Roles map[string]string
		}{e.Path, append([]string(nil), e.Files...), roles})
	}
	c.Library.Roles = map[string]string{}
	for k, v := range p.Library.Roles {
		c.Library.Roles[k] = v
	}
	return c
}

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// An event log of the compiled route carries the plan AND the overlay proof
// the /bin/sh script appends once the transpile succeeded; the proof's
// "overlay" is an object where the plan's is a string, so the reader must
// decode each line by its kind (the pre-existing defect made every compiled
// package that reached its proof unverifiable).
func TestReadPlansSeparatesProofsFromPlans(t *testing.T) {
	name := filepath.Join(t.TempDir(), "events.jsonl")
	lines := `{"schema":"` + schema + `","kind":"plan","package":"example/tiny","mode":"compiled","overlay":"/tmp/overlay.json","argv":["/bin/sh","-c","x"]}
{"schema":"` + schema + `","kind":"overlay-proof","package":"example/tiny","go_tool":"/pinned/go","overlay":{"path":"/tmp/overlay.json","sha256":"00"},"compile_trace":{"path":"/tmp/t","sha256":"00"},"files":[],"compiler_argv":["/pinned/go","test","-overlay=/tmp/overlay.json","example/tiny"]}
`
	if err := os.WriteFile(name, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
	plans, proofs, err := readPlans(name)
	if err != nil {
		t.Fatalf("readPlans: %v", err)
	}
	if len(plans) != 1 || plans[0].Overlay != "/tmp/overlay.json" || len(proofs) != 1 || proofs[0].Overlay.Path != "/tmp/overlay.json" {
		t.Fatalf("plans = %+v, proofs = %+v", plans, proofs)
	}
}

// TestVerifyIdentityReadsTheScriptFile pins the large-package form of the
// compiled plan (the Sprint 171 leaf's `fork/exec /bin/sh: argument list
// too long` on ssa, types2 and go/types): the hook writes the script into
// the persisted overlay directory and hands `/bin/sh <path>`; the verifier
// checks the same text it would have checked inline, so every identity and
// role rule still applies to it.
func TestVerifyIdentityReadsTheScriptFile(t *testing.T) {
	const pkg = "example.com/m/avlint32"
	const tool = "/pinned/bashy"
	dir := t.TempDir()
	body := "set -e\n'" + tool + "' 'transpile' '--bashpp' '--source=go' '--go-import-path' '" + pkg + "' '--go-library' '/tmp/lib' '--go-file' '" + dir + "/avlint32.go' > '/tmp/out'\n"
	script := filepath.Join(dir, "plan.sh")
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	p := planRecord{ImportPath: pkg + ".test", TestMain: true, Argv: []string{"/bin/sh", script}, Script: script}
	p.Library.ImportPath = pkg
	p.Library.Roles = map[string]string{dir + "/avlint32.go": "go"}
	p.Program.Packages = []struct {
		Path  string
		Files []string
		Roles map[string]string
	}{{pkg, []string{dir + "/avlint32.go"}, map[string]string{dir + "/avlint32.go": "go"}}}
	if err := verifyIdentity(p, pkg, "compiled", tool); err != nil {
		t.Fatalf("file-form identity: %v", err)
	}
	if err := verifyRoles(p, pkg, "compiled"); err != nil {
		t.Fatalf("file-form roles: %v", err)
	}
	// The same negatives the inline form has: the wrong identity in the file
	// is refused from the file's text, not from the argv.
	if err := os.WriteFile(script, []byte(strings.Replace(body, "'"+pkg+"' '--go-library'", "'"+pkg+".test' '--go-library'", 1)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := verifyIdentity(p, pkg, "compiled", tool); err == nil || !strings.Contains(err.Error(), "tested package's own identity") {
		t.Fatalf("file-form wrong identity: %v", err)
	}
}
