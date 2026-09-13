// Copyright 2026 The bashpp-tests Authors. All rights reserved.
// Sprint: #150; Story: S150.8; Story-ID: 65db485f62ab
// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// package-verify checks the evidence of the package packet: for every
// authenticated package, in the requested mode, cmd/go enumerated a non-zero
// set of original test bodies (from its own generated _testmain.go), the
// backend handed exactly the tested package, its test files and the external
// test package to Bash++ as a package map with _testmain.go as the program
// (every file once, under cmd/go's own role for it; the program under
// cmd/go's testmain identity with the TestMain fact, the library under the
// tested package's own identity without it), the native test binary was
// never the executed argv, and the terminal is upstream's. A PASS
// additionally needs one test2json terminal per enumerated test. A package
// build, an empty enumeration, or a native run can never be PASS; a failed
// Bash++ run is a retained product failure.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const schema = "bashpp-tests/upstream-gotest-backend/v1"

type planRecord struct {
	Schema      string   `json:"schema"`
	Kind        string   `json:"kind"`
	Package     string   `json:"package"`
	Mode        string   `json:"mode"`
	NativeArgv  []string `json:"native_argv"`
	Argv        []string `json:"argv"`
	Testmain    string   `json:"testmain"`
	Disposition string   `json:"disposition"`
	ImportPath  string   `json:"import_path"`
	TestMain    bool     `json:"test_main"`
	Overlay     string   `json:"overlay"`
	// TranspileStatus is the file the compiled script writes the library
	// transpile's exit status to before anything else runs.
	TranspileStatus string   `json:"transpile_status"`
	Artifacts       []string `json:"artifacts"`
	Deviations      []string `json:"deviations"`
	ProgramArgv     []string `json:"program_argv"`
	Enumeration     struct {
		Tests, Benchmarks, Examples int
		FuzzTargets                 int `json:"fuzz_targets"`
	} `json:"enumeration"`
	Program struct {
		Path     string `json:"path"`
		Files    []string
		Packages []struct {
			Path  string
			Files []string
			Roles map[string]string
		}
	} `json:"program"`
	// Library is the compiled route's one Bash++ library invocation: the
	// identity the tested package's files are checked under and the role of
	// every file it was handed (S165.0, D8).
	Library struct {
		ImportPath string `json:"import_path"`
		TestMain   bool   `json:"test_main"`
		Roles      map[string]string
	} `json:"library"`
	Tool struct{ Path, Version string } `json:"tool"`
}

type overlayProof struct {
	Schema  string    `json:"schema"`
	Kind    string    `json:"kind"`
	Package string    `json:"package"`
	GoTool  string    `json:"go_tool"`
	Overlay fileProof `json:"overlay"`
	Trace   fileProof `json:"compile_trace"`
	Files   []struct {
		Original  string `json:"original"`
		Generated string `json:"generated"`
		SHA256    string `json:"sha256"`
	} `json:"files"`
	CompilerArgv []string `json:"compiler_argv"`
}

type fileProof struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type goEvent struct {
	Action, Package, Test, Output string
}

func main() {
	matrix := flag.String("matrix", "", "authenticated package matrix TSV")
	dir := flag.String("evidence", "", "per-mode evidence directory")
	mode := flag.String("mode", "", "interpreted or compiled")
	version := flag.String("version", "", "pinned Bash++ version")
	tool := flag.String("tool", "", "pinned Bash++ path")
	flag.Parse()

	f, err := os.Open(*matrix)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	bad, product := false, false
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 5 {
			fatal(fmt.Errorf("invalid matrix row %q", line))
		}
		capability, pkg := fields[0], fields[1]
		status, err := verify(pkg, *dir, *mode, *version, *tool)
		if err != nil {
			bad = true
			fmt.Printf("FAIL %-40s %-34s %v\n", capability, pkg, err)
			continue
		}
		if strings.HasSuffix(status, "-PRODUCT-FAIL") {
			product = true
		}
		fmt.Printf("%-20s %-40s %s\n", status, capability, pkg)
	}
	if err := s.Err(); err != nil {
		fatal(err)
	}
	if bad {
		os.Exit(1)
	}
	if product {
		fmt.Println("NON-GREEN: honest Bash++ product failures are retained in this packet")
		os.Exit(3)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "package-verify:", err)
	os.Exit(2)
}

