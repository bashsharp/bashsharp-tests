// Copyright 2026 The bashpp-tests Authors. All rights reserved.
// Sprint: #150; Story: S150.8; Story-ID: 65db485f62ab
// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// Direct Bash++ backend for Go's own package-test runner (cmd/go). The
// unmodified cmd/go enumerates the tests (load.TestPackagesFor generates
// _testmain.go from Go's own test metadata) and builds the native test
// binary; the one patched site in test.go asks this file, right before the
// test binary would be executed, for the Bash++ program that IS that test
// binary: the package under test with its in-package test files, the
// external test package if any, and the generated _testmain.go — handed to
// Bash++ as an explicit package map — with the exact test argv. The native
// test binary is never run while the backend is selected.
package test

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"cmd/go/internal/load"
	"cmd/go/internal/work"
)

const bashppGoTestSchema = "bashpp-tests/upstream-gotest-backend/v1"

var bashppEventMu sync.Mutex

func bashppEmit(record map[string]any) {
	path := os.Getenv("BASHPP_GOTEST_EVENTS")
	if path == "" {
		return
	}
	record["schema"] = bashppGoTestSchema
	line, err := json.Marshal(record)
	if err != nil {
		return
	}
	bashppEventMu.Lock()
	defer bashppEventMu.Unlock()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(line, '\n'))
}

// bashppTestCount counts the tests, benchmarks, fuzz targets and examples in
// the _testmain.go cmd/go generated: Go's own enumeration, read back from
// Go's own artifact. It never looks at the tested sources.
func bashppTestCount(testmain string) (tests, benchmarks, fuzz, examples int, err error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, testmain, nil, 0)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || len(value.Values) != 1 {
				continue
			}
			lit, ok := value.Values[0].(*ast.CompositeLit)
			if !ok {
				continue
			}
			switch value.Names[0].Name {
			case "tests":
				tests = len(lit.Elts)
			case "benchmarks":
				benchmarks = len(lit.Elts)
			case "fuzzTargets":
				fuzz = len(lit.Elts)
			case "examples":
				examples = len(lit.Elts)
			}
		}
	}
	return tests, benchmarks, fuzz, examples, nil
}

// bashppFileRole is cmd/go's own classification of one Go file of a package
// test: an ordinary file of the tested package (GoFiles), an in-package test
// file compiled into the tested package's recompiled variant (TestGoFiles),
// or a file of the external test package (XTestGoFiles). The role is what
// the file IS to cmd/go; it decides the library flag it is handed under and
// which of the two library units it forms, never the identity.
type bashppFileRole string

const (
	bashppRoleGo    bashppFileRole = "go"
	bashppRoleTest  bashppFileRole = "test"
	bashppRoleXTest bashppFileRole = "xtest"
)

// flag is the library flag of the role: --go-file, --go-test-file,
// --go-xtest-file (bashy transpile --go-library).
func (role bashppFileRole) flag() string {
	if role == bashppRoleGo {
		return "--go-file"
	}
	return "--go-" + string(role) + "-file"
}

type bashppPackageFile struct {
	path string
	role bashppFileRole
}

// bashppIdentity is the declared identity of one Bash++ invocation: the
// import path it is checked under (the compiler's -p) and whether it is
// cmd/go's generated test main. cmd/go builds a package test as three
// compile units, each under its own identity: the tested package (ptest,
// -p <pkg>, the in-package test files merged in), the external test package
// (pxtest, -p <pkg>_test) and the generated testmain (pmain, -p <pkg>.test,
// whose importer-stack label "testmain" is what exempts it from the internal
// rule: load/pkg.go disallowInternal). The library transpile compiles the
// first two as one library under the tested package's OWN identity — the
// external test package is formed from the xtest role — so the tested
// package sees its own internal siblings exactly as cmd/go grants them to
// <pkg>; it never asserts TestMain. The testmain identity and the TestMain
// fact go only to the invocation that receives _testmain.go.
type bashppIdentity struct {
	importPath string
	testMain   bool
}

func (id bashppIdentity) args() []string {
	args := []string{"--go-import-path", id.importPath}
	if id.testMain {
		args = append(args, "--go-test-main")
	}
	return args
}

// bashppProgramIdentity is the generated testmain's: cmd/go's own path for
// it (load/test.go: p.ImportPath + ".test") with the TestMain fact, asserted
// explicitly (--go-test-main, S165.0 for D8) rather than inferred from the
// ".test" suffix, at the one site that knows the program it runs IS that
// testmain.
func bashppProgramIdentity(pmain *load.Package) bashppIdentity {
	return bashppIdentity{importPath: pmain.ImportPath, testMain: true}
}

