// Copyright 2026 The bashpp-tests Authors. All rights reserved.
// Sprint: #149; Story: S149.4; Story-ID: 60134b3f734f
// Sprint: #154; Story: S154.0; Story-ID: 4877afd3a207
// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifierRejectsStructuredMismatch(t *testing.T) {
	for _, field := range []string{"action", "recipe_flags", "native_argv"} {
		t.Run(field, func(t *testing.T) {
			phase, backend, result := compileEvidence("interpreted")
			switch field {
			case "action":
				backend.Action = "build"
			case "recipe_flags":
				backend.RecipeFlags = []string{"-l"}
			case "native_argv":
				backend.NativeArgv = []string{"bashy", "--check"}
			}
			_, err := verifyCompileEvidence(t, "interpreted", phase, backend, result)
			if err == nil || !strings.Contains(err.Error(), "action, flags, native argv") {
				t.Fatalf("verifyRow error = %v, want structured mismatch", err)
			}
		})
	}
}

func TestVerifierRejectsMissingCompileProof(t *testing.T) {
	phase, backend, result := compileEvidence("compiled")
	result.MapProof = nil
	_, err := verifyCompileEvidence(t, "compiled", phase, backend, result)
	if err == nil || !strings.Contains(err.Error(), "existence and hash proof") {
		t.Fatalf("verifyRow error = %v, want missing proof", err)
	}
}

func TestVerifierAcceptsCompileProof(t *testing.T) {
	phase, backend, result := compileEvidence("compiled")
	status, err := verifyCompileEvidence(t, "compiled", phase, backend, result)
	if err != nil || status != "COMPILE-ONLY-PASS" {
		t.Fatalf("verifyRow = %q, %v", status, err)
	}
}

// Sprint: #249; Story: #715; Story-ID: 90f96d4f4dae
func TestVerifierRequiresRunInDirCompanionEvidence(t *testing.T) {
	phase, backend, result := runInDirEvidence("compiled")
	status, err := verifyRunInDirEvidence(t, "compiled", "pass", phase, backend, result)
	if err != nil || status != "RUNINDIR-PASS" {
		t.Fatalf("verifyRow = %q, %v", status, err)
	}
	backend.PackageMap.CompanionProof = nil
	if _, err := verifyRunInDirEvidence(t, "compiled", "pass", phase, backend, result); err == nil || !strings.Contains(err.Error(), "companion path and hash evidence") {
		t.Fatalf("verifyRow missing proof error = %v", err)
	}
	backend.PackageMap.CompanionProof = []companionFile{{Path: backend.PackageMap.Companions[0], SHA256: strings.Repeat("z", 64)}}
	if _, err := verifyRunInDirEvidence(t, "compiled", "pass", phase, backend, result); err == nil || !strings.Contains(err.Error(), "companion path and hash evidence") {
		t.Fatalf("verifyRow tampered proof error = %v", err)
	}
}

func runInDirEvidence(mode string) (eventRecord, eventRecord, eventRecord) {
	cwd := "/tmp/upstream/asmrun.dir"
	native := []string{"go", "run", "."}
	companion := cwd + "/f_amd64.s"
	phase := eventRecord{Kind: "phase", Test: "fixedbugs/asmrun.go", Action: "runindir", PhaseKind: "execute", CompileInputs: []string{"."}, ProgramArgv: []string{}, Argv: native, Cwd: cwd}
	backend := eventRecord{
		Kind: "backend", Test: phase.Test, BackendSchema: backendSchema, Mode: mode, Tool: toolIdentity{Path: "/bin/bashy", Version: "test"},
		Action: phase.Action, Phase: phase.PhaseKind, CompileInputs: phase.CompileInputs, ProgramArgv: phase.ProgramArgv, NativeArgv: native,
		Disposition: "check-then-run", Deviations: []string{"structured evidence"},
		PackageMap: &packageMap{GoList: []string{"go", "list", "-json", "-deps", "."}, Dir: cwd, Files: []string{cwd + "/main.go"}, Companions: []string{companion}, CompanionProof: []companionFile{{Path: companion, SHA256: strings.Repeat("a", 64)}}},
	}
	if mode == "compiled" {
		backend.Disposition = "transpile-build-run"
	}
	result := eventRecord{Kind: "phase_result", Test: phase.Test, Exit: 0}
	return phase, backend, result
}

func verifyRunInDirEvidence(t *testing.T, mode, goAction string, records ...eventRecord) (string, error) {
	t.Helper()
	test := "fixedbugs/asmrun.go"
	dir := t.TempDir()
	base := filepath.Join(dir, strings.NewReplacer("/", "_", ".", "_").Replace(test))
	writeJSONLines(t, base+".go-test.json", goRecord{Action: goAction, Test: "Test/" + test})
	items := make([]any, 0, len(records)+1)
	for _, record := range records {
		items = append(items, record)
	}
	items = append(items, eventRecord{Kind: "terminal", Test: test, Failed: goAction == "fail"})
	writeJSONLines(t, base+".events.jsonl", items...)
	return verifyRow(matrixRow{Test: test, Action: "runindir"}, dir, mode, "test", "/bin/bashy")
}

func TestVerifierRetainsCompileProductFailure(t *testing.T) {
	for _, mode := range []string{"interpreted", "compiled"} {
		t.Run(mode, func(t *testing.T) {
			phase, backend, result := compileEvidence(mode)
			result.Exit = 2
			status, err := verifyCompileEvidenceAction(t, mode, "fail", phase, backend, result)
			if err != nil || status != "COMPILE-PRODUCT-FAIL" {
				t.Fatalf("verifyRow = %q, %v, want retained product failure", status, err)
			}
			result.Exit = 0
			if _, err := verifyCompileEvidenceAction(t, mode, "fail", phase, backend, result); err == nil || !strings.Contains(err.Error(), "nonzero compile phase exit") {
				t.Fatalf("verifyRow error = %v, want recorded nonzero exit", err)
			}
		})
	}
}

func TestVerifierRejectsCompileExecutePhase(t *testing.T) {
	phase, backend, result := compileEvidence("interpreted")
	phase.PhaseKind, backend.Phase = "execute", "execute"
	_, err := verifyCompileEvidenceAction(t, "interpreted", "pass", phase, backend, result)
	if err == nil || !strings.Contains(err.Error(), "compile-only phase") {
		t.Fatalf("verifyRow error = %v, want compile-only rejection", err)
	}
}

func TestVerifierAcceptsBuildOnly(t *testing.T) {
	for _, mode := range []string{"interpreted", "compiled"} {
		t.Run(mode, func(t *testing.T) {
			phase, backend, result := buildEvidence(mode, "fixedbugs/issue59404.go", []string{"-gcflags=-l=4"}, "")
			status, err := verifyBuildEvidence(t, mode, "fixedbugs/issue59404.go", "pass", phase, backend, result)
			if err != nil || status != "BUILD-ONLY-PASS" {
				t.Fatalf("verifyRow = %q, %v", status, err)
			}
		})
	}
}

func TestVerifierRequiresUpstreamRunenvGoexperiment(t *testing.T) {
	phase, backend, result := buildEvidence("interpreted", "arenas/smoke.go", []string{}, "arenas")
	if status, err := verifyBuildEvidence(t, "interpreted", "arenas/smoke.go", "pass", phase, backend, result); err != nil || status != "BUILD-ONLY-PASS" {
		t.Fatalf("verifyRow = %q, %v", status, err)
	}
	phase.EnvDelta = []string{}
	if _, err := verifyBuildEvidence(t, "interpreted", "arenas/smoke.go", "pass", phase, backend, result); err == nil || !strings.Contains(err.Error(), "GOEXPERIMENT") {
		t.Fatalf("verifyRow error = %v, want missing runenv GOEXPERIMENT", err)
	}
}