func verify(pkg, dir, mode, version, tool string) (string, error) {
	base := filepath.Join(dir, strings.NewReplacer("/", "_", ".", "_").Replace(pkg))
	action, terminals, err := goTest(base+".go-test.json", pkg)
	if err != nil {
		return "", err
	}
	plans, proofs, err := readPlans(base + ".events.jsonl")
	if err != nil {
		return "", err
	}
	if len(plans) != 1 {
		return "", fmt.Errorf("wanted exactly one backend plan for %s, got %d", pkg, len(plans))
	}
	p := plans[0]
	if p.Schema != schema || p.Kind != "plan" || p.Package != pkg || p.Mode != mode || p.Tool.Path != tool || p.Tool.Version != version {
		return "", fmt.Errorf("backend plan identity is incomplete")
	}
	if len(p.Deviations) == 0 {
		return "", fmt.Errorf("backend deviations are not explicit")
	}
	if len(p.NativeArgv) == 0 || filepath.Base(p.NativeArgv[0]) != filepath.Base(pkg)+".test" {
		return "", fmt.Errorf("native argv %v is not the built test binary cmd/go would have run", p.NativeArgv)
	}
	if p.Disposition == "unsupported" {
		if action != "fail" {
			return "", fmt.Errorf("unsupported plan with upstream action %s", action)
		}
		declared := false
		for _, d := range p.Deviations {
			if strings.Contains(d, "non-Go inputs") {
				declared = true
			}
		}
		if !declared {
			return "", fmt.Errorf("plan unsupported without a non-Go-input reason: %v", p.Deviations)
		}
		return "PACKAGE-PRODUCT-FAIL", nil
	}
	want := map[string]string{"interpreted": "run-package-map", "compiled": "transpile-overlay-go-test"}[mode]
	if want == "" {
		return "", fmt.Errorf("unknown backend mode %q", mode)
	}
	if p.Disposition != want {
		return "", fmt.Errorf("disposition = %s, want %s", p.Disposition, want)
	}
	bodies := p.Enumeration.Tests + p.Enumeration.Benchmarks + p.Enumeration.Examples + p.Enumeration.FuzzTargets
	if bodies == 0 {
		// Tests, benchmarks, examples and fuzz targets are all original test
		// bodies in Go's own metadata (reflectdata carries only benchmarks).
		return "", fmt.Errorf("cmd/go enumerated no original test bodies for %s; an empty enumeration is never a Bash++ result", pkg)
	}
	if filepath.Base(p.Testmain) != "_testmain.go" || len(p.Program.Files) != 1 || p.Program.Files[0] != p.Testmain || p.Program.Path != pkg+".test" {
		return "", fmt.Errorf("program is not cmd/go's generated _testmain.go for %s: %+v", pkg, p.Program)
	}
	tested := false
	for _, entry := range p.Program.Packages {
		// A package whose tests are all external (cmd/internal/testdir) is
		// represented by its _test package alone, as cmd/go builds it.
		if (entry.Path == pkg || entry.Path == pkg+"_test") && len(entry.Files) != 0 {
			tested = true
		}
		if entry.Path != pkg && entry.Path != pkg+"_test" {
			return "", fmt.Errorf("program map carries a foreign package %s", entry.Path)
		}
	}
	if !tested {
		return "", fmt.Errorf("program map lacks the tested package %s or its test package", pkg)
	}
	if len(p.Argv) == 0 {
		return "", fmt.Errorf("no backend argv recorded")
	}
	if p.Argv[0] == p.NativeArgv[0] {
		return "", fmt.Errorf("the native test binary was the executed argv")
	}
	if err := verifyIdentity(p, pkg, mode, tool); err != nil {
		return "", err
	}
	if err := verifyRoles(p, pkg, mode); err != nil {
		return "", err
	}
	switch mode {
	case "interpreted":
		if p.Argv[0] != tool || !contains(p.Argv, "--go-file") {
			return "", fmt.Errorf("interpreted argv does not run Bash++ on the program: %v", p.Argv[:min(6, len(p.Argv))])
		}
	case "compiled":
		if p.Argv[0] != "/bin/sh" || !strings.Contains(strings.Join(p.Argv, " "), "transpile") {
			return "", fmt.Errorf("compiled argv does not transpile the program")
		}
		if refused, err := transpileRefused(p); err != nil {
			return "", err
		} else if refused {
			// Bash++ refused the tested sources before any overlay existed:
			// the script stops there (set -e), cmd/go never compiled the
			// originals, and the terminal is the refusal — a product row.
			if action != "fail" {
				return "", fmt.Errorf("transpile refused with upstream action %s", action)
			}
			if len(proofs) != 0 {
				return "", fmt.Errorf("transpile refused but an overlay proof was recorded")
			}
			return "PACKAGE-PRODUCT-FAIL", nil
		}
		if err := verifyOverlay(p, proofs, pkg); err != nil {
			return "", err
		}
	}
	if action == "pass" {
		if terminals < p.Enumeration.Tests+p.Enumeration.Examples {
			return "", fmt.Errorf("upstream pass with %d per-test terminals for %d enumerated tests", terminals, p.Enumeration.Tests)
		}
		return "PACKAGE-PASS", nil
	}
	return "PACKAGE-PRODUCT-FAIL", nil
}