// bashppLibraryIdentity is the tested package's own: what cmd/go compiles
// ptest under (-p <pkg>). A library is never the test main.
func bashppLibraryIdentity(p *load.Package) bashppIdentity {
	return bashppIdentity{importPath: p.ImportPath, testMain: false}
}

// bashppTestVariantFiles classifies the files of one of cmd/go's test
// variants of the tested package by role, each path exactly once. The
// recompiled in-package variant (ptest) lists its test files in GoFiles AND
// TestGoFiles, and the external-test variant (pxtest, <path>_test) lists the
// xtest files as its GoFiles, so a plain GoFiles/TestGoFiles/XTestGoFiles
// walk double-counts and misfiles them (and a duplicated file is refused by
// the front end). Each path is classified once: xtest package → xtest; a
// file named in TestGoFiles → test; everything else → go. The in-package
// variant's XTestGoFiles (copied from the original package's metadata) are
// NOT its files — cmd/go compiles them into pxtest only — so they never
// form the tested package. A package whose tests are all external
// (cmd/internal/testdir) has no in-package unit at all: its files are xtest
// files and nothing else, which is how the external test package is formed
// honestly.
func bashppTestVariantFiles(imp *load.Package, tested string) []bashppPackageFile {
	files := make([]bashppPackageFile, 0, len(imp.GoFiles)+len(imp.TestGoFiles)+len(imp.XTestGoFiles))
	seen := map[string]bool{}
	add := func(name string, role bashppFileRole) {
		path := filepath.Join(imp.Dir, name)
		if !seen[path] {
			seen[path] = true
			files = append(files, bashppPackageFile{path, role})
		}
	}
	if imp.ImportPath == tested+"_test" {
		for _, name := range imp.GoFiles {
			add(name, bashppRoleXTest)
		}
		for _, name := range imp.XTestGoFiles {
			add(name, bashppRoleXTest)
		}
		return files
	}
	isTest := make(map[string]bool, len(imp.TestGoFiles))
	for _, name := range imp.TestGoFiles {
		isTest[name] = true
	}
	for _, name := range imp.GoFiles {
		if isTest[name] {
			add(name, bashppRoleTest)
		} else {
			add(name, bashppRoleGo)
		}
	}
	for _, name := range imp.TestGoFiles {
		add(name, bashppRoleTest)
	}
	return files
}

func bashppFilePaths(files []bashppPackageFile) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.path)
	}
	return paths
}

func bashppFileRoles(files []bashppPackageFile) map[string]string {
	roles := make(map[string]string, len(files))
	for _, file := range files {
		roles[file.path] = string(file.role)
	}
	return roles
}

func bashppLibraryArgs(files []bashppPackageFile) []string {
	args := make([]string, 0, len(files)*2)
	for _, file := range files {
		args = append(args, file.role.flag(), file.path)
	}
	return args
}

func bashppLibraryOutputNames(dir string, files []bashppPackageFile) ([]string, error) {
	generated := make([]string, 0, len(files))
	seen := make(map[string]string, len(files))
	for _, file := range files {
		name := filepath.Join(dir, filepath.Base(file.path))
		if previous, ok := seen[name]; ok {
			return nil, fmt.Errorf("library output collision for %s and %s", previous, file.path)
		}
		seen[name] = file.path
		generated = append(generated, name)
	}
	return generated, nil
}

func bashppQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func bashppOverlayProof(eventFile, pkg, overlay, trace, goTool string, originals, generated []string) string {
	lines := []string{
		// sha256 prints the digest WITHOUT a newline: it is spliced into one
		// JSON line. The whole event is assembled in a scratch file and appended
		// to the shared event log in a single write, so packages cmd/go tests in
		// parallel never interleave their lines.
		"sha256() { if command -v sha256sum >/dev/null 2>&1; then printf '%s' \"$(sha256sum \"$1\" | awk '{print $1}')\"; else printf '%s' \"$(shasum -a 256 \"$1\" | awk '{print $1}')\"; fi; }",
		"{",
		"printf '%s' " + bashppQuote(`{"schema":"`+bashppGoTestSchema+`","kind":"overlay-proof","package":"`+pkg+`","go_tool":"`+goTool+`","overlay":{"path":"`+overlay+`","sha256":"`),
		"sha256 " + bashppQuote(overlay),
		"printf '%s' " + bashppQuote(`"},"compile_trace":{"path":"`+trace+`","sha256":"`),
		"sha256 " + bashppQuote(trace),
		"printf '%s' " + bashppQuote(`"},"files":[`),
	}
	for i := range originals {
		if i != 0 {
			lines = append(lines, "printf ','")
		}
		lines = append(lines,
			"printf '%s' "+bashppQuote(`{"original":"`+originals[i]+`","generated":"`+generated[i]+`","sha256":"`),
			"sha256 "+bashppQuote(generated[i]),
			"printf '%s' "+bashppQuote(`"}`))
	}
	lines = append(lines,
		"printf '%s\\n' "+bashppQuote(`],"compiler_argv":["`+goTool+`","test","-overlay=`+overlay+`","`+pkg+`"]}`),
		"} > "+bashppQuote(overlay+".event"),
		"cat "+bashppQuote(overlay+".event")+" >> "+bashppQuote(eventFile))
	return strings.Join(lines, "\n")
}

func bashppLibraryOverlayScript(transcript, overlay string, originals, generated []string) string {
	lines := []string{"set -e"}
	for _, original := range originals {
		lines = append(lines, "awk -v o="+bashppQuote(original)+" '$1 == \"library\" && $2 == o && $3 == \"->\" { n++ } END { exit n == 1 ? 0 : 1 }' "+bashppQuote(transcript))
	}
	lines = append(lines, "awk '")
	lines = append(lines, `function esc(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); return s }`)
	// cmd/go's overlay file is {"Replace":{original:generated,…}}; the
	// pairs below are printed comma-separated and END closes both braces.
	lines = append(lines, `BEGIN { printf "{\"Replace\":{" }`)
	lines = append(lines, "/^library[[:space:]]/ { if ($1 != \"library\" || $3 != \"->\" || NF != 4) exit 1; if (n++) printf \",\"; printf \"\\\"%s\\\":\\\"%s\\\"\", esc($2), esc($4) }")
	lines = append(lines, "END { if (n != "+strconv.Itoa(len(originals))+") exit 1; printf \"}}\\n\" }")
	lines = append(lines, "' "+bashppQuote(transcript)+" > "+bashppQuote(overlay))
	for i, original := range originals {
		lines = append(lines, "test \"$(awk -v o="+bashppQuote(original)+" '$1 == \"library\" && $2 == o && $3 == \"->\" { print $4 }' "+bashppQuote(transcript)+")\" = "+bashppQuote(generated[i]))
	}
	return strings.Join(lines, "\n")
}