func TestVerifierRejectsRewrappedBuildFlags(t *testing.T) {
	for _, wrapped := range [][]string{{"-gcflags=all=-l=4"}, {"-gcflags=-l=4 -N"}, {}} {
		phase, backend, result := buildEvidence("compiled", "fixedbugs/issue59638.go", wrapped, "")
		_, err := verifyBuildEvidence(t, "compiled", "fixedbugs/issue59638.go", "pass", phase, backend, result)
		if err == nil || !strings.Contains(err.Error(), "exact upstream go-command flags") {
			t.Fatalf("verifyRow(%v) error = %v, want verbatim flag mismatch", wrapped, err)
		}
	}
}

func TestVerifierRejectsBuildExecutePhaseOrArtifactRun(t *testing.T) {
	phase, backend, result := buildEvidence("compiled", "fixedbugs/issue59404.go", []string{"-gcflags=-l=4"}, "")
	execPhase := phase
	execPhase.PhaseKind = "execute"
	execPhase.Action = "build"
	execBackend := backend
	execBackend.Phase = "execute"
	execResult := result
	_, err := verifyBuildEvidence(t, "compiled", "fixedbugs/issue59404.go", "pass",
		phase, backend, result, execPhase, execBackend, execResult)
	if err == nil || !strings.Contains(err.Error(), "exactly one build phase") {
		t.Fatalf("verifyRow error = %v, want single build-only phase", err)
	}
}

func TestVerifierRejectsBuildArtifactOutsideUpstreamCwd(t *testing.T) {
	phase, backend, result := buildEvidence("compiled", "fixedbugs/issue59404.go", []string{"-gcflags=-l=4"}, "")
	backend.Artifacts = []string{backend.Artifacts[0], "/elsewhere/a.exe"}
	_, err := verifyBuildEvidence(t, "compiled", "fixedbugs/issue59404.go", "pass", phase, backend, result)
	if err == nil || !strings.Contains(err.Error(), "upstream working directory") {
		t.Fatalf("verifyRow error = %v, want cwd artifact mismatch", err)
	}
}

func TestVerifierRetainsBuildProductFailure(t *testing.T) {
	phase, backend, result := buildEvidence("compiled", "arenas/smoke.go", []string{}, "arenas")
	result.Exit = 1
	result.ArtifactProof = []fileProof{{Path: backend.Artifacts[0], Exists: true, Bytes: 10, SHA256: strings.Repeat("a", 64)}, {Path: backend.Artifacts[1]}}
	result.MapProof = []fileProof{{Path: backend.Maps[0], Exists: true, Bytes: 5, SHA256: strings.Repeat("c", 64)}}
	status, err := verifyBuildEvidence(t, "compiled", "arenas/smoke.go", "fail", phase, backend, result)
	if err != nil || status != "BUILD-PRODUCT-FAIL" {
		t.Fatalf("verifyRow = %q, %v, want retained product failure", status, err)
	}
	result.Exit = 0
	if _, err := verifyBuildEvidence(t, "compiled", "arenas/smoke.go", "fail", phase, backend, result); err == nil || !strings.Contains(err.Error(), "nonzero build phase exit") {
		t.Fatalf("verifyRow error = %v, want recorded nonzero exit", err)
	}
}

func buildEvidence(mode, test string, recipeFlags []string, goexperiment string) (eventRecord, eventRecord, eventRecord) {
	cwd := "/tmp/testdir"
	long := "/goroot/test/" + test
	native := []string{"go", "build", "", "-o", "a.exe", long}
	envDelta := []string{}
	if goexperiment != "" {
		envDelta = []string{"GOEXPERIMENT=" + goexperiment}
	}
	phase := eventRecord{Kind: "phase", Test: test, Action: "build", PhaseKind: "compile", CompileInputs: []string{long}, ProgramArgv: []string{}, RecipeFlags: recipeFlags, Argv: native, Cwd: cwd, EnvDelta: envDelta}
	backend := eventRecord{Kind: "backend", Test: test, BackendSchema: backendSchema, Mode: mode, Tool: toolIdentity{Path: "/bin/bashy", Version: "test"}, Action: phase.Action, Phase: phase.PhaseKind, CompileInputs: phase.CompileInputs, ProgramArgv: phase.ProgramArgv, RecipeFlags: phase.RecipeFlags, NativeArgv: native, Disposition: "check-only", Deviations: []string{"structured evidence"}}
	result := eventRecord{Kind: "phase_result", Test: test, Exit: 0}
	if mode == "compiled" {
		backend.Disposition = "transpile-build-only"
		backend.Artifacts = []string{"/tmp/module/main.go", cwd + "/a.exe"}
		backend.Maps = []string{"/tmp/module/main.go.map"}
		result.ArtifactProof = []fileProof{{Path: backend.Artifacts[0], Exists: true, Bytes: 10, SHA256: strings.Repeat("a", 64)}, {Path: backend.Artifacts[1], Exists: true, Bytes: 20, SHA256: strings.Repeat("b", 64)}}
		result.MapProof = []fileProof{{Path: backend.Maps[0], Exists: true, Bytes: 5, SHA256: strings.Repeat("c", 64)}}
	}
	return phase, backend, result
}

func verifyBuildEvidence(t *testing.T, mode, test, goAction string, records ...eventRecord) (string, error) {
	t.Helper()
	dir := t.TempDir()
	base := filepath.Join(dir, strings.NewReplacer("/", "_", ".", "_").Replace(test))
	writeJSONLines(t, base+".go-test.json", goRecord{Action: goAction, Test: "Test/" + test})
	items := make([]any, 0, len(records)+1)
	for _, record := range records {
		items = append(items, record)
	}
	items = append(items, eventRecord{Kind: "terminal", Test: test, Failed: goAction == "fail"})
	writeJSONLines(t, base+".events.jsonl", items...)
	return verifyRow(matrixRow{Test: test, Action: "build"}, dir, mode, "test", "/bin/bashy")
}

func compileEvidence(mode string) (eventRecord, eventRecord, eventRecord) {
	native := []string{"go", "tool", "compile", "-N", "bug020.go"}
	phase := eventRecord{Kind: "phase", Test: "fixedbugs/bug020.go", Action: "compile", PhaseKind: "compile", CompileInputs: []string{"bug020.go"}, ProgramArgv: []string{}, RecipeFlags: []string{"-N"}, Argv: native}
	backend := eventRecord{Kind: "backend", Test: phase.Test, BackendSchema: backendSchema, Mode: mode, Tool: toolIdentity{Path: "/bin/bashy", Version: "test"}, Action: phase.Action, Phase: phase.PhaseKind, CompileInputs: phase.CompileInputs, ProgramArgv: phase.ProgramArgv, RecipeFlags: phase.RecipeFlags, NativeArgv: native, Disposition: "check-only", Deviations: []string{"structured evidence"}}
	result := eventRecord{Kind: "phase_result", Test: phase.Test, Exit: 0}
	if mode == "compiled" {
		backend.Disposition = "transpile-compile-only"
		backend.Artifacts = []string{"/tmp/main.go", "/tmp/program"}
		backend.Maps = []string{"/tmp/main.go.map"}
		backend.CompilerArgv = []string{"go", "tool", "compile", "-importcfg=/tmp/importcfg", "-N", "/tmp/main.go"}
		result.ArtifactProof = []fileProof{{Path: "/tmp/main.go", Exists: true, Bytes: 10, SHA256: strings.Repeat("a", 64)}, {Path: "/tmp/program", Exists: true, Bytes: 20, SHA256: strings.Repeat("b", 64)}}
		result.MapProof = []fileProof{{Path: "/tmp/main.go.map", Exists: true, Bytes: 5, SHA256: strings.Repeat("c", 64)}}
	}
	return phase, backend, result
}