// verifyIdentity checks the two identities of a package test (S165.0, D8),
// never confused. The PROGRAM is cmd/go's testmain package (<pkg>.test) and
// the plan asserts --go-test-main on the one Bash++ invocation that receives
// _testmain.go — the interpreted argv, directly after that identity — so the
// identity-keyed internal-visibility rule is fed the fact from the one site
// that knows it, never a name. The LIBRARY (compiled route: the transpile
// inside the /bin/sh script) is the tested package's own identity (<pkg>),
// what cmd/go compiles ptest under, so the tested package's own internal
// siblings are decided on its identity (cmd/compile → cmd/compile/internal/*)
// and never on the testmain's; the library never asserts the fact — cmd/go's
// own _testmain.go is compiled natively under the overlay. A library checked
// under the testmain identity, a fact asserted on the library, a dropped
// fact and a fact on another identity are all seam failures.
func verifyIdentity(p planRecord, pkg, mode, tool string) error {
	if p.ImportPath != pkg+".test" || !p.TestMain {
		return fmt.Errorf("plan does not declare the testmain identity %s.test with the TestMain fact: import_path=%q test_main=%v", pkg, p.ImportPath, p.TestMain)
	}
	switch mode {
	case "interpreted":
		asserted := false
		for i, arg := range p.Argv {
			if arg != "--go-import-path" || i+1 >= len(p.Argv) {
				continue
			}
			if p.Argv[i+1] != pkg+".test" {
				return fmt.Errorf("interpreted argv declares an identity other than the testmain's: --go-import-path %s", p.Argv[i+1])
			}
			if i+2 < len(p.Argv) && p.Argv[i+2] == "--go-test-main" {
				asserted = true
			}
		}
		if !asserted {
			return fmt.Errorf("interpreted argv does not carry --go-import-path %s.test --go-test-main: %v", pkg, p.Argv[:min(8, len(p.Argv))])
		}
		return nil
	case "compiled":
		if len(p.Argv) != 3 || p.Argv[0] != "/bin/sh" || p.Argv[1] != "-c" {
			return fmt.Errorf("compiled argv is not one /bin/sh -c script: %v", p.Argv[:min(3, len(p.Argv))])
		}
		if p.Library.ImportPath != pkg || p.Library.TestMain {
			return fmt.Errorf("plan does not declare the library under the tested package's own identity %s without the TestMain fact: import_path=%q test_main=%v", pkg, p.Library.ImportPath, p.Library.TestMain)
		}
		want := "'" + tool + "' 'transpile' '--bashpp' '--source=go' '--go-import-path' '" + pkg + "' '--go-library'"
		if !strings.Contains(p.Argv[2], want) {
			return fmt.Errorf("compiled transpile does not check the library under the tested package's own identity (%s)", want)
		}
		if strings.Contains(p.Argv[2], "'--go-test-main'") {
			return fmt.Errorf("compiled script asserts the TestMain fact on a library; the fact belongs to cmd/go's own testmain, which the overlay route compiles natively")
		}
		return nil
	}
	return fmt.Errorf("unknown backend mode %q", mode)
}