// bashppTestPlan returns the argv that replaces the native test binary, or
// nil when the backend is not selected. args[0] is the built test binary
// (or an exec wrapper); args[1:] are the exact test flags cmd/go chose.
func bashppTestPlan(p *load.Package, buildAction *work.Action, args []string) []string {
	mode := os.Getenv("BASHPP_GOTEST_BACKEND")
	if mode == "" {
		return nil
	}
	tool := os.Getenv("BASHPP_GOTEST_TOOL")
	pmain := buildAction.Package
	testmain := filepath.Join(pmain.Dir, "_testmain.go")
	record := map[string]any{
		"kind":        "plan",
		"package":     p.ImportPath,
		"mode":        mode,
		"native_argv": append([]string(nil), args...),
		"testmain":    testmain,
		"tool":        map[string]any{"path": tool, "version": os.Getenv("BASHPP_GOTEST_VERSION")},
	}
	tests, benchmarks, fuzz, examples, err := bashppTestCount(testmain)
	if err != nil {
		record["disposition"] = "unsupported"
		record["deviations"] = []string{"cmd/go's generated _testmain.go could not be read: " + err.Error()}
		bashppEmit(record)
		return []string{"/bin/sh", "-c", "echo " + bashppQuote("Bash++ gotest backend: "+err.Error()) + " >&2; exit 1"}
	}
	record["enumeration"] = map[string]any{"tests": tests, "benchmarks": benchmarks, "fuzz_targets": fuzz, "examples": examples}

	// The program: every package pmain imports that cmd/go built for this
	// test — the package under test (with its in-package test files merged
	// by load.TestPackagesFor) and the external test package — in import
	// order, then _testmain.go as the main package. Every file is handed
	// once, under the role cmd/go gave it.
	var mapArgs []string
	var packages []map[string]any
	var classified []bashppPackageFile
	var assemblyCompanions []string
	for _, imp := range pmain.Internal.Imports {
		if imp.ImportPath != p.ImportPath && imp.ImportPath != p.ImportPath+"_test" {
			continue
		}
		if len(imp.CgoFiles) != 0 || (mode != "compiled" && len(imp.SFiles) != 0) {
			record["disposition"] = "unsupported"
			record["deviations"] = []string{fmt.Sprintf("package %s has non-Go inputs %v; no direct Go-source meaning", imp.ImportPath, append(append([]string(nil), imp.SFiles...), imp.CgoFiles...))}
			bashppEmit(record)
			return []string{"/bin/sh", "-c", "echo 'Bash++ gotest backend: non-Go inputs' >&2; exit 1"}
		}
		assemblyCompanions = append(assemblyCompanions, imp.SFiles...)
		variant := bashppTestVariantFiles(imp, p.ImportPath)
		classified = append(classified, variant...)
		files := bashppFilePaths(variant)
		mapArgs = append(mapArgs, "--go-package", imp.ImportPath+"="+strings.Join(files, ","))
		packages = append(packages, map[string]any{"path": imp.ImportPath, "files": files, "roles": bashppFileRoles(variant)})
	}
	record["program"] = map[string]any{"path": pmain.ImportPath, "packages": packages, "files": []string{testmain}}
	testArgs := args[1:]
	record["program_argv"] = testArgs
	// Two identities, never confused (S165.0, D8): the program's is cmd/go's
	// own testmain package (<pkg>.test) with the TestMain fact, handed to the
	// one Bash++ invocation that receives _testmain.go; the library's is the
	// tested package's own (<pkg>), what cmd/go compiles ptest under, so the
	// tested package's internal imports are decided on ITS identity — cmd/go
	// grants cmd/compile its cmd/compile/internal/* on the directory, never
	// on the testmain's path — and the library never asserts the fact.
	program := bashppProgramIdentity(pmain)
	library := bashppLibraryIdentity(p)
	record["import_path"] = program.importPath
	record["test_main"] = program.testMain
	deviations := []string{
		"cmd/go enumerated the tests and built the native test binary; the binary is never executed while the backend is selected",
		"the tested package, its in-package test files and the external test package are handed to Bash++ as an explicit package map with the generated _testmain.go as the main package; every file once, under cmd/go's own role for it",
		"the program's declared identity is cmd/go's testmain package (<pkg>.test) and the backend asserts the TestMain fact (--go-test-main) only on the invocation that receives _testmain.go; the tested package's files are checked under the tested package's own identity (<pkg>)",
	}
	var plan []string
	switch mode {
	case "interpreted":
		plan = append(append([]string{tool, "--bashpp", "--source=go"}, program.args()...), mapArgs...)
		plan = append(plan, "--go-file", testmain)
		if len(testArgs) != 0 {
			plan = append(plan, "--")
			plan = append(plan, testArgs...)
		}
		record["disposition"] = "run-package-map"
	case "compiled":
		goTool := os.Getenv("BASHPP_GOTEST_GO")
		// The overlay directory is evidence (the generated files cmd/go
		// compiled instead of the originals, the overlay, the -n trace, the
		// proof) and must outlive the run: cmd/go deletes its $WORK (pmain.Dir)
		// when the test ends, which left the verifier nothing to read for any
		// package whose transpile succeeded. It lives beside the event log.
		overlayDir := filepath.Join(pmain.Dir, "bashpp-overlay")
		if events := os.Getenv("BASHPP_GOTEST_EVENTS"); events != "" {
			overlayDir = events + ".overlay"
		}
		if err := os.RemoveAll(overlayDir); err != nil {
			record["disposition"] = "configuration-error"
			record["deviations"] = append(deviations, "could not clear the overlay directory: "+err.Error())
			bashppEmit(record)
			return []string{"/bin/sh", "-c", "exit 1"}
		}
		if err := os.MkdirAll(overlayDir, 0o700); err != nil {
			record["disposition"] = "configuration-error"
			record["deviations"] = append(deviations, "could not create overlay directory: "+err.Error())
			bashppEmit(record)
			return []string{"/bin/sh", "-c", "exit 1"}
		}
		overlay := filepath.Join(overlayDir, "overlay.json")
		trace := filepath.Join(overlayDir, "go-test-n.trace")
		libraryDir := filepath.Join(overlayDir, "library")
		if err := os.MkdirAll(libraryDir, 0o700); err != nil {
			record["disposition"] = "configuration-error"
			record["deviations"] = append(deviations, "could not create library output directory: "+err.Error())
			bashppEmit(record)
			return []string{"/bin/sh", "-c", "exit 1"}
		}
		// The overlay replaces exactly the classified originals, each once.
		overlayOriginals := bashppFilePaths(classified)
		generated, err := bashppLibraryOutputNames(libraryDir, classified)
		if err != nil {
			record["disposition"] = "configuration-error"
			record["deviations"] = append(deviations, err.Error())
			bashppEmit(record)
			return []string{"/bin/sh", "-c", "exit 1"}
		}
		libraryArgs := bashppLibraryArgs(classified)
		transpile := append(append(append([]string{tool, "transpile", "--bashpp", "--source=go"}, library.args()...), "--go-library", libraryDir), libraryArgs...)
		record["library"] = map[string]any{"import_path": library.importPath, "test_main": library.testMain, "roles": bashppFileRoles(classified)}
		transpileWords := make([]string, len(transpile))
		for i, word := range transpile {
			transpileWords[i] = bashppQuote(word)
		}
		transcript := filepath.Join(overlayDir, "library-output.txt")
		// The transpile's exit status is evidence too: a refusal by Bash++
		// (a diagnostic on the tested sources) is a product row, while a
		// clean transpile that still produced no overlay proof is a seam
		// defect; the verifier tells them apart by this file. Its stderr
		// stays on cmd/go's captured output, the upstream terminal.
		status := filepath.Join(overlayDir, "transpile.status")
		lines := []string{
			"status=0",
			strings.Join(transpileWords, " ") + " > " + bashppQuote(transcript) + " || status=$?",
			"printf '%s\\n' \"$status\" > " + bashppQuote(status),
			"test \"$status\" -eq 0",
			bashppLibraryOverlayScript(transcript, overlay, overlayOriginals, generated),
		}
		record["transpile_status"] = status
		goTest := append([]string{"env", "-u", "BASHPP_GOTEST_BACKEND", goTool, "test", "-overlay=" + overlay, p.ImportPath}, testArgs...)
		words := make([]string, len(goTest))
		for j, word := range goTest {
			words[j] = bashppQuote(word)
		}
		dryRun := append([]string{"env", "-u", "BASHPP_GOTEST_BACKEND", goTool, "test", "-n", "-overlay=" + overlay, p.ImportPath}, testArgs...)
		dryWords := make([]string, len(dryRun))
		for j, word := range dryRun {
			dryWords[j] = bashppQuote(word)
		}
		// go test -n prints the commands it would run to stderr; the trace
		// captures both streams so it proves which files the compiler was
		// handed (stdout of a dry run is empty).
		lines = append(lines, strings.Join(dryWords, " ")+" > "+bashppQuote(trace)+" 2>&1")
		lines = append(lines, bashppOverlayProof(os.Getenv("BASHPP_GOTEST_EVENTS"), p.ImportPath, overlay, trace, goTool, overlayOriginals, generated))
		lines = append(lines, "exec "+strings.Join(words, " "))
		plan = []string{"/bin/sh", "-c", "set -e\n" + strings.Join(lines, "\n")}
		record["artifacts"] = generated
		record["overlay"] = overlay
		record["disposition"] = "transpile-overlay-go-test"
		deviations = append(deviations,
			"one transpiler invocation under the tested package's own identity (--go-import-path <pkg>, no TestMain fact) classifies GoFiles, TestGoFiles and XTestGoFiles with their corresponding library flags; every reported library output is mapped by cmd/go -overlay",
			"cmd/go's original _testmain.go (its own <pkg>.test, compiled natively under the overlay) enumerates and runs the tests; cmd/go retains internal-import policy and assembles any .s companion natively")
		if len(assemblyCompanions) != 0 {
			deviations = append(deviations, "D3(b): cmd/go assembles the package's .s test companions natively under the overlay; they are an authority compiler-artifact step, never Bash++ tested-source execution")
		}
	default:
		record["disposition"] = "configuration-error"
		bashppEmit(record)
		return []string{"/bin/sh", "-c", "echo 'Bash++ gotest backend: unknown mode' >&2; exit 1"}
	}
	record["argv"] = plan
	record["deviations"] = deviations
	bashppEmit(record)
	return plan
}