// Sprint: #162; Story: S162.4b; Story-ID: 41dd6897a116
func TestVerifierRequiresDirectCompilerEvidence(t *testing.T) {
	phase, backend, result := compileEvidence("compiled")
	status, err := verifyCompileEvidence(t, "compiled", phase, backend, result)
	if err != nil || status != "COMPILE-ONLY-PASS" {
		t.Fatalf("verifyRow = %q, %v", status, err)
	}
	backend.CompilerArgv = []string{"go", "build", "-gcflags=-complete", "/tmp/main.go"}
	if _, err := verifyCompileEvidence(t, "compiled", phase, backend, result); err == nil || !strings.Contains(err.Error(), "direct upstream-shaped compiler argv") {
		t.Fatalf("verifyRow error = %v, want direct compiler argv rejection", err)
	}
}

func verifyCompileEvidence(t *testing.T, mode string, records ...eventRecord) (string, error) {
	t.Helper()
	return verifyCompileEvidenceAction(t, mode, "pass", records...)
}

func verifyCompileEvidenceAction(t *testing.T, mode, goAction string, records ...eventRecord) (string, error) {
	t.Helper()
	dir := t.TempDir()
	base := filepath.Join(dir, "fixedbugs_bug020_go")
	writeJSONLines(t, base+".go-test.json", goRecord{Action: goAction, Test: "Test/fixedbugs/bug020.go"})
	items := make([]any, 0, len(records)+1)
	for _, record := range records {
		items = append(items, record)
	}
	items = append(items, eventRecord{Kind: "terminal", Test: "fixedbugs/bug020.go", Failed: goAction == "fail"})
	writeJSONLines(t, base+".events.jsonl", items...)
	return verifyRow(matrixRow{Test: "fixedbugs/bug020.go", Action: "compile"}, dir, mode, "test", "/bin/bashy")
}