// verifyRoles checks that every file of the program map was handed once,
// under cmd/go's own role for it, and that the roles form cmd/go's two test
// packages: the tested package's entry (<pkg>) carries ordinary files (go)
// and in-package test files (test, a _test.go name), the external test
// package's entry (<pkg>_test) carries xtest files only — so a package whose
// tests are all external (cmd/internal/testdir) is one xtest entry and
// nothing else, and the external test package is formed from that role,
// never from a file relabelled as ordinary. In compiled mode the transpile
// must hand every file under the flag of its role (--go-file,
// --go-test-file, --go-xtest-file) with the roles recorded on the library.
func verifyRoles(p planRecord, pkg, mode string) error {
	seen := map[string]bool{}
	for _, entry := range p.Program.Packages {
		if len(entry.Roles) != len(entry.Files) {
			return fmt.Errorf("package %s records %d roles for %d files", entry.Path, len(entry.Roles), len(entry.Files))
		}
		for _, file := range entry.Files {
			if seen[file] {
				return fmt.Errorf("file %s was handed more than once", file)
			}
			seen[file] = true
			role, ok := entry.Roles[file]
			if !ok {
				return fmt.Errorf("file %s of %s has no recorded role", file, entry.Path)
			}
			isTestName := strings.HasSuffix(file, "_test.go")
			switch {
			case entry.Path == pkg+"_test" && role != "xtest":
				return fmt.Errorf("file %s of the external test package %s has role %s, want xtest", file, entry.Path, role)
			case entry.Path == pkg && role == "xtest":
				return fmt.Errorf("file %s of the tested package %s has role xtest; an external test file never forms the tested package", file, entry.Path)
			case role == "go" && isTestName:
				return fmt.Errorf("test file %s was handed as an ordinary file of %s", file, entry.Path)
			case (role == "test" || role == "xtest") && !isTestName:
				return fmt.Errorf("file %s of %s has role %s but is not a _test.go file", file, entry.Path, role)
			case role != "go" && role != "test" && role != "xtest":
				return fmt.Errorf("file %s of %s has unknown role %q", file, entry.Path, role)
			}
			if mode == "compiled" {
				if p.Library.Roles[file] != role {
					return fmt.Errorf("library records role %q for %s, the program map %q", p.Library.Roles[file], file, role)
				}
				flag := "--go-file"
				if role != "go" {
					flag = "--go-" + role + "-file"
				}
				if !strings.Contains(p.Argv[2], "'"+flag+"' '"+file+"'") {
					return fmt.Errorf("compiled transpile does not hand %s as %s", file, flag)
				}
			}
		}
	}
	if mode == "compiled" && len(p.Library.Roles) != len(seen) {
		return fmt.Errorf("library records %d roles for %d files", len(p.Library.Roles), len(seen))
	}
	return nil
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// goTest returns the package terminal action and the number of per-test
// terminal actions test2json reported.
func goTest(name, pkg string) (action string, terminals int, err error) {
	f, err := os.Open(name)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 1<<20), 1<<28)
	for s.Scan() {
		var ev goEvent
		if json.Unmarshal(s.Bytes(), &ev) != nil || ev.Package != pkg {
			continue
		}
		switch ev.Action {
		case "pass", "fail", "skip":
			if ev.Test == "" {
				action = ev.Action
			} else if !strings.Contains(ev.Test, "/") {
				terminals++
			}
		}
	}
	if action == "" {
		return "", 0, fmt.Errorf("missing go test terminal action for %s", pkg)
	}
	return action, terminals, s.Err()
}

func readPlans(name string) ([]planRecord, []overlayProof, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	var plans []planRecord
	var proofs []overlayProof
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 1<<20), 1<<28)
	for s.Scan() {
		// The kind decides the record shape: a plan's "overlay" is the path
		// string, a proof's is {path, sha256}. Decoding every line as a plan
		// first rejected every event log that reached the proof — i.e. every
		// compiled package whose transpile succeeded (pre-existing since the
		// overlay route, fb3f24e; pinned by TestReadPlansSeparatesProofsFromPlans).
		var head struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(s.Bytes(), &head); err != nil {
			return nil, nil, err
		}
		switch head.Kind {
		case "plan":
			var p planRecord
			if err := json.Unmarshal(s.Bytes(), &p); err != nil {
				return nil, nil, err
			}
			plans = append(plans, p)
		case "overlay-proof":
			var proof overlayProof
			if err := json.Unmarshal(s.Bytes(), &proof); err != nil {
				return nil, nil, err
			}
			proofs = append(proofs, proof)
		}
	}
	return plans, proofs, s.Err()
}

