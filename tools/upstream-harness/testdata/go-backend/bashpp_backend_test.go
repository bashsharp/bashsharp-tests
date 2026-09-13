// Copyright 2026 The bashpp-tests Authors. All rights reserved.
// Sprint: #162; Story: S162.0; Story-ID: cda64bde8fea
// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cmd/go/internal/load"
)

func TestLibraryOverlayUsesOneInvocationAndAllFileClasses(t *testing.T) {
	dir := t.TempDir()
	files := []bashppPackageFile{
		{path: filepath.Join(dir, "a.go"), role: bashppRoleGo},
		{path: filepath.Join(dir, "a_test.go"), role: bashppRoleTest},
		{path: filepath.Join(dir, "external_test.go"), role: bashppRoleXTest},
	}
	args := bashppLibraryArgs(files)
	if want := []string{"--go-file", files[0].path, "--go-test-file", files[1].path, "--go-xtest-file", files[2].path}; !sameStrings(args, want) {
		t.Fatalf("library args = %v, want %v", args, want)
	}
	libraryDir := filepath.Join(dir, "library")
	generated, err := bashppLibraryOutputNames(libraryDir, files)
	if err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(dir, "library-output.txt")
	data := "library " + files[0].path + " -> " + generated[0] + "\n" +
		"library " + files[1].path + " -> " + generated[1] + "\n" +
		"library " + files[2].path + " -> " + generated[2] + "\n"
	if err := os.WriteFile(transcript, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	overlay := filepath.Join(dir, "overlay.json")
	script := bashppLibraryOverlayScript(transcript, overlay, []string{files[0].path, files[1].path, files[2].path}, generated)
	if err := exec.Command("/bin/sh", "-c", script).Run(); err != nil {
		t.Fatalf("library transcript validation: %v\n%s", err, script)
	}
	var got struct {
		Replace map[string]string
	}
	overlayData, err := os.ReadFile(overlay)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(overlayData, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Replace) != len(files) {
		t.Fatalf("overlay replacements = %v, want %d entries", got.Replace, len(files))
	}
	for i, file := range files {
		if got.Replace[file.path] != generated[i] {
			t.Errorf("overlay[%q] = %q, want %q", file.path, got.Replace[file.path], generated[i])
		}
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// The file roles are cmd/go's own, each file once. cmd/go's recompiled
// in-package variant (ptest) lists the test files in GoFiles AND
// TestGoFiles; the external-test variant (pxtest, <pkg>_test) lists the
// xtest files as its GoFiles (the XTestGoFiles metadata the in-package
// variant still carries is never its files); a package whose tests are all
// external (cmd/internal/testdir) has no in-package variant at all and its
// one file is an xtest file — the shape that forms the external test package.
func TestTestVariantFilesClassifyEachFileOnce(t *testing.T) {
	const tested = "cmd/compile/internal/abt"
	dir := "/src/" + tested
	ptest := &load.Package{PackagePublic: load.PackagePublic{
		ImportPath: tested, Dir: dir,
		GoFiles:      []string{"avlint32.go", "avlint32_test.go"},
		TestGoFiles:  []string{"avlint32_test.go"},
		XTestGoFiles: []string{"x_test.go"},
	}}
	pxtest := &load.Package{PackagePublic: load.PackagePublic{
		ImportPath: tested + "_test", Dir: dir,
		GoFiles: []string{"x_test.go"},
	}}
	xonly := &load.Package{PackagePublic: load.PackagePublic{
		ImportPath: "cmd/internal/testdir_test", Dir: "/src/cmd/internal/testdir",
		GoFiles: []string{"testdir_test.go"},
	}}
	if bashppRoleGo.flag() != "--go-file" || bashppRoleTest.flag() != "--go-test-file" || bashppRoleXTest.flag() != "--go-xtest-file" {
		t.Fatalf("role flags = %s %s %s", bashppRoleGo.flag(), bashppRoleTest.flag(), bashppRoleXTest.flag())
	}
	for name, tt := range map[string]struct {
		imp    *load.Package
		tested string
		want   []bashppPackageFile
	}{
		"recompiled in-package variant": {ptest, tested, []bashppPackageFile{
			{filepath.Join(dir, "avlint32.go"), bashppRoleGo},
			{filepath.Join(dir, "avlint32_test.go"), bashppRoleTest},
		}},
		"external test variant": {pxtest, tested, []bashppPackageFile{
			{filepath.Join(dir, "x_test.go"), bashppRoleXTest},
		}},
		"external tests only": {xonly, "cmd/internal/testdir", []bashppPackageFile{
			{"/src/cmd/internal/testdir/testdir_test.go", bashppRoleXTest},
		}},
	} {
		t.Run(name, func(t *testing.T) {
			got := bashppTestVariantFiles(tt.imp, tt.tested)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("files = %v, want %v", got, tt.want)
			}
			args := bashppLibraryArgs(got)
			for i, file := range tt.want {
				if args[2*i] != file.role.flag() || args[2*i+1] != file.path {
					t.Fatalf("library args = %v, want %s %s at %d", args, file.role.flag(), file.path, i)
				}
			}
			roles := bashppFileRoles(got)
			if len(roles) != len(tt.want) {
				t.Fatalf("roles = %v, want one per file", roles)
			}
			for _, file := range tt.want {
				if roles[file.path] != string(file.role) {
					t.Fatalf("roles[%s] = %q, want %q", file.path, roles[file.path], file.role)
				}
			}
		})
	}
}

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
// Two identities, never confused: the program's is cmd/go's testmain
// package (<pkg>.test) with the TestMain fact; the library's is the tested
// package's own (<pkg>) and never asserts the fact — cmd/compile's own
// cmd/compile/internal/* imports are granted to cmd/compile, not to
// cmd/compile.test.
func TestIdentitiesSeparateTheLibraryFromTheTestMain(t *testing.T) {
	p := &load.Package{PackagePublic: load.PackagePublic{ImportPath: "cmd/compile", Name: "main"}}
	pmain := &load.Package{PackagePublic: load.PackagePublic{ImportPath: "cmd/compile.test", Name: "main"}}
	program, library := bashppProgramIdentity(pmain), bashppLibraryIdentity(p)
	if want := []string{"--go-import-path", "cmd/compile.test", "--go-test-main"}; !sameStrings(program.args(), want) {
		t.Fatalf("program identity = %v, want %v", program.args(), want)
	}
	if want := []string{"--go-import-path", "cmd/compile"}; !sameStrings(library.args(), want) {
		t.Fatalf("library identity = %v, want %v", library.args(), want)
	}
	if strings.Contains(strings.Join(library.args(), " "), "--go-test-main") || library.testMain {
		t.Fatalf("the library asserted the TestMain fact: %v", library.args())
	}
	if program.importPath == library.importPath {
		t.Fatalf("the program and the library share an identity: %s", program.importPath)
	}
}