func writeJSONLines(t *testing.T, name string, records ...any) {
	t.Helper()
	var data []byte
	for _, record := range records {
		line, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, line...)
		data = append(data, '\n')
	}
	if err := os.WriteFile(name, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// Sprint: #154; Story: S154.0; Story-ID: 4877afd3a207
// TestVerifierAcceptsDeclaredOptimizerDiagnostics: an interpreted errorcheck
// root whose recipe wants optimizer diagnostics (-m/-live/-race/-d=) is
// declared unsupported by the seam with the compiler-artifact reason, and the
// verifier accepts the declaration the way it accepts interpreted asmcheck.
// Without the declared reason, or on a non-failing terminal, it still rejects.
func TestVerifierAcceptsDeclaredOptimizerDiagnostics(t *testing.T) {
	optimizerEvidence := func(deviation string) []eventRecord {
		test := "escape/escape.go"
		native := []string{"go", "tool", "compile", "-m", "-l", "escape.go"}
		flags := []string{"-0", "-m", "-l"}
		phase := eventRecord{Kind: "phase", Test: test, Action: "errorcheck", PhaseKind: "compile", CompileInputs: []string{"/goroot/test/" + test}, ProgramArgv: []string{}, RecipeFlags: flags, Argv: native}
		backend := eventRecord{Kind: "backend", Test: test, BackendSchema: backendSchema, Mode: "interpreted", Tool: toolIdentity{Path: "/bin/bashy", Version: "test"}, Action: phase.Action, Phase: phase.PhaseKind, CompileInputs: phase.CompileInputs, ProgramArgv: phase.ProgramArgv, RecipeFlags: phase.RecipeFlags, NativeArgv: native, Disposition: "unsupported", Deviations: []string{deviation}}
		// A backendErr step never starts the command; done() records exit -1.
		result := eventRecord{Kind: "phase_result", Test: test, Exit: -1}
		return []eventRecord{phase, backend, result}
	}
	declared := "optimizer diagnostics are a compiler artifact; the check interface has no inlining, escape-analysis or SSA meaning"

	verify := func(goAction string, records []eventRecord) (string, error) {
		dir := t.TempDir()
		base := filepath.Join(dir, "escape_escape_go")
		writeJSONLines(t, base+".go-test.json", goRecord{Action: goAction, Test: "Test/escape/escape.go"})
		items := make([]any, 0, len(records)+1)
		for _, record := range records {
			items = append(items, record)
		}
		items = append(items, eventRecord{Kind: "terminal", Test: "escape/escape.go", Failed: goAction == "fail"})
		writeJSONLines(t, base+".events.jsonl", items...)
		return verifyRow(matrixRow{Test: "escape/escape.go", Action: "errorcheck"}, dir, "interpreted", "test", "/bin/bashy")
	}

	status, err := verify("fail", optimizerEvidence(declared))
	if err != nil || status != "UNSUPPORTED" {
		t.Fatalf("verifyRow = %q, %v, want accepted UNSUPPORTED declaration", status, err)
	}
	if _, err := verify("fail", optimizerEvidence("structured evidence")); err == nil || !strings.Contains(err.Error(), "disposition") {
		t.Fatalf("verifyRow error = %v, want undeclared unsupported rejection", err)
	}
	if _, err := verify("pass", optimizerEvidence(declared)); err == nil || !strings.Contains(err.Error(), "upstream action") {
		t.Fatalf("verifyRow error = %v, want rejection of a passing unsupported declaration", err)
	}
}

// Sprint: #150; Story: S150.6; Story-ID: 4228ed646074
func runEvidence(mode string, recipeFlags, programArgv []string) (eventRecord, eventRecord, eventRecord) {
	test := "fixedbugs/issue32680.go"
	native := append([]string{"go", "run", ""}, recipeFlags...)
	native = append(native, test)
	native = append(native, programArgv...)
	phase := eventRecord{Kind: "phase", Test: test, Action: "run", PhaseKind: "execute", CompileInputs: []string{test}, ProgramArgv: programArgv, RecipeFlags: recipeFlags, Argv: native, Cwd: "/tmp/testdir"}
	backend := eventRecord{Kind: "backend", Test: test, BackendSchema: backendSchema, Mode: mode, Tool: toolIdentity{Path: "/bin/bashy", Version: "test"}, Action: phase.Action, Phase: phase.PhaseKind, CompileInputs: phase.CompileInputs, ProgramArgv: phase.ProgramArgv, RecipeFlags: phase.RecipeFlags, NativeArgv: native, Disposition: "check-then-run", Deviations: []string{"structured evidence"}}
	if mode == "compiled" {
		backend.Disposition = "transpile-build-run"
	}
	if len(recipeFlags) != 0 {
		backend.Deviations = append(backend.Deviations, "upstream go-command recipe flags are passed verbatim")
	}
	result := eventRecord{Kind: "phase_result", Test: test, Exit: 0}
	return phase, backend, result
}

func verifyRunEvidence(t *testing.T, mode, goAction string, records ...eventRecord) (string, error) {
	t.Helper()
	dir := t.TempDir()
	base := filepath.Join(dir, "fixedbugs_issue32680_go")
	writeJSONLines(t, base+".go-test.json", goRecord{Action: goAction, Test: "Test/fixedbugs/issue32680.go"})
	items := make([]any, 0, len(records)+1)
	for _, record := range records {
		items = append(items, record)
	}
	items = append(items, eventRecord{Kind: "terminal", Test: "fixedbugs/issue32680.go", Failed: goAction == "fail"})
	writeJSONLines(t, base+".events.jsonl", items...)
	return verifyRow(matrixRow{Test: "fixedbugs/issue32680.go", Action: "run"}, dir, mode, "test", "/bin/bashy")
}

func TestVerifierAcceptsDirectRun(t *testing.T) {
	for _, mode := range []string{"interpreted", "compiled"} {
		t.Run(mode, func(t *testing.T) {
			phase, backend, result := runEvidence(mode, []string{"-gcflags=-d=ssa/check/on"}, []string{})
			status, err := verifyRunEvidence(t, mode, "pass", phase, backend, result)
			if err != nil || status != "RUN-PASS" {
				t.Fatalf("verifyRow = %q, %v", status, err)
			}
			// Upstream checkExpectedOutput can fail after a clean exit; that is
			// a retained product failure, not a seam defect.
			status, err = verifyRunEvidence(t, mode, "fail", phase, backend, result)
			if err != nil || status != "RUN-PRODUCT-FAIL" {
				t.Fatalf("verifyRow = %q, %v, want retained product failure", status, err)
			}
		})
	}
}

func TestVerifierRejectsRunBoundaryAndUndeclaredFlags(t *testing.T) {
	phase, backend, result := runEvidence("compiled", []string{"-race"}, []string{"extra.go"})
	phase.ProgramArgv, backend.ProgramArgv = []string{"extra.go"}, []string{"extra.go"}
	if _, err := verifyRunEvidence(t, "compiled", "pass", phase, backend, result); err == nil || !strings.Contains(err.Error(), "boundary") {
		t.Fatalf("verifyRow error = %v, want source/argument boundary rejection", err)
	}
	phase, backend, result = runEvidence("compiled", []string{"-race"}, []string{})
	backend.Deviations = []string{"structured evidence"}
	if _, err := verifyRunEvidence(t, "compiled", "pass", phase, backend, result); err == nil || !strings.Contains(err.Error(), "declared") {
		t.Fatalf("verifyRow error = %v, want undeclared recipe-flag rejection", err)
	}
	phase, backend, result = runEvidence("interpreted", nil, []string{})
	backend.Disposition = "transpile-build-run"
	if _, err := verifyRunEvidence(t, "interpreted", "pass", phase, backend, result); err == nil || !strings.Contains(err.Error(), "direct") {
		t.Fatalf("verifyRow error = %v, want wrong-mode rejection", err)
	}
}

// Sprint: #150; Story: S150.5; Story-ID: e87e1cbcbb20
func buildRunEvidence(mode string) []eventRecord {
	test := "fixedbugs/issue46234.go"
	cwd, long := "/tmp/testdir", "/goroot/test/"+test
	build := eventRecord{Kind: "phase", Test: test, Action: "buildrun", PhaseKind: "compile", CompileInputs: []string{long}, ProgramArgv: []string{}, RecipeFlags: []string{}, Argv: []string{"go", "build", "", "-o", "a.exe", long}, Cwd: cwd}
	buildBackend := eventRecord{Kind: "backend", Test: test, BackendSchema: backendSchema, Mode: mode, Tool: toolIdentity{Path: "/bin/bashy", Version: "test"}, Action: "buildrun", Phase: "compile", CompileInputs: build.CompileInputs, ProgramArgv: build.ProgramArgv, RecipeFlags: build.RecipeFlags, NativeArgv: build.Argv, Disposition: "check-only", Deviations: []string{"structured evidence"}}
	run := eventRecord{Kind: "phase", Test: test, Action: "buildrun", PhaseKind: "execute", CompileInputs: []string{}, ProgramArgv: []string{}, RecipeFlags: []string{}, Argv: []string{"./a.exe"}, Cwd: cwd}
	runBackend := eventRecord{Kind: "backend", Test: test, BackendSchema: backendSchema, Mode: mode, Tool: toolIdentity{Path: "/bin/bashy", Version: "test"}, Action: "buildrun", Phase: "execute", CompileInputs: run.CompileInputs, ProgramArgv: run.ProgramArgv, RecipeFlags: run.RecipeFlags, NativeArgv: run.Argv, Disposition: "run-remembered-program", Deviations: []string{"structured evidence"}, Program: &programRec{Files: build.CompileInputs}}
	if mode == "compiled" {
		buildBackend.Disposition = "transpile-build-only"
		buildBackend.Artifacts = []string{"/tmp/module/main.go", cwd + "/a.exe"}
		buildBackend.Maps = []string{"/tmp/module/main.go.map"}
		runBackend.Disposition = "run-artifact"
		runBackend.Program.Artifact = cwd + "/a.exe"
		runBackend.Artifacts = []string{cwd + "/a.exe"}
	}
	return []eventRecord{build, buildBackend, {Kind: "phase_result", Test: test, Exit: 0}, run, runBackend, {Kind: "phase_result", Test: test, Exit: 0}}
}

func verifyBuildRunEvidence(t *testing.T, mode, goAction string, records []eventRecord) (string, error) {
	t.Helper()
	dir := t.TempDir()
	base := filepath.Join(dir, "fixedbugs_issue46234_go")
	writeJSONLines(t, base+".go-test.json", goRecord{Action: goAction, Test: "Test/fixedbugs/issue46234.go"})
	items := make([]any, 0, len(records)+1)
	for _, record := range records {
		items = append(items, record)
	}
	items = append(items, eventRecord{Kind: "terminal", Test: "fixedbugs/issue46234.go", Failed: goAction == "fail"})
	writeJSONLines(t, base+".events.jsonl", items...)
	return verifyRow(matrixRow{Test: "fixedbugs/issue46234.go", Action: "buildrun"}, dir, mode, "test", "/bin/bashy")
}

func TestVerifierBuildRunProgramContinuity(t *testing.T) {
	for _, mode := range []string{"interpreted", "compiled"} {
		t.Run(mode, func(t *testing.T) {
			records := buildRunEvidence(mode)
			status, err := verifyBuildRunEvidence(t, mode, "pass", records)
			if err != nil || status != "BUILDRUN-PASS" {
				t.Fatalf("verifyRow = %q, %v", status, err)
			}
			// The execute phase must act on exactly what the build phase compiled.
			records = buildRunEvidence(mode)
			records[4].Program.Files = []string{"/goroot/test/other.go"}
			if _, err := verifyBuildRunEvidence(t, mode, "pass", records); err == nil || !strings.Contains(err.Error(), "not what the build phase compiled") {
				t.Fatalf("verifyRow error = %v, want program continuity rejection", err)
			}
			// A failed build stops upstream: one phase, recorded nonzero exit.
			records = buildRunEvidence(mode)[:3]
			records[2].Exit = 1
			status, err = verifyBuildRunEvidence(t, mode, "fail", records)
			if err != nil || status != "BUILDRUN-PRODUCT-FAIL" {
				t.Fatalf("verifyRow = %q, %v, want retained build failure", status, err)
			}
		})
	}
}

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// buildDirEvidence is the compiled phase sequence of a buildrundir root with
// one .s companion (asmhdr / retjmp shape): upstream's -gensymabis generate
// phase, the one Go compile with -asmhdr/-symabis, the object assembly, pack,
// link and execute — each assembly phase run natively by the pinned
// assembler and recorded, the Go files transpiled and compiled directly.
func buildDirEvidence(mode string, assembly bool) []eventRecord {
	test := "asmdir.go"
	cwd, dir := "/tmp/testdir", "/goroot/test/asmdir.dir"
	gos, asms := []string{dir + "/main.go"}, []string{dir + "/f.s"}
	goTool := "/goroot/bin/go"
	tool := toolIdentity{Path: "/bin/bashy", Version: "test"}
	mk := func(kind, phaseKind string, inputs, argv []string) eventRecord {
		return eventRecord{Kind: kind, Test: test, BackendSchema: backendSchema, Mode: mode, Tool: tool, Action: "buildrundir", Phase: phaseKind, PhaseKind: phaseKind, CompileInputs: inputs, ProgramArgv: []string{}, RecipeFlags: []string{}, NativeArgv: argv, Argv: argv, Cwd: cwd, Deviations: []string{"structured evidence"}}
	}
	result := func() eventRecord { return eventRecord{Kind: "phase_result", Test: test, Exit: 0} }
	var records []eventRecord
	compileArgv := []string{goTool, "tool", "compile", "-p=main", "-e", "-D", ".", "-importcfg=/tmp/importcfg", "-o", "go.o"}
	if assembly {
		symabis := append([]string{goTool, "tool", "asm", "-p=main", "-gensymabis", "-o", "symabis"}, asms...)
		phase, backend := mk("phase", "generate", asms, symabis), mk("backend", "generate", asms, symabis)
		backend.Disposition, backend.Artifacts = "assemble-native", []string{cwd + "/symabis"}
		records = append(records, phase, backend, result())
		compileArgv = append(compileArgv, "-asmhdr", "go_asm.h", "-symabis", "symabis")
	}
	compileArgv = append(compileArgv, gos...)
	phase, backend := mk("phase", "compile", gos, compileArgv), mk("backend", "compile", gos, compileArgv)
	generated := "/tmp/module/main.go"
	compilerArgv := append(append([]string(nil), compileArgv[:len(compileArgv)-len(gos)]...), generated)
	proof := result()
	backend.ImportBase, backend.ImportPath = ".", "main"
	identityArgs := []string{"--go-import-base", ".", "--go-import-path", "main"}
	if mode == "compiled" {
		backend.Disposition = "transpile-compile-directory-build"
		backend.Artifacts = []string{generated, cwd + "/go.o"}
		backend.Maps = []string{generated + ".map"}
		backend.CompilerArgv = compilerArgv
		proof.ArtifactProof = []fileProof{{Path: generated, Exists: true, Bytes: 10, SHA256: strings.Repeat("a", 64)}, {Path: cwd + "/go.o", Exists: true, Bytes: 20, SHA256: strings.Repeat("b", 64)}}
		proof.MapProof = []fileProof{{Path: generated + ".map", Exists: true, Bytes: 5, SHA256: strings.Repeat("c", 64)}}
	} else {
		backend.Disposition = "check-only"
	}
	records = append(records, phase, backend, proof)
	objects := []string{"go.o"}
	var assembled []string
	if assembly {
		asm := append([]string{goTool, "tool", "asm", "-p=main", "-e", "-I", ".", "-o", "asm.o"}, asms...)
		phase, backend := mk("phase", "compile", asms, asm), mk("backend", "compile", asms, asm)
		backend.Disposition, backend.Artifacts = "assemble-native", []string{cwd + "/asm.o"}
		records = append(records, phase, backend, result())
		objects = append(objects, "asm.o")
		assembled = []string{cwd + "/asm.o"}
	}
	packArgv := append([]string{goTool, "tool", "pack", "c", "all.a"}, objects...)
	phase, backend = mk("phase", "link", objects, packArgv), mk("backend", "link", objects, packArgv)
	packed := &programRec{Files: gos, MapArgs: identityArgs, ImportBase: ".", ImportPath: "main", Object: "all.a", Assembled: assembled}
	backend.Disposition, backend.Program = "pack-adopt-check", packed
	if mode == "compiled" {
		packed.Artifact = cwd + "/all.a"
		backend.Disposition, backend.Artifacts = "pack-adopt-artifact", []string{cwd + "/all.a"}
	}
	records = append(records, phase, backend, result())
	linkArgv := []string{goTool, "tool", "link", "-o", "a.exe", "-importcfg=/tmp/importcfg", "all.a"}
	phase, backend = mk("phase", "link", []string{"all.a"}, linkArgv), mk("backend", "link", []string{"all.a"}, linkArgv)
	linked := &programRec{Files: gos, MapArgs: identityArgs, ImportBase: ".", ImportPath: "main", Object: "a.exe", Assembled: assembled}
	backend.Disposition, backend.Program = "link-adopt-check", linked
	if mode == "compiled" {
		linked.Artifact = cwd + "/a.exe"
		backend.Disposition, backend.Artifacts = "link-adopt-artifact", []string{cwd + "/a.exe"}
	}
	records = append(records, phase, backend, result())
	runArgv := []string{cwd + "/a.exe"}
	phase, backend = mk("phase", "execute", []string{}, runArgv), mk("backend", "execute", []string{}, runArgv)
	backend.Disposition, backend.Program = "run-remembered-program", linked
	if mode == "compiled" {
		backend.Disposition, backend.Artifacts = "run-artifact", []string{cwd + "/a.exe"}
	}
	records = append(records, phase, backend, result())
	return records
}

func verifyBuildDirEvidence(t *testing.T, mode, action, goAction string, records []eventRecord) (string, error) {
	t.Helper()
	dir := t.TempDir()
	base := filepath.Join(dir, "asmdir_go")
	writeJSONLines(t, base+".go-test.json", goRecord{Action: goAction, Test: "Test/asmdir.go"})
	items := make([]any, 0, len(records)+1)
	for _, record := range records {
		items = append(items, record)
	}
	items = append(items, eventRecord{Kind: "terminal", Test: "asmdir.go", Failed: goAction == "fail"})
	writeJSONLines(t, base+".events.jsonl", items...)
	return verifyRow(matrixRow{Test: "asmdir.go", Action: action}, dir, mode, "test", "/bin/bashy")
}

func TestVerifierBuildDirAssemblyCompanions(t *testing.T) {
	// The compiled sequence with a companion passes; without one (Go-only
	// directory) it passes too, in both modes.
	for _, tt := range []struct {
		mode     string
		assembly bool
	}{{"compiled", true}, {"compiled", false}, {"interpreted", false}} {
		status, err := verifyBuildDirEvidence(t, tt.mode, "buildrundir", "pass", buildDirEvidence(tt.mode, tt.assembly))
		if err != nil || status != "BUILDDIR-PASS" {
			t.Fatalf("%s assembly=%v: verifyRow = %q, %v", tt.mode, tt.assembly, status, err)
		}
	}
	// builddir stops at link: an execute phase is not a builddir phase.
	records := buildDirEvidence("compiled", true)
	if status, err := verifyBuildDirEvidence(t, "compiled", "builddir", "pass", records[:len(records)-3]); err != nil || status != "BUILDDIR-PASS" {
		t.Fatalf("builddir: verifyRow = %q, %v", status, err)
	}
	if _, err := verifyBuildDirEvidence(t, "compiled", "builddir", "pass", records); err == nil || !strings.Contains(err.Error(), "execute after link for builddir") {
		t.Fatalf("builddir with an execute phase: error = %v", err)
	}
	// The interpreted companion refusal is the declared generate-phase
	// limitation: one unsupported phase over the .s inputs, a failed terminal.
	records = buildDirEvidence("compiled", true)[:3]
	for i := range records {
		records[i].Mode = "interpreted"
	}
	records[1].Disposition, records[1].Artifacts = "unsupported", nil
	if status, err := verifyBuildDirEvidence(t, "interpreted", "buildrundir", "fail", records); err != nil || status != "UNSUPPORTED" {
		t.Fatalf("interpreted companion: verifyRow = %q, %v", status, err)
	}
	if _, err := verifyBuildDirEvidence(t, "interpreted", "buildrundir", "pass", records); err == nil {
		t.Fatal("an interpreted refusal with a passing terminal must be rejected")
	}
	// A link failure (a declaration no assembly implements) is a product
	// failure with its recorded nonzero exit, never a pass.
	records = buildDirEvidence("compiled", true)
	records = records[:len(records)-3]
	records[len(records)-1].Exit = 2
	if status, err := verifyBuildDirEvidence(t, "compiled", "buildrundir", "fail", records); err != nil || status != "BUILDDIR-PRODUCT-FAIL" {
		t.Fatalf("link failure: verifyRow = %q, %v", status, err)
	}
	records[len(records)-1].Exit = 0
	if _, err := verifyBuildDirEvidence(t, "compiled", "buildrundir", "fail", records); err == nil || !strings.Contains(err.Error(), "nonzero phase exit") {
		t.Fatalf("failure without a nonzero exit: error = %v", err)
	}
}

func TestVerifierBuildDirRejectsPermissiveShapes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(records []eventRecord)
		want   string
	}{
		{"compiled unsupported phase", func(r []eventRecord) { r[1].Disposition = "unsupported" }, "recorded meaning in every phase"},
		{"assembly not run through go tool asm", func(r []eventRecord) { r[1].NativeArgv[2] = "compile"; r[0].Argv[2] = "compile" }, "not go tool asm"},
		{"symabis phase without -gensymabis", func(r []eventRecord) {
			r[1].NativeArgv = append([]string{r[1].NativeArgv[0], "tool", "asm", "-o", "symabis"}, r[1].CompileInputs...)
			r[0].Argv = r[1].NativeArgv
		}, "-gensymabis wanted true"},
		{"assembly phase claiming a generated artifact", func(r []eventRecord) { r[1].Maps = []string{"/tmp/x.map"} }, "nothing generated"},
		{"Go compile dropped -symabis", func(r []eventRecord) {
			var argv []string
			for _, a := range r[4].CompilerArgv {
				if a != "-symabis" && a != "symabis" {
					argv = append(argv, a)
				}
			}
			r[4].CompilerArgv = argv
		}, "-symabis"},
		{"Go compile through cmd/go", func(r []eventRecord) {
			r[4].CompilerArgv = []string{"/goroot/bin/go", "build", "-o", "go.o", "/tmp/module/main.go"}
		}, "direct upstream-shaped compiler argv"},
		{"pack adopting a foreign object", func(r []eventRecord) { r[10].Program.Assembled = []string{"/elsewhere/other.o"} }, "assembled objects"},
		{"pack with a non-Go source compiled as Go", func(r []eventRecord) {
			r[4].CompileInputs = append(r[4].CompileInputs, "/goroot/test/asmdir.dir/f.s")
			r[3].CompileInputs = r[4].CompileInputs
		}, "non-Go input"},
		{"link adopting another program", func(r []eventRecord) { r[13].Program.Files = []string{"/goroot/test/other.go"} }, "did not adopt the packed archive"},
		{"execute running something else", func(r []eventRecord) {
			r[16].Program = &programRec{Files: r[16].Program.Files, Object: "b.exe", Artifact: "/tmp/testdir/b.exe"}
		}, "did not run the linked program"},
		{"execute before link", func(r []eventRecord) {
			r[12].PhaseKind, r[12].CompileInputs, r[13].Phase, r[13].CompileInputs = "execute", []string{}, "execute", []string{}
			r[13].Disposition = "run-artifact"
		}, "execute after pack"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := buildDirEvidence("compiled", true)
			tt.mutate(records)
			_, err := verifyBuildDirEvidence(t, "compiled", "buildrundir", "pass", records)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// TestCompilerIdentityReadsUpstreamArgv pins the verifier's reading of
// upstream's compile identity: the -D / -p of a direct `go tool compile`
// argv, last occurrence winning, inputs excluded by name; none for any other
// native command.
func TestCompilerIdentityReadsUpstreamArgv(t *testing.T) {
	for _, tt := range []struct {
		argv       []string
		inputs     []string
		base, path string
	}{
		{[]string{"go", "tool", "compile", "-e", "-p=p", "-importcfg=/i", "/t/x.go"}, []string{"/t/x.go"}, "", "p"},
		{[]string{"go", "tool", "compile", "-p=p", "-d=panic", "-C", "-e", "-importcfg=/i", "-+", "-p=runtime", "/t/x.go"}, []string{"/t/x.go"}, "", "runtime"},
		{[]string{"go", "tool", "compile", "-p=p", "-e", "-importcfg=/i", "-p", "example.com/dotted", "/t/x.go"}, []string{"/t/x.go"}, "", "example.com/dotted"},
		{[]string{"go", "tool", "compile", "-e", "-D", "test", "-importcfg=/i", "-p=main", "/t/d/main.go"}, []string{"/t/d/main.go"}, "test", "main"},
		{[]string{"go", "tool", "compile", "-e", "-D", "test", "-importcfg=/i", "-o", "test/a.a", "-p", "test/a", "/t/d/a.go", "/t/d/b.go"}, []string{"/t/d/a.go", "/t/d/b.go"}, "test", "test/a"},
		{[]string{"go", "tool", "compile", "-p=main", "-e", "-D", ".", "-importcfg=/i", "-o", "go.o", "-asmhdr", "go_asm.h", "-symabis", "symabis", "/t/d/main.go"}, []string{"/t/d/main.go"}, ".", "main"},
		{[]string{"go", "tool", "compile", "-p=", "-e", "/t/x.go"}, []string{"/t/x.go"}, "", ""},
		{[]string{"go", "tool", "compile", "-e", "/t/x.go", "-p"}, []string{"/t/x.go"}, "", ""},
		{[]string{"go", "run", "-gcflags=-p=x", "/t/x.go"}, []string{"/t/x.go"}, "", ""},
		{[]string{"go", "build", "-o", "a.exe", "/t/x.go"}, []string{"/t/x.go"}, "", ""},
		{[]string{"go", "tool", "asm", "-p=main", "-gensymabis", "-o", "symabis", "/t/d/f.s"}, []string{"/t/d/f.s"}, "", ""},
		{[]string{"/t/a.exe"}, nil, "", ""},
	} {
		base, path := compilerIdentity(tt.argv, tt.inputs)
		if base != tt.base || path != tt.path {
			t.Errorf("compilerIdentity(%v) = %q/%q, want %q/%q", tt.argv, base, path, tt.base, tt.path)
		}
	}
}

// identityCompileEvidence is a single-file compile phase whose native argv
// carries upstream's -p (compileFile's `-p=p`) and whose backend recorded
// the handoff.
func identityCompileEvidence(mode string) (eventRecord, eventRecord, eventRecord) {
	phase, backend, result := compileEvidence(mode)
	native := []string{"go", "tool", "compile", "-e", "-p=p", "-importcfg=/tmp/importcfg", "-N", "bug020.go"}
	phase.Argv, backend.NativeArgv = native, native
	backend.ImportPath = "p"
	if mode == "compiled" {
		backend.CompilerArgv = []string{"go", "tool", "compile", "-e", "-p=p", "-importcfg=/tmp/importcfg", "-N", "/tmp/main.go"}
	}
	return phase, backend, result
}

// TestVerifierRequiresUpstreamIdentityHandoff (S165.0, D8): a phase that
// reaches the front end records exactly the identity of upstream's own
// compile argv — never one it invented, never a dropped one — and a phase
// whose native argv carries none records none.
func TestVerifierRequiresUpstreamIdentityHandoff(t *testing.T) {
	for _, mode := range []string{"interpreted", "compiled"} {
		t.Run(mode, func(t *testing.T) {
			phase, backend, result := identityCompileEvidence(mode)
			if status, err := verifyCompileEvidence(t, mode, phase, backend, result); err != nil || status != "COMPILE-ONLY-PASS" {
				t.Fatalf("verifyRow = %q, %v", status, err)
			}
			// dropped: upstream said -p=p, the seam handed nothing
			backend.ImportPath = ""
			if _, err := verifyCompileEvidence(t, mode, phase, backend, result); err == nil || !strings.Contains(err.Error(), "not upstream's own compile identity") {
				t.Fatalf("dropped identity: error = %v", err)
			}
			// invented: the seam handed an identity upstream never chose
			backend.ImportPath = "main"
			if _, err := verifyCompileEvidence(t, mode, phase, backend, result); err == nil || !strings.Contains(err.Error(), "not upstream's own compile identity") {
				t.Fatalf("invented identity: error = %v", err)
			}
			backend.ImportPath, backend.ImportBase = "p", "test"
			if _, err := verifyCompileEvidence(t, mode, phase, backend, result); err == nil || !strings.Contains(err.Error(), "not upstream's own compile identity") {
				t.Fatalf("invented base: error = %v", err)
			}
			// a recipe's own -p after upstream's is the identity gc compiles under
			backend.ImportBase = ""
			native := append(append([]string(nil), phase.Argv[:len(phase.Argv)-1]...), "-p=example.com/dotted", "bug020.go")
			phase.Argv, backend.NativeArgv = native, native
			if _, err := verifyCompileEvidence(t, mode, phase, backend, result); err == nil || !strings.Contains(err.Error(), "not upstream's own compile identity") {
				t.Fatalf("recipe -p ignored: error = %v", err)
			}
			backend.ImportPath = "example.com/dotted"
			if status, err := verifyCompileEvidence(t, mode, phase, backend, result); err != nil || status != "COMPILE-ONLY-PASS" {
				t.Fatalf("recipe -p honoured: verifyRow = %q, %v", status, err)
			}
			// a native argv without -p (the S149 evidence shape) records none
			phase, backend, result = compileEvidence(mode)
			if status, err := verifyCompileEvidence(t, mode, phase, backend, result); err != nil || status != "COMPILE-ONLY-PASS" {
				t.Fatalf("no identity: verifyRow = %q, %v", status, err)
			}
			backend.ImportPath = "p"
			if _, err := verifyCompileEvidence(t, mode, phase, backend, result); err == nil || !strings.Contains(err.Error(), "not upstream's own compile identity") {
				t.Fatalf("identity invented for a -p-less argv: error = %v", err)
			}
		})
	}
	// go run carries no compiler identity: the seam must hand none.
	phase, backend, result := runEvidence("interpreted", nil, []string{})
	if status, err := verifyRunEvidence(t, "interpreted", "pass", phase, backend, result); err != nil || status != "RUN-PASS" {
		t.Fatalf("run: verifyRow = %q, %v", status, err)
	}
	backend.ImportPath = "main"
	if _, err := verifyRunEvidence(t, "interpreted", "pass", phase, backend, result); err == nil || !strings.Contains(err.Error(), "not upstream's own compile identity") {
		t.Fatalf("run with an invented identity: error = %v", err)
	}
	// An unsupported phase hands nothing to the front end; its record is
	// not an identity claim, whatever upstream's argv carried.
	phase, backend, _ = identityCompileEvidence("interpreted")
	backend.Disposition = "unsupported"
	for _, recorded := range []string{"", "p"} {
		backend.ImportPath = recorded
		if err := checkIdentity(phase, backend); err != nil {
			t.Fatalf("unsupported phase recording %q: %v", recorded, err)
		}
	}
	backend.Disposition = "check-only"
	if err := checkIdentity(phase, backend); err != nil {
		t.Fatalf("check-only with upstream's identity: %v", err)
	}
}

// runDirEvidence is the phase sequence of a single-package rundir root
// (intrinsic's shape): one directory compile under upstream's `-D test
// -p=main`, the link adopting it, the execute running the remembered program
// under the same identity (S165.0, D8 request (b)).
func runDirEvidence(mode string) []eventRecord {
	test := "sysdir.go"
	cwd, dir := "/tmp/testdir", "/goroot/test/sysdir.dir"
	gos := []string{dir + "/main.go"}
	goTool := "/goroot/bin/go"
	tool := toolIdentity{Path: "/bin/bashy", Version: "test"}
	mk := func(kind, phaseKind string, inputs, argv []string) eventRecord {
		return eventRecord{Kind: kind, Test: test, BackendSchema: backendSchema, Mode: mode, Tool: tool, Action: "rundir", Phase: phaseKind, PhaseKind: phaseKind, CompileInputs: inputs, ProgramArgv: []string{}, RecipeFlags: []string{}, NativeArgv: argv, Argv: argv, Cwd: cwd, Deviations: []string{"structured evidence"}}
	}
	result := func() eventRecord { return eventRecord{Kind: "phase_result", Test: test, Exit: 0} }
	compileArgv := append([]string{goTool, "tool", "compile", "-e", "-D", "test", "-importcfg=/tmp/importcfg", "-p=main"}, gos...)
	phase, backend := mk("phase", "compile", gos, compileArgv), mk("backend", "compile", gos, compileArgv)
	phase.Package = &packageID{Base: "test", Path: "main"}
	backend.PackageMap = &packageMap{Base: "test", Path: "main"}
	backend.ImportBase, backend.ImportPath = "test", "main"
	generated := "/tmp/module/main.go"
	proof := result()
	if mode == "compiled" {
		backend.Disposition = "transpile-compile-package-map"
		backend.Artifacts = []string{generated, cwd + "/main.o"}
		backend.Maps = []string{generated + ".map"}
		backend.CompilerArgv = append(append([]string(nil), compileArgv[:len(compileArgv)-1]...), generated)
		proof.ArtifactProof = []fileProof{{Path: generated, Exists: true, Bytes: 10, SHA256: strings.Repeat("a", 64)}, {Path: cwd + "/main.o", Exists: true, Bytes: 20, SHA256: strings.Repeat("b", 64)}}
		proof.MapProof = []fileProof{{Path: generated + ".map", Exists: true, Bytes: 5, SHA256: strings.Repeat("c", 64)}}
	} else {
		backend.Disposition = "check-package-map"
	}
	records := []eventRecord{phase, backend, proof}
	program := &programRec{Files: gos, MapArgs: []string{"--go-import-base", "test", "--go-import-path", "main"}, ImportBase: "test", ImportPath: "main"}
	linkArgv := []string{goTool, "tool", "link", "-o", "a.exe", "-importcfg=/tmp/importcfg", "main.o"}
	phase, backend = mk("phase", "link", []string{"main.o"}, linkArgv), mk("backend", "link", []string{"main.o"}, linkArgv)
	backend.Disposition, backend.Program = "link-adopt-check", program
	if mode == "compiled" {
		program.Artifact, program.Object = cwd+"/a.exe", "a.exe"
		backend.Disposition, backend.Artifacts = "link-adopt-artifact", []string{cwd + "/a.exe"}
	}
	records = append(records, phase, backend, result())
	runArgv := []string{cwd + "/a.exe"}
	phase, backend = mk("phase", "execute", []string{}, runArgv), mk("backend", "execute", []string{}, runArgv)
	backend.Disposition, backend.Program = "run-remembered-program", program
	if mode == "compiled" {
		backend.Disposition, backend.Artifacts = "run-artifact", []string{cwd + "/a.exe"}
	}
	records = append(records, phase, backend, result())
	return records
}

func verifyRunDirEvidence(t *testing.T, mode, goAction string, records []eventRecord) (string, error) {
	t.Helper()
	dir := t.TempDir()
	base := filepath.Join(dir, "sysdir_go")
	writeJSONLines(t, base+".go-test.json", goRecord{Action: goAction, Test: "Test/sysdir.go"})
	items := make([]any, 0, len(records)+1)
	for _, record := range records {
		items = append(items, record)
	}
	items = append(items, eventRecord{Kind: "terminal", Test: "sysdir.go", Failed: goAction == "fail"})
	writeJSONLines(t, base+".events.jsonl", items...)
	return verifyRow(matrixRow{Test: "sysdir.go", Action: "rundir"}, dir, mode, "test", "/bin/bashy")
}

// TestVerifierRunDirProgramKeepsIdentity (S165.0, D8 request (b)): the
// program a single-package directory root's link and execute phases act on
// carries the compile phase's identity and hands it through its map args;
// a program without it — the pre-S165 single-package shape — is rejected.
func TestVerifierRunDirProgramKeepsIdentity(t *testing.T) {
	for _, mode := range []string{"interpreted", "compiled"} {
		t.Run(mode, func(t *testing.T) {
			records := runDirEvidence(mode)
			if status, err := verifyRunDirEvidence(t, mode, "pass", records); err != nil || status != "RUNDIR-PASS" {
				t.Fatalf("verifyRow = %q, %v", status, err)
			}
			for _, tt := range []struct {
				name   string
				mutate func(p *programRec)
				want   string
			}{
				{"no identity on the program", func(p *programRec) { p.ImportBase, p.ImportPath, p.MapArgs = "", "", nil }, "differs from its compile phase"},
				{"identity named but not handed", func(p *programRec) { p.MapArgs = nil }, "hand --go-import-base"},
				{"another identity handed", func(p *programRec) { p.MapArgs = []string{"--go-import-base", "test", "--go-import-path", "p"} }, "hand --go-import-path"},
				{"the base dropped", func(p *programRec) { p.MapArgs = []string{"--go-import-path", "main"} }, "hand --go-import-base"},
			} {
				records := runDirEvidence(mode)
				for i := 4; i < len(records); i += 3 {
					copied := *records[i].Program
					records[i].Program = &copied
				}
				tt.mutate(records[4].Program)
				if mode == "compiled" {
					tt.mutate(records[7].Program)
				} else {
					records[7].Program = records[4].Program
				}
				if _, err := verifyRunDirEvidence(t, mode, "pass", records); err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("%s: error = %v, want %q", tt.name, err, tt.want)
				}
			}
			// The compile phase itself must record upstream's -D / -p.
			records = runDirEvidence(mode)
			records[1].ImportPath = "p"
			if _, err := verifyRunDirEvidence(t, mode, "pass", records); err == nil || !strings.Contains(err.Error(), "not upstream's own compile identity") {
				t.Fatalf("compile phase identity: error = %v", err)
			}
			// And upstream's own package identity must agree with its argv.
			records = runDirEvidence(mode)
			records[0].Package = &packageID{Base: "test", Path: "test/main"}
			if _, err := verifyRunDirEvidence(t, mode, "pass", records); err == nil || !strings.Contains(err.Error(), "disagrees with its compile argv") {
				t.Fatalf("package identity vs argv: error = %v", err)
			}
		})
	}
}

// TestVerifierRunDirLinkAdoptsCompiledObject pins the compiled rundir link
// check against the seam as it records it since the direct-gc route (S162):
// the link input is upstream's object name for the compile phase's artifact
// and the link phase records the a.exe it linked as the program artifact.
func TestVerifierRunDirLinkAdoptsCompiledObject(t *testing.T) {
	records := runDirEvidence("compiled")
	records[4].CompileInputs, records[3].CompileInputs = []string{"other.o"}, []string{"other.o"}
	if _, err := verifyRunDirEvidence(t, "compiled", "pass", records); err == nil || !strings.Contains(err.Error(), "not the last compile phase's object") {
		t.Fatalf("foreign link input: error = %v", err)
	}
	records = runDirEvidence("compiled")
	copied := *records[4].Program
	copied.Artifact = "/tmp/testdir/main.o"
	records[4].Program = &copied
	if _, err := verifyRunDirEvidence(t, "compiled", "pass", records); err == nil || !strings.Contains(err.Error(), "linked program as its artifact") {
		t.Fatalf("link recording the object: error = %v", err)
	}
}

func TestS243DirectoryMapImportClosureEvidence(t *testing.T) {
	dir := t.TempDir()
	write := func(name, source string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	good := mappedPackage{Path: "test/good", Files: []string{write("good.go", "package good; const N = 1")}}
	bad := mappedPackage{Path: "test/bad", Files: []string{write("bad.go", "package bad; var N int = `bad`")}}
	via := mappedPackage{Path: "test/via", Files: []string{write("via.go", "package via; import _ `./bad`")}}
	available := []mappedPackage{good, bad, via}
	main := write("main.go", "package main; import _ `./good`")
	m := &packageMap{Base: "test", Path: "main", Selection: "import-closure", Available: available, Packages: []mappedPackage{good}}
	if err := verifyDirectoryMap(m, []string{main}, available); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"package main; import _ `./bad`", "package main; import _ `test/bad`", "package main; import _ `./via`"} {
		write("main.go", source)
		if err := verifyDirectoryMap(m, []string{main}, available); err == nil {
			t.Fatal("accepted omitted rejected dependency")
		}
	}
	m.Packages = []mappedPackage{bad, via}
	if err := verifyDirectoryMap(m, []string{main}, available); err != nil {
		t.Fatal(err)
	}
	m.Available = []mappedPackage{good, via}
	if err := verifyDirectoryMap(m, []string{main}, available); err == nil {
		t.Fatal("accepted altered upstream enumeration")
	}
	m.Available = available
	m.Packages = []mappedPackage{{Path: "test/bad", Files: good.Files}, via}
	if err := verifyDirectoryMap(m, []string{main}, available); err == nil {
		t.Fatal("accepted substituted imported source")
	}
}