func verifyOverlay(p planRecord, proofs []overlayProof, pkg string) error {
	if len(proofs) != 1 {
		return fmt.Errorf("wanted exactly one overlay proof, got %d", len(proofs))
	}
	proof := proofs[0]
	if proof.Schema != schema || proof.Kind != "overlay-proof" || proof.Package != pkg || proof.GoTool == "" || proof.Overlay.Path != p.Overlay {
		return fmt.Errorf("overlay proof identity is incomplete")
	}
	if len(proof.CompilerArgv) != 4 || proof.CompilerArgv[0] != proof.GoTool || proof.CompilerArgv[1] != "test" || proof.CompilerArgv[2] != "-overlay="+p.Overlay || proof.CompilerArgv[3] != pkg {
		return fmt.Errorf("overlay proof does not record the pinned go test argv")
	}
	data, err := os.ReadFile(proof.Overlay.Path)
	if err != nil {
		return fmt.Errorf("read overlay JSON: %w", err)
	}
	if digest(data) != proof.Overlay.SHA256 {
		return fmt.Errorf("overlay JSON digest does not match recorded proof")
	}
	trace, err := os.ReadFile(proof.Trace.Path)
	if err != nil || digest(trace) != proof.Trace.SHA256 {
		return fmt.Errorf("go test -n compile trace does not match recorded proof")
	}
	var overlay struct {
		Replace map[string]string `json:"Replace"`
	}
	if err := json.Unmarshal(data, &overlay); err != nil || len(overlay.Replace) == 0 {
		return fmt.Errorf("invalid overlay JSON")
	}
	if len(proof.Files) != len(p.Artifacts) || len(proof.Files) != len(overlay.Replace) {
		return fmt.Errorf("overlay proof does not cover every generated file")
	}
	seenOriginal, seenGenerated := map[string]bool{}, map[string]bool{}
	for _, file := range proof.Files {
		if file.Original == "" || file.Generated == "" || len(file.SHA256) != 64 || overlay.Replace[file.Original] != file.Generated || seenOriginal[file.Original] || seenGenerated[file.Generated] || !contains(p.Artifacts, file.Generated) {
			return fmt.Errorf("overlay proof is not a one-to-one original/generated mapping")
		}
		data, err := os.ReadFile(file.Generated)
		if err != nil || digest(data) != file.SHA256 {
			return fmt.Errorf("generated file digest does not match recorded proof")
		}
		seenOriginal[file.Original], seenGenerated[file.Generated] = true, true
		if traceArg(trace, file.Original) || !traceArg(trace, file.Generated) {
			return fmt.Errorf("compile trace does not prove generated file replaces %s", file.Original)
		}
	}
	for _, entry := range p.Program.Packages {
		for _, original := range entry.Files {
			if !seenOriginal[original] {
				return fmt.Errorf("package source %s was not mapped by overlay", original)
			}
		}
	}
	return nil
}

// transpileRefused reads the exit status the compiled script recorded for
// the library transpile: non-zero is Bash++'s refusal of the tested sources.
// A missing status file means the script never ran that far — a seam
// defect, not a product row.
func transpileRefused(p planRecord) (bool, error) {
	if p.TranspileStatus == "" {
		return false, fmt.Errorf("plan records no transpile status file")
	}
	data, err := os.ReadFile(p.TranspileStatus)
	if err != nil {
		return false, fmt.Errorf("read transpile status: %w", err)
	}
	status := strings.TrimSpace(string(data))
	if status == "" {
		return false, fmt.Errorf("transpile status %s is empty", p.TranspileStatus)
	}
	for _, c := range status {
		if c < '0' || c > '9' {
			return false, fmt.Errorf("transpile status %q is not an exit status", status)
		}
	}
	return status != "0", nil
}

func traceArg(trace []byte, want string) bool {
	for _, field := range strings.Fields(string(trace)) {
		if strings.Trim(field, "'\"") == want {
			return true
		}
	}
	return false
}

func digest(data []byte) string {
	// sha256 is intentionally kept here rather than trusting a shell's report.
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
