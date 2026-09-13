// Copyright 2026 The bashpp-tests Authors. All rights reserved.
// Sprint: #157; Story: S157.2; Story-ID: 31520c72b5e0
// Sprint: #149; Stories: S149.1 (3f416ade73ef), S149.2 (1e87cb008ec3), S149.3 (60d35d1ec914)
// Sprint: #150; Stories: S150.6 (4228ed646074), S150.5 (e87e1cbcbb20), S150.1 (a136a527c0b3), S150.2 (8f758b9dcd5a)
// Sprint: #154; Story: S154.0; Story-ID: 4877afd3a207
// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
//
// Direct Go-source backend for the authenticated Go 1.27 testdir seam. The
// only program description accepted here is the compileInputs/programArgv
// boundary selected by upstream and handed to planExec, plus the package
// identity (-D base, -p path) upstream chose for a directory package or
// carries in its own direct compile argv for a single file (S165.0, D8).
package testdir_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const backendSchema = "bashpp-tests/upstream-testdir-backend/v1"

func goFileArgs(inputs []string) []string {
	args := make([]string, 0, len(inputs)*2)
	for _, input := range inputs {
		args = append(args, "--go-file", input)
	}
	return args
}

func directSourceCommand(selected *exec.Cmd, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Dir = selected.Dir
	cmd.Env = append([]string(nil), selected.Env...)
	cmd.Stdin = selected.Stdin
	cmd.Stdout = selected.Stdout
	cmd.Stderr = selected.Stderr
	cmd.ExtraFiles = selected.ExtraFiles
	cmd.SysProcAttr = selected.SysProcAttr
	return cmd
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func shellCommand(selected *exec.Cmd, commands ...[]string) *exec.Cmd {
	lines := []string{"set -e"}
	for i, command := range commands {
		words := make([]string, len(command))
		for j, word := range command {
			words[j] = shellQuote(word)
		}
		if i == len(commands)-1 {
			words = append([]string{"exec"}, words...)
		}
		lines = append(lines, strings.Join(words, " "))
	}
	return directSourceCommand(selected, "/bin/sh", "-c", strings.Join(lines, "\n"))
}

func (t test) backendEvent(mode, action, phase, disposition string, compileInputs, programArgv, recipeFlags, nativeArgv, artifacts, maps, deviations []string) {
	t.backendEventMap(mode, action, phase, disposition, compileInputs, programArgv, recipeFlags, nativeArgv, artifacts, maps, deviations, nil)
}

// backendEventMap is backendEvent plus the structured package map handed to
// Bash++ for a directory package phase, so a verifier can check it without
// parsing prose.
func (t test) backendEventMap(mode, action, phase, disposition string, compileInputs, programArgv, recipeFlags, nativeArgv, artifacts, maps, deviations []string, packageMap map[string]any) {
	t.backendEventProgram(mode, action, phase, disposition, compileInputs, programArgv, recipeFlags, nativeArgv, artifacts, maps, deviations, packageMap, nil)
}

func (t test) backendEventCompiler(mode, action, phase, disposition string, compileInputs, programArgv, recipeFlags, nativeArgv, artifacts, maps, deviations []string, packageMap map[string]any, compilerArgv []string) {
	t.backendEventProgram(mode, action, phase, disposition, compileInputs, programArgv, recipeFlags, nativeArgv, artifacts, maps, deviations, packageMap, map[string]any{"compiler_argv": compilerArgv})
}

// backendEventProgram is backendEventMap plus the program record a later
// phase of the same upstream test acted on (S150.5): the files and package
// map earlier phases were handed and, in compiled mode, the artifact they
// built. A verifier can then check that link/execute acted on exactly what
// upstream compiled, without the seam ever looking for anything on disk.
func (t test) backendEventProgram(mode, action, phase, disposition string, compileInputs, programArgv, recipeFlags, nativeArgv, artifacts, maps, deviations []string, packageMap map[string]any, program map[string]any) {
	var compilerArgv []string
	if program != nil {
		compilerArgv, _ = program["compiler_argv"].([]string)
	}
	var identity backendIdentity
	if value, ok := backendIdentities.Load(t.eventName()); ok {
		identity = value.(backendIdentity)
	}
	events.emit(t.eventName(), "backend", map[string]any{
		"program":        program,
		"package_map":    packageMap,
		"backend_schema": backendSchema,
		"mode":           mode,
		"action":         action,
		"import_base":    identity.Base,
		"import_path":    identity.Path,
		"compile_inputs": nonNil(compileInputs),
		"program_argv":   nonNil(programArgv),
		"recipe_flags":   nonNil(recipeFlags),
		"native_argv":    nonNil(nativeArgv),
		"compiler_argv":  nonNil(compilerArgv),
		"artifacts":      nonNil(artifacts),
		"maps":           nonNil(maps),
		"tool": map[string]any{
			"path":    os.Getenv("BASHPP_TESTDIR_TOOL"),
			"version": os.Getenv("BASHPP_TESTDIR_VERSION"),
		},
		"phase":       phase,
		"disposition": disposition,
		"deviations":  nonNil(deviations),
	})
}

// packageGroup is one directory package upstream has already planned for a
// test, in upstream order: the -p path it chose and the exact files.
type packageGroup struct {
	path  string
	files []string
}

// backendPackages remembers, per upstream test, the directory packages
// planned so far, so a later package's plan can hand every earlier one to
// Bash++ as an explicit --go-package entry — the in-memory equivalent of the
// importcfg upstream accumulates for the same test. Tests run in parallel;
// the key is the upstream test identity.
var backendPackages sync.Map

// backendProgram is what the phases of one upstream test have handed the
// seam so far as the program: the files (and package map) of the last
// compile phase and, in compiled mode, the artifact that phase built. A
// later `link` or input-less `execute` phase of the same test acts on it.
type backendProgram struct {
	files        []string
	mapArgs      []string
	artifact     string
	compilerArgv []string
	// object is upstream's own name for the artifact of the last phase
	// (its -o operand: builddir's go.o, then all.a), when it had one; the
	// directory compile without -o keeps the .go -> .o rule instead.
	object string
	// assembled is the objects the pinned assembler produced natively from
	// the directory's .s companions (S165.0, D3(a)); pack may consume them
	// and nothing else may.
	assembled []string
	// identity is the package identity the compile phase handed Bash++
	// (S165.0, D8); the interpreted execute phase runs the remembered
	// sources under the same one, so a single-package directory program
	// keeps upstream's `-p main` at execute exactly like a multi-package one.
	identity backendIdentity
}

// backendPrograms is keyed by the upstream test identity, like backendPackages.
var backendPrograms sync.Map

func (p backendProgram) record() map[string]any {
	return map[string]any{"files": nonNil(p.files), "map_args": nonNil(p.mapArgs), "artifact": p.artifact, "compiler_argv": nonNil(p.compilerArgv), "object": p.object, "assembled": nonNil(p.assembled),
		"import_base": p.identity.Base, "import_path": p.identity.Path}
}

// backendIdentity is the package identity upstream's own compiler argv
// carries — its -D relative-import base and its -p import path — handed to
// Bash++ unchanged as --go-import-base / --go-import-path (S165.0, D8). A
// directory package's identity is the one upstream's compileInDir chose; a
// single-file compile phase's is whatever -p upstream's own argv carries
// (`-p=p` for errorcheck/compile, `-p=main` for a directory build). The seam
// never invents one: a phase whose native argv carries no -p hands Bash++
// no identity and the checker keeps its directory rule.
type backendIdentity struct {
	Base string
	Path string
}

// compilerIdentity reads the -D / -p of a direct `go tool compile` argv the
// way the compiler's own flag parsing does — the last occurrence wins, so a
// recipe's own `-p=…` after upstream's `-p=p` is the identity gc compiles
// under. The compile inputs are excluded by name; nothing else is
// interpreted. Any other native command (go run, go build, go tool asm)
// carries no compiler identity.
func compilerIdentity(nativeArgv, compileInputs []string) backendIdentity {
	var id backendIdentity
	if len(nativeArgv) < 3 || nativeArgv[1] != "tool" || nativeArgv[2] != "compile" {
		return id
	}
	inputs := make(map[string]bool, len(compileInputs))
	for _, input := range compileInputs {
		inputs[input] = true
	}
	args := nativeArgv[3:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if inputs[arg] {
			continue
		}
		name := strings.TrimLeft(arg, "-")
		if len(name) == len(arg) || len(arg)-len(name) > 2 {
			continue
		}
		var value string
		if flag, inline, ok := strings.Cut(name, "="); ok {
			name, value = flag, inline
		} else if name == "p" || name == "D" {
			i++
			if i >= len(args) {
				break
			}
			value = args[i]
		}
		switch name {
		case "p":
			id.Path = value
		case "D":
			id.Base = value
		}
	}
	return id
}

// args is the identity as the Bash++ front end takes it; nothing when the
// native argv carried none.
func (id backendIdentity) args() []string {
	var args []string
	if id.Base != "" {
		args = append(args, "--go-import-base", id.Base)
	}
	if id.Path != "" {
		args = append(args, "--go-import-path", id.Path)
	}
	return args
}

// backendIdentities is the identity of the phase the seam is planning, per
// upstream test (keyed like backendPackages), so every backend event of that
// phase records the import_base / import_path handed to Bash++. Set for the
// duration of backendPlan only.
var backendIdentities sync.Map

// owns reports whether a link-phase input set is exactly what this test's
// earlier phases handed the seam: the object of its compiler artifact (by
// upstream's -o name or the .go -> .o rule) and, for `go tool pack`, any
// object the seam assembled natively for the same test.
func (p backendProgram) owns(compileInputs []string, pack bool) bool {
	if len(p.files) == 0 || len(compileInputs) == 0 || (!pack && len(compileInputs) != 1) {
		return false
	}
	assembled := make(map[string]bool, len(p.assembled))
	for _, object := range p.assembled {
		assembled[filepath.Base(object)] = true
	}
	compiler := 0
	for _, input := range compileInputs {
		base := filepath.Base(input)
		switch {
		case p.object != "" && base == p.object:
			compiler++
		case p.object == "" && strings.TrimSuffix(base, ".o")+".go" == filepath.Base(p.files[0]):
			compiler++
		case pack && assembled[base]:
		default:
			return false
		}
	}
	return compiler == 1
}

// directCompileCommand is the upstream compiler invocation with the original
// Go inputs replaced by the one generated Go file.  In particular it retains
// upstream's -p and -importcfg resolution rather than asking cmd/go to invent
// a package build (and its implicit -complete).
func directCompileCommand(nativeArgv, compileInputs []string, generated string) ([]string, error) {
	if len(nativeArgv) < 4 || nativeArgv[1] != "tool" || nativeArgv[2] != "compile" {
		return nil, fmt.Errorf("upstream compile argv is not a go tool compile invocation: %v", nativeArgv)
	}
	inputs := make(map[string]bool, len(compileInputs))
	for _, input := range compileInputs {
		inputs[input] = true
	}
	args := make([]string, 0, len(nativeArgv)-len(compileInputs)+1)
	args = append(args, nativeArgv[0])
	for _, arg := range nativeArgv[1:] {
		if !inputs[arg] {
			args = append(args, arg)
		}
	}
	return append(args, generated), nil
}

func compilerOutput(argv []string, generated, dir string) string {
	for i, arg := range argv {
		if arg == "-o" && i+1 < len(argv) {
			if filepath.IsAbs(argv[i+1]) {
				return argv[i+1]
			}
			return filepath.Join(dir, argv[i+1])
		}
		if strings.HasPrefix(arg, "-o=") {
			name := strings.TrimPrefix(arg, "-o=")
			if filepath.IsAbs(name) {
				return name
			}
			return filepath.Join(dir, name)
		}
	}
	// Without -o, cmd/compile writes the basename-derived object into its
	// working directory, rather than next to an absolute source argument.
	// The linker phase uses that same working directory for its source.o
	// rewrite, so retain the generated basename but resolve it in dir.
	return filepath.Join(dir, strings.TrimSuffix(filepath.Base(generated), ".go")+".o")
}

func TestCompilerOutputUsesPhaseDirectoryForImplicitObject(t *testing.T) {
	phaseDir := t.TempDir()
	generated := filepath.Join(t.TempDir(), "main.go")
	if got, want := compilerOutput([]string{"go", "tool", "compile", generated}, generated, phaseDir), filepath.Join(phaseDir, "main.o"); got != want {
		t.Fatalf("implicit compiler output = %q, want %q", got, want)
	}
	if got, want := compilerOutput([]string{"go", "tool", "compile", "-o", "test/a.a", generated}, generated, phaseDir), filepath.Join(phaseDir, "test", "a.a"); got != want {
		t.Fatalf("explicit compiler output = %q, want %q", got, want)
	}
}

// backendModule writes the temporary Go module that hosts transpiled source.
func backendModule(t test, shellrt string) (moduleDir string, err error) {
	moduleDir = t.TempDir()
	module := fmt.Sprintf("module bashpp_s1572\n\ngo 1.27\n\nrequire mvdan.cc/sh/v3 v3.13.1\nreplace mvdan.cc/sh/v3 => %s\n", shellrt)
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(module), 0o600); err != nil {
		return "", fmt.Errorf("write Bash++ backend module: %w", err)
	}
	return moduleDir, nil
}

// backendLink gives upstream's link phase a meaning (S150.1). Upstream hands
// it the object name of the last package it compiled (its own .go -> .o
// rewrite) plus any -ldflags; the seam checks that object is the program its
// last directory compile phase handed it and adopts that program: nothing to
// link in interpreted mode (the sources run later), the artifact the pinned
// build already linked in compiled mode. Anything else is unsupported.
func (t test) backendLink(step *planStep, mode, action string, compileInputs, programArgv, recipeFlags, nativeArgv, deviations []string) {
	value, ok := backendPrograms.Load(t.eventName())
	program, _ := value.(backendProgram)
	// builddir/buildrundir pack the compiler object (and the natively
	// assembled objects) into all.a before linking it; the archive is the
	// program the following link phase adopts.
	pack := len(nativeArgv) >= 5 && nativeArgv[1] == "tool" && nativeArgv[2] == "pack" && nativeArgv[3] == "c"
	if !ok || !program.owns(compileInputs, pack) {
		step.backendErr = fmt.Errorf("Bash++ backend unsupported link phase: input %v is not the program this test's compile phases handed the seam", compileInputs)
		t.backendEvent(mode, action, "link", "unsupported", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
			append(deviations, "the link input is not the object of the last directory package upstream compiled through Bash++"))
		return
	}
	stage, adopt := "link", "link-adopt"
	if pack {
		stage, adopt = "pack", "pack-adopt"
		deviations = append(deviations,
			"the pack inputs are upstream's object names for the generated package's compiler artifact and the natively assembled companions; the seam adopts them and never packs anything else")
	} else {
		deviations = append(deviations,
			"the link input is upstream's object name for the last package it compiled; the seam adopts the program that compile phase handed it and never links objects",
			"upstream -ldflags are retained as evidence only")
	}
	switch mode {
	case "interpreted":
		if pack {
			// The following link phase names the archive; the sources stay
			// the program.
			program.object = filepath.Base(nativeArgv[4])
			backendPrograms.Store(t.eventName(), program)
		}
		*step.cmd = *directSourceCommand(step.cmd, "/bin/sh", "-c", ":")
		t.backendEventProgram(mode, action, "link", adopt+"-check", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
			append(deviations, "an interpreter has nothing to "+stage+": the checked sources are the program and run at the execute phase"), nil, program.record())
	case "compiled":
		if program.artifact == "" {
			step.backendErr = fmt.Errorf("Bash++ backend unsupported link phase: no artifact was built for this program")
			t.backendEvent(mode, action, "link", "unsupported", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil, deviations)
			return
		}
		linkArgv := append([]string(nil), nativeArgv...)
		assembled := make(map[string]string, len(program.assembled))
		for _, object := range program.assembled {
			assembled[filepath.Base(object)] = object
		}
		for i, arg := range linkArgv {
			if i < 3 {
				continue
			}
			switch {
			case pack && i == 4:
				// the archive upstream names
			case assembled[arg] != "":
				linkArgv[i] = assembled[arg]
			case (program.object != "" && arg == program.object) || (program.object == "" && strings.HasSuffix(arg, ".o")):
				linkArgv[i] = program.artifact
			}
		}
		var linked string
		if pack {
			linked = linkArgv[4]
			if !filepath.IsAbs(linked) {
				linked = filepath.Join(step.cmd.Dir, linked)
			}
		} else {
			linked = compilerOutput(linkArgv, "", step.cmd.Dir)
		}
		program.artifact = linked
		program.object = filepath.Base(linked)
		backendPrograms.Store(t.eventName(), program)
		*step.cmd = *directSourceCommand(step.cmd, linkArgv[0], linkArgv[1:]...)
		deviation := "the pinned Go linker receives the object produced by the generated source; upstream link flags and importcfg are retained exactly"
		if pack {
			deviation = "the pinned Go packer receives the object produced by the generated source and the natively assembled objects; upstream's archive name is retained exactly"
		}
		t.backendEventProgram(mode, action, "link", adopt+"-artifact", compileInputs, programArgv, recipeFlags, nativeArgv, []string{linked}, nil,
			append(deviations, deviation), nil, program.record())
	default:
		step.backendErr = fmt.Errorf("unsupported Bash++ backend mode %q", mode)
		t.backendEvent(mode, action, "link", "configuration-error", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil, deviations)
	}
}

// assemblyInputs reports whether every compile input is an assembly source.
func assemblyInputs(inputs []string) bool {
	if len(inputs) == 0 {
		return false
	}
	for _, input := range inputs {
		if !strings.HasSuffix(input, ".s") {
			return false
		}
	}
	return true
}

// backendAssemble gives the .s companions of a builddir/buildrundir root
// their D3(a) meaning (S165.0). Upstream hands them to `go tool asm`
// directly, twice: `-gensymabis` (its generate phase, before the Go compile)
// and the object assembly (a compile phase, after it). Assembly is a
// compiler artifact: in compiled mode the pinned assembler runs upstream's
// exact argv, natively and unchanged, and the seam records it as a phase of
// the test — no Go source of the root is ever run natively (the Go files go
// through transpile + direct compile like every compile-only recipe, and the
// object assembly is remembered for pack). Interpreted mode has no assembly
// meaning and stays unsupported with its recorded reason.
func (t test) backendAssemble(step *planStep, mode, action, phase string, compileInputs, programArgv, recipeFlags, nativeArgv, deviations []string) {
	if mode == "interpreted" {
		step.backendErr = fmt.Errorf("Bash++ backend unsupported %s phase: compile input %q is not a Go source file", phase, compileInputs[0])
		t.backendEvent(mode, action, phase, "unsupported", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
			append(deviations, "assembly is a compiler artifact; an interpreter has no assembly meaning and the companion's Go declarations have no body to run"))
		return
	}
	if len(nativeArgv) < 4 || nativeArgv[1] != "tool" || nativeArgv[2] != "asm" {
		step.backendErr = fmt.Errorf("Bash++ backend unsupported %s phase: assembly inputs %v are not handed to go tool asm: %v", phase, compileInputs, nativeArgv)
		t.backendEvent(mode, action, phase, "unsupported", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
			append(deviations, "assembly sources have a meaning only as upstream's own go tool asm invocation"))
		return
	}
	if mode != "compiled" {
		step.backendErr = fmt.Errorf("unsupported Bash++ backend mode %q", mode)
		t.backendEvent(mode, action, phase, "configuration-error", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil, deviations)
		return
	}
	output := compilerOutput(nativeArgv, "", step.cmd.Dir)
	symabis := false
	for _, arg := range nativeArgv {
		if arg == "-gensymabis" {
			symabis = true
		}
	}
	// step.cmd is left exactly as upstream built it: the pinned assembler
	// on the upstream .s files in the upstream working directory.
	step.artifacts = []string{output}
	if !symabis {
		if value, ok := backendPrograms.Load(t.eventName()); ok {
			program := value.(backendProgram)
			program.assembled = append(append([]string(nil), program.assembled...), output)
			backendPrograms.Store(t.eventName(), program)
		}
	}
	t.backendEvent(mode, action, phase, "assemble-native", compileInputs, programArgv, recipeFlags, nativeArgv, step.artifacts, nil,
		append(deviations,
			"D3(a): assembly is a compiler artifact; the pinned toolchain's go tool asm runs upstream's exact argv on the upstream .s files, natively and unchanged, as a recorded phase of this test",
			"no Go source of the root runs natively: the directory's Go files reach the compiler only as the transpiled file, and the assembled object is consumed only by this test's pack phase"))
}

// goListPackage is the subset of `go list -json` the seam reads.
type goListPackage struct {
	ImportPath string
	Name       string
	Dir        string
	GoFiles    []string
	SFiles     []string
	CgoFiles   []string
	Standard   bool
	Module     *struct {
		Path string
		Main bool
	}
}

// resolveModuleProgram asks the pinned go command what "." is in dir: the main
// package's Go files (absolute) and every in-module dependency as an ordered
// --go-package entry (go list -deps emits dependencies before dependents).
// A package with assembly or cgo files has no direct Go-source meaning.
func resolveModuleProgram(goTool, dir string, env []string) (files, mapArgs []string, record map[string]any, err error) {
	if goTool == "" {
		return nil, nil, nil, fmt.Errorf("resolving a module program requires BASHPP_TESTDIR_GO")
	}
	cmd := exec.Command(goTool, "list", "-json", "-deps", ".")
	cmd.Dir, cmd.Env = dir, env
	out, err := cmd.Output()
	record = map[string]any{"go_list": append([]string{goTool}, cmd.Args[1:]...), "dir": dir}
	if err != nil {
		return nil, nil, record, fmt.Errorf("go list -json -deps . failed in %s: %v", dir, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	var packages []map[string]any
	var main *goListPackage
	for {
		var pkg goListPackage
		if err := dec.Decode(&pkg); err != nil {
			break
		}
		if pkg.Standard || pkg.Module == nil || !pkg.Module.Main {
			continue
		}
		if len(pkg.SFiles) != 0 || len(pkg.CgoFiles) != 0 {
			return nil, nil, record, fmt.Errorf("module package %s has non-Go inputs %v", pkg.ImportPath, append(pkg.SFiles, pkg.CgoFiles...))
		}
		abs := make([]string, 0, len(pkg.GoFiles))
		for _, f := range pkg.GoFiles {
			abs = append(abs, filepath.Join(pkg.Dir, f))
		}
		if pkg.Name == "main" && pkg.Dir == dir {
			p := pkg
			main = &p
			files = abs
			continue
		}
		mapArgs = append(mapArgs, "--go-package", pkg.ImportPath+"="+strings.Join(abs, ","))
		packages = append(packages, map[string]any{"path": pkg.ImportPath, "files": abs})
	}
	if main == nil || len(files) == 0 {
		return nil, nil, record, fmt.Errorf("go list found no main package in %s", dir)
	}
	if len(packages) != 0 {
		mapArgs = append([]string{"--go-import-path", main.ImportPath}, mapArgs...)
	}
	record["base"] = ""
	record["path"] = main.ImportPath
	record["packages"] = packages
	return files, mapArgs, record, nil
}

// artifactUse states what happens to the cwd a.exe a build-only phase writes:
// nothing for `build`; for `buildrun` only this test's later execute phase
// runs it (S150.5).
func artifactUse(action string) string {
	if action == "buildrun" {
		return "the a.exe artifact is written to the upstream working directory and is executed only by this test's later execute phase"
	}
	return "the a.exe artifact is written to the upstream working directory and is never executed"
}

// optimizerDiagnosticFlags reports whether an errorcheck recipe asks the
// compiler for optimizer diagnostics: -m (any -m… form), -live, or a
// -d= debug flag. The check interface has no inlining, escape analysis or
// SSA, so interpreted mode declares such a recipe unsupported, the same shape
// as interpreted asmcheck.
func optimizerDiagnosticFlags(flags []string) bool {
	for _, flag := range flags {
		if strings.HasPrefix(flag, "-m") || strings.HasPrefix(flag, "-live") || debugDiagnosticFlag(flag) {
			return true
		}
	}
	return false
}

// debugDiagnosticFlag mirrors partition-emit.go: a -d= flag whose output the
// expectations depend on. -d=ssa/check/on (appended by upstream to every
// errorcheck compile) and -d=panic (gc panics on its first ordinary error)
// carry no expectation and never make a recipe an optimizer-diagnostic one.
func debugDiagnosticFlag(flag string) bool {
	if !strings.HasPrefix(flag, "-d=") {
		return false
	}
	for _, option := range strings.Split(strings.TrimPrefix(flag, "-d="), ",") {
		switch option {
		case "", "ssa/check/on", "panic":
			continue
		}
		return true
	}
	return false
}

// asmBuildArgs mirrors the upstream asmcheck flag merge: -gcflags values are
// folded into the single -S=2 argument; every other flag is a go build flag.
func asmBuildArgs(recipeFlags []string) (gcflags string, buildFlags []string) {
	gcflags = "-S=2"
	for i := 0; i < len(recipeFlags); i++ {
		flag := recipeFlags[i]
		switch {
		case strings.HasPrefix(flag, "-gcflags="):
			gcflags += " " + strings.TrimPrefix(flag, "-gcflags=")
		case strings.HasPrefix(flag, "--gcflags="):
			gcflags += " " + strings.TrimPrefix(flag, "--gcflags=")
		case flag == "-gcflags", flag == "--gcflags":
			i++
			if i < len(recipeFlags) {
				gcflags += " " + recipeFlags[i]
			}
		default:
			buildFlags = append(buildFlags, flag)
		}
	}
	return gcflags, buildFlags
}

func (t test) backendPlan(step *planStep, action, phase string, pkg *packageIdentity, compileInputs, programArgv, recipeFlags []string) {
	mode := os.Getenv("BASHPP_TESTDIR_BACKEND")
	tool := os.Getenv("BASHPP_TESTDIR_TOOL")
	nativeArgv := append([]string(nil), step.cmd.Args...)
	if mode == "" {
		return
	}
	// The identity of this phase (S165.0, D8): upstream's own -D / -p for a
	// directory package, else whatever -p its direct compile argv carries;
	// every backend event of the phase records it as import_base /
	// import_path, and it is handed to Bash++ wherever the phase reaches
	// the front end.
	identity := compilerIdentity(nativeArgv, compileInputs)
	if pkg != nil {
		identity = backendIdentity{Base: pkg.Base, Path: pkg.Path}
	}
	backendIdentities.Store(t.eventName(), identity)
	defer backendIdentities.Delete(t.eventName())
	// Upstream bounds a command only when the recipe says `-t N`; every other
	// phase is unbounded, which is right for the native compiler and wrong
	// for a product under test that can hang. The backend lane applies the
	// Sprint 148 60-second per-stage deadline (BASHPP_TESTDIR_DEADLINE
	// overrides) through upstream's own timer, so a timed-out root is
	// upstream's errTimeout — a product row, never a seam error.
	step.deadline = 60
	if v := os.Getenv("BASHPP_TESTDIR_DEADLINE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			step.deadline = n
		}
	}
	if len(compileInputs) == 0 {
		// An execute phase with no inputs runs the program an earlier phase
		// of this same test built (buildrun's `./a.exe`, rundir's linked
		// a.exe): the seam remembers what upstream handed it, never looks.
		if value, ok := backendPrograms.Load(t.eventName()); ok && phase == "execute" && tool != "" {
			program := value.(backendProgram)
			deviations := []string{
				"upstream native command argv is preserved as evidence and never used to classify the action",
				"the execute phase carries no compile inputs; the program is the one this test's earlier compile phase handed the seam",
			}
			switch mode {
			case "interpreted":
				runArgs := append([]string{"--bashpp", "--source=go"}, program.mapArgs...)
				runArgs = append(runArgs, goFileArgs(program.files)...)
				if len(programArgv) != 0 {
					runArgs = append(runArgs, "--")
					runArgs = append(runArgs, programArgv...)
				}
				*step.cmd = *directSourceCommand(step.cmd, tool, runArgs...)
				t.backendEventProgram(mode, action, phase, "run-remembered-program", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
					append(deviations, "the remembered sources run directly through the Bash++ interpreter with only the upstream program argv; no artifact exists in interpreted mode"), nil, program.record())
				return
			case "compiled":
				if program.artifact == "" {
					break
				}
				*step.cmd = *directSourceCommand(step.cmd, program.artifact, programArgv...)
				t.backendEventProgram(mode, action, phase, "run-artifact", compileInputs, programArgv, recipeFlags, nativeArgv, []string{program.artifact}, nil,
					append(deviations, "the artifact the earlier compile phase built from the transpiled sources runs with only the upstream program argv"), nil, program.record())
				return
			}
		}
		step.backendErr = fmt.Errorf("Bash++ backend unsupported %s phase without Go source inputs", phase)
		t.backendEvent(mode, action, phase, "unsupported", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
			[]string{"upstream selected no language-source inputs and no earlier phase of this test handed the seam a program; native tested-source execution is disabled in backend mode"})
		return
	}

	deviations := []string{
		"upstream native command argv is preserved as evidence and never used to classify the action",
		"the upstream run fast path is replaced by its existing source-execution plan so planExec receives language sources",
	}
	if phase == "link" {
		t.backendLink(step, mode, action, compileInputs, programArgv, recipeFlags, nativeArgv, deviations)
		return
	}
	// runindir: upstream built a module (overlay copy + go.mod) and runs `go run
	// .` in it. "." means whatever the go command's own on-disk policy says in
	// that exact directory; the seam asks the pinned go once and hands the
	// answer to Bash++ as files plus an explicit package map. It never walks
	// the directory itself.
	// sourceFiles is what Bash++ is handed as --go-file inputs; it equals
	// upstream's compileInputs except for runindir's ".", which the go
	// command resolves. The backend event always records upstream's inputs.
	sourceFiles := compileInputs
	var moduleMapArgs []string
	var moduleMap map[string]any
	if len(compileInputs) == 1 && compileInputs[0] == "." && phase == "execute" {
		goTool := os.Getenv("BASHPP_TESTDIR_GO")
		resolved, mapArgs, record, err := resolveModuleProgram(goTool, step.cmd.Dir, step.cmd.Env)
		if err != nil {
			step.backendErr = fmt.Errorf("Bash++ backend unsupported execute phase: %v", err)
			t.backendEventMap(mode, action, phase, "unsupported", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
				append(deviations, "the module program upstream prepared has no direct Go-source meaning: "+err.Error()), record)
			return
		}
		deviations = append(deviations, "the \".\" input is resolved once by the pinned go command's own module policy (`go list -json -deps .` in the upstream-prepared module directory, with the upstream environment) into the main package's Go files and the in-module dependency packages, in dependency order; the seam walks no directory")
		sourceFiles, moduleMapArgs, moduleMap = resolved, mapArgs, record
		record["files"] = nonNil(resolved)
	}
	// builddir/buildrundir (S165.0): the directory's .s companions are
	// assembled by upstream's own go tool asm invocations around the one
	// Go compile; see backendAssemble.
	directoryBuild := (action == "builddir" || action == "buildrundir") && pkg == nil && (phase == "compile" || phase == "generate")
	if directoryBuild && assemblyInputs(sourceFiles) {
		t.backendAssemble(step, mode, action, phase, compileInputs, programArgv, recipeFlags, nativeArgv, deviations)
		return
	}
	for _, input := range sourceFiles {
		if !strings.HasSuffix(input, ".go") {
			step.backendErr = fmt.Errorf("Bash++ backend unsupported %s phase: compile input %q is not a Go source file", phase, input)
			t.backendEvent(mode, action, phase, "unsupported", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
				append(deviations, "non-Go compile input has no direct Go-source meaning"))
			return
		}
	}
	// An upstream "generate" phase is a `go run` of the selected source whose
	// output becomes the next phase's input (runoutput, errorcheckoutput);
	// it has exactly the execute phase's direct meaning.
	run := (phase == "execute" || phase == "generate") && !directoryBuild
	compileOnly := action == "compile" && phase == "compile"
	// The Go files of a builddir/buildrundir directory: one direct compile
	// with upstream's exact flags (-p=main -e -D . -importcfg, -o go.o and,
	// with companions, -asmhdr go_asm.h -symabis symabis).
	directoryBuild = directoryBuild && phase == "compile"
	buildOnly := (action == "build" || action == "buildrun") && phase == "compile"
	diagnostics := phase == "compile" && pkg == nil &&
		(action == "errorcheck" || action == "errorcheckoutput" || action == "errorcheckwithauto")
	directory := phase == "compile" && pkg != nil
	assembly := action == "asmcheck" && phase == "compile"
	if !run && !compileOnly && !buildOnly && !diagnostics && !directory && !assembly && !directoryBuild {
		step.backendErr = fmt.Errorf("Bash++ backend unsupported source phase %q", phase)
		t.backendEvent(mode, action, phase, "unsupported", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
			append(deviations, "only upstream execute/generate phases and the compile phases of compile, build, builddir, errorcheck, directory and asmcheck actions have a direct Bash++ meaning"))
		return
	}
	if tool == "" || os.Getenv("BASHPP_TESTDIR_VERSION") == "" {
		step.backendErr = fmt.Errorf("Bash++ backend requires an identified BASHPP_TESTDIR_TOOL")
		t.backendEvent(mode, action, phase, "configuration-error", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil, deviations)
		return
	}

	fileArgs := goFileArgs(sourceFiles)
	if moduleMap != nil {
		fileArgs = append(append([]string(nil), moduleMapArgs...), fileArgs...)
	}

	// Directory packages: every earlier package of this same upstream test is
	// an explicit --go-package entry; the current package carries upstream's
	// own -D base and -p path. Nothing is discovered on disk.
	var mapArgs []string
	var packageMap map[string]any
	if !directory {
		// A single-file phase carries upstream's own compile identity when
		// its native argv does (`-p=p` for errorcheck/compile, `-p=main` for
		// a directory build): the same handoff as the directory phase, so the
		// checker decides internal visibility on the identity gc compiles
		// under. go run / go build argv carry none and none is invented.
		mapArgs = identity.args()
		if len(mapArgs) != 0 {
			deviations = append(deviations,
				fmt.Sprintf("upstream's own compile identity (-D %q -p %q) is handed to Bash++ as an explicit --go-import-base/--go-import-path; the seam invents no identity", identity.Base, identity.Path))
		}
	}
	if directory {
		key := t.eventName()
		var earlier []packageGroup
		if value, ok := backendPackages.Load(key); ok {
			earlier = value.([]packageGroup)
		}
		mapArgs = []string{"--go-import-base", pkg.Base, "--go-import-path", pkg.Path}
		for _, group := range earlier {
			mapArgs = append(mapArgs, "--go-package", group.path+"="+strings.Join(group.files, ","))
		}
		backendPackages.Store(key, append(append([]packageGroup(nil), earlier...), packageGroup{path: pkg.Path, files: append([]string(nil), compileInputs...)}))
		packages := make([]map[string]any, 0, len(earlier))
		for _, group := range earlier {
			packages = append(packages, map[string]any{"path": group.path, "files": nonNil(group.files)})
		}
		packageMap = map[string]any{"base": pkg.Base, "path": pkg.Path, "packages": packages}
		// The last directory package upstream compiles is the program a later
		// link/execute phase of this test acts on (rundir, errorcheckandrundir).
		// The program carries upstream's identity and every earlier group —
		// a single-package program too (S165.0, D8): its execute phase runs
		// under the same `-p main` its compile phase was checked under.
		program := backendProgram{files: append([]string(nil), compileInputs...), mapArgs: append([]string(nil), mapArgs...), identity: identity}
		backendPrograms.Store(key, program)
		deviations = append(deviations,
			fmt.Sprintf("upstream package identity -D %s -p %s and the %d earlier package(s) of this test are handed to Bash++ as an explicit package map; relative imports are never resolved on disk", pkg.Base, pkg.Path, len(earlier)))
	}

	switch mode {
	case "interpreted":
		checkArgs := append([]string{"--bashpp", "--source=go", "--check"}, mapArgs...)
		checkArgs = append(checkArgs, fileArgs...)
		switch {
		case assembly:
			step.backendErr = fmt.Errorf("Bash++ backend unsupported: assembly is a compiler artifact; interpreted mode has no asmcheck meaning")
			t.backendEvent(mode, action, phase, "unsupported", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
				append(deviations, "asmcheck compares generated assembly; only compiled mode produces one"))
			return
		case diagnostics:
			if optimizerDiagnosticFlags(recipeFlags) {
				step.backendErr = fmt.Errorf("Bash++ backend unsupported: optimizer diagnostics are a compiler artifact; the check interface has no inlining, escape-analysis or SSA meaning")
				t.backendEvent(mode, action, phase, "unsupported", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
					append(deviations, "optimizer diagnostics are a compiler artifact; the check interface has no inlining, escape-analysis or SSA meaning"))
				return
			}
			*step.cmd = *directSourceCommand(step.cmd, tool, checkArgs...)
			t.backendEvent(mode, action, phase, "check-diagnostics", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
				append(deviations,
					"the Bash++ check interface reports every diagnostic as file:line:col: message on the exact upstream input path; upstream errorCheck applies its own expectations unchanged",
					"compiler recipe flags (-e, -d=panic, -C, -p) have no check-interface representation and remain explicit evidence; nothing is executed"))
			return
		case directory:
			*step.cmd = *directSourceCommand(step.cmd, tool, checkArgs...)
			t.backendEventMap(mode, action, phase, "check-package-map", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil,
				append(deviations, "directory package phase stops after Bash++ check against the explicit package map; no init or main is executed"), packageMap)
			return
		case compileOnly || buildOnly || directoryBuild:
			*step.cmd = *directSourceCommand(step.cmd, tool, checkArgs...)
			checkDeviations := append(append([]string(nil), deviations...),
				"the Bash++ check interface does not accept compiler recipe flags; they remain explicit evidence",
				"compile-only phase stops after Bash++ check; no init or main is executed")
			if directoryBuild {
				checkDeviations = append(append([]string(nil), deviations...),
					"the Bash++ check interface has no compiler or artifact semantics; upstream's direct compile flags and the go.o object remain explicit evidence only",
					"directory build phase stops after Bash++ check; no object is produced and no init or main is executed")
				backendPrograms.Store(t.eventName(), backendProgram{files: append([]string(nil), compileInputs...), mapArgs: append([]string(nil), mapArgs...), identity: identity, object: filepath.Base(compilerOutput(nativeArgv, compileInputs[0], step.cmd.Dir))})
			}
			if buildOnly {
				checkDeviations = append(append([]string(nil), deviations...),
					"the Bash++ check interface has no compiler or artifact semantics; upstream go-command recipe flags and the cwd a.exe artifact remain explicit evidence only",
					"build-only phase stops after Bash++ check; no artifact is produced and no init or main is executed")
				backendPrograms.Store(t.eventName(), backendProgram{files: append([]string(nil), compileInputs...)})
			}
			t.backendEvent(mode, action, phase, "check-only", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil, checkDeviations)
			return
		}
		runArgs := append([]string{"--bashpp", "--source=go"}, fileArgs...)
		if len(programArgv) != 0 {
			runArgs = append(runArgs, "--")
			runArgs = append(runArgs, programArgv...)
		}
		runDeviations := deviations
		if len(recipeFlags) != 0 {
			runDeviations = append(append([]string(nil), deviations...),
				"upstream go-command recipe flags have no representation in the direct Go-source interpreter and remain explicit evidence only")
		}
		*step.cmd = *shellCommand(step.cmd,
			append([]string{tool}, checkArgs...),
			append([]string{tool}, runArgs...))
		t.backendEventMap(mode, action, phase, "check-then-run", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil, runDeviations, moduleMap)

	case "compiled":
		goTool := os.Getenv("BASHPP_TESTDIR_GO")
		shellrt := os.Getenv("BASHPP_SHELLRT_ROOT")
		if goTool == "" || shellrt == "" {
			step.backendErr = fmt.Errorf("compiled Bash++ backend requires BASHPP_TESTDIR_GO and caller-supplied BASHPP_SHELLRT_ROOT")
			t.backendEvent(mode, action, phase, "configuration-error", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil, deviations)
			return
		}
		moduleDir, err := backendModule(t, shellrt)
		if err != nil {
			step.backendErr = err
			t.backendEvent(mode, action, phase, "module-failed", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil, deviations)
			return
		}
		generated := filepath.Join(moduleDir, "main.go")
		artifact := filepath.Join(moduleDir, "program")
		sourceMap := filepath.Join(moduleDir, "main.go.map")
		transpileArgs := append([]string{"transpile", "--bashpp", "--source=go"}, mapArgs...)
		transpileArgs = append(transpileArgs, fileArgs...)
		transpileArgs = append(transpileArgs, "-o", generated)
		if buildOnly {
			transpileArgs = append(transpileArgs, "--map", sourceMap)
			built := filepath.Join(step.cmd.Dir, "a.exe")
			buildArgs := []string{goTool, "build", "-C", moduleDir}
			buildArgs = append(buildArgs, recipeFlags...)
			buildArgs = append(buildArgs, "-o", built, ".")
			*step.cmd = *shellCommand(step.cmd,
				append([]string{tool}, transpileArgs...),
				buildArgs)
			step.artifacts = []string{generated, built}
			step.maps = []string{sourceMap}
			backendPrograms.Store(t.eventName(), backendProgram{files: append([]string(nil), compileInputs...), artifact: built})
			t.backendEvent(mode, action, phase, "transpile-build-only", compileInputs, programArgv, recipeFlags, nativeArgv,
				step.artifacts, step.maps,
				append(deviations,
					"upstream go-command recipe flags are passed verbatim to the pinned Go build of the generated module; they are never rewrapped as compile-tool or all= flags",
					"the upstream-selected environment, including any runenv GOEXPERIMENT, is preserved unchanged",
					artifactUse(action)))
			return
		}
		if assembly {
			transpileArgs = append(transpileArgs, "--map", sourceMap)
			gcflags, buildFlags := asmBuildArgs(recipeFlags)
			buildArgs := []string{goTool, "build", "-C", moduleDir, "-gcflags=" + gcflags}
			buildArgs = append(buildArgs, buildFlags...)
			buildArgs = append(buildArgs, "-o", artifact, filepath.Base(generated))
			listing := filepath.Join(moduleDir, "asm-listing.txt")
			transpileCommand := append([]string{tool}, transpileArgs...)
			quotedTranspile := make([]string, len(transpileCommand))
			for i, word := range transpileCommand {
				quotedTranspile[i] = shellQuote(word)
			}
			quotedBuild := make([]string, len(buildArgs))
			for i, word := range buildArgs {
				quotedBuild[i] = shellQuote(word)
			}
			asmScript := strings.Join([]string{
				"set -e",
				strings.Join(quotedTranspile, " "),
				"set +e",
				strings.Join(quotedBuild, " ") + " > " + shellQuote(listing) + " 2>&1",
				"rc=$?",
				"sed -E 's/(\\.go:[0-9]+)\\[[^]]*\\]\\)/\\1)/' " + shellQuote(listing),
				"exit \"$rc\"",
			}, "\n")
			*step.cmd = *directSourceCommand(step.cmd, "/bin/sh", "-c", asmScript)
			step.artifacts = []string{generated, artifact, listing}
			step.maps = []string{sourceMap}
			t.backendEvent(mode, action, phase, "transpile-build-assembly", compileInputs, programArgv, recipeFlags, nativeArgv,
				step.artifacts, step.maps,
				append(deviations,
					"the generated module is built with the upstream -S=2 listing request and the upstream asmcheck flag merge; its //line directives cite the exact upstream input path",
					"the generated file is compiled as a file argument, so its symbols are qualified as command-line-arguments like upstream's compilation of the original",
					"the -S listing's physical-position suffix (origin:line[generated:line]) is removed before upstream asmCheck indexes it, so the unchanged matcher keys generated code by origin file:line; the raw listing is retained as an artifact",
					"the upstream-selected GOOS/GOARCH environment is preserved unchanged; the program is never executed"))
			return
		}
		if compileOnly || diagnostics || directory || directoryBuild {
			transpileArgs = append(transpileArgs, "--map", sourceMap)
			compilerArgv, err := directCompileCommand(nativeArgv, compileInputs, generated)
			if err != nil {
				step.backendErr = err
				t.backendEvent(mode, action, phase, "configuration-error", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil, deviations)
				return
			}
			artifact = compilerOutput(compilerArgv, generated, step.cmd.Dir)
			*step.cmd = *shellCommand(step.cmd,
				append([]string{tool}, transpileArgs...),
				compilerArgv)
			step.artifacts = []string{generated, artifact}
			step.maps = []string{sourceMap}
			compileDeviations := append(deviations,
				"the pinned compiler is invoked directly on the generated Go file with upstream's exact compile flags, -p resolution, and importcfg; cmd/go is not involved and cannot add -complete",
				"the backend event records compiler_argv, the exact executed compiler command with only the source input replaced by the generated file")
			disposition := "transpile-compile-only"
			switch {
			case diagnostics:
				disposition = "transpile-compile-diagnostics"
				compileDeviations = append(compileDeviations, "transpile and compiler diagnostics are emitted on the exact upstream input path via //line directives; upstream errorCheck applies its own expectations unchanged")
			case directory:
				disposition = "transpile-compile-package-map"
				compileDeviations = append(compileDeviations, "the direct compiler reuses upstream's directory importcfg, including the object paths produced by earlier generated package phases")
				if value, ok := backendPrograms.Load(t.eventName()); ok {
					program := value.(backendProgram)
					program.artifact = artifact
					program.compilerArgv = append([]string(nil), compilerArgv...)
					backendPrograms.Store(t.eventName(), program)
				}
			case directoryBuild:
				disposition = "transpile-compile-directory-build"
				compileDeviations = append(compileDeviations,
					"the direct compiler keeps upstream's -o object, -D . and, with .s companions, -asmhdr go_asm.h / -symabis symabis in the upstream working directory; the generated file's constants and declarations are what the assembler sees",
					"the object is the program this test's pack, link and execute phases act on")
				backendPrograms.Store(t.eventName(), backendProgram{files: append([]string(nil), compileInputs...), mapArgs: append([]string(nil), mapArgs...), identity: identity, artifact: artifact, compilerArgv: append([]string(nil), compilerArgv...), object: filepath.Base(artifact)})
			}
			t.backendEventCompiler(mode, action, phase, disposition, compileInputs, programArgv, recipeFlags, nativeArgv,
				step.artifacts, step.maps, compileDeviations, packageMap, compilerArgv)
			return
		}
		// An ordinary run root with recipe flags reaches this path through
		// upstream's `go run <flags> <file>` branch; the flags are go-command
		// flags and are passed verbatim to the pinned build of the generated
		// module, exactly as the build action does (never rewrapped).
		buildArgs := []string{goTool, "build", "-C", moduleDir}
		buildArgs = append(buildArgs, recipeFlags...)
		buildArgs = append(buildArgs, "-o", artifact, ".")
		runDeviations := append(deviations, "generated source is built in a temporary module with a caller-supplied mvdan.cc/sh/v3 replacement")
		if len(recipeFlags) != 0 {
			runDeviations = append(runDeviations, "upstream go-command recipe flags are passed verbatim to the pinned Go build of the generated module; they are never rewrapped as compile-tool or all= flags")
		}
		*step.cmd = *shellCommand(step.cmd,
			append([]string{tool}, transpileArgs...),
			buildArgs,
			append([]string{artifact}, programArgv...))
		t.backendEventMap(mode, action, phase, "transpile-build-run", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil, runDeviations, moduleMap)

	default:
		step.backendErr = fmt.Errorf("unsupported Bash++ backend mode %q", mode)
		t.backendEvent(mode, action, phase, "configuration-error", compileInputs, programArgv, recipeFlags, nativeArgv, nil, nil, deviations)
	}
}

// Sprint: #165; Story: S165.0; Story-ID: 1528c3c2b1df
func TestAssemblyInputsAreOnlyAssemblySources(t *testing.T) {
	if !assemblyInputs([]string{"/t/a.s"}) || !assemblyInputs([]string{"/t/a.s", "/t/b_amd64.s"}) {
		t.Error("assembly-only inputs must be recognized")
	}
	for _, inputs := range [][]string{nil, {}, {"/t/main.go"}, {"/t/a.s", "/t/main.go"}, {"."}} {
		if assemblyInputs(inputs) {
			t.Errorf("assemblyInputs(%v) = true", inputs)
		}
	}
}

// TestProgramOwnsLinkInputs pins what pack and link may adopt: upstream's own
// object name for the seam's compiler artifact, the objects the seam
// assembled natively (pack only), and nothing else.
func TestProgramOwnsLinkInputs(t *testing.T) {
	built := backendProgram{files: []string{"/t/main.go"}, artifact: "/w/go.o", object: "go.o", assembled: []string{"/w/asm.o"}}
	if !built.owns([]string{"go.o", "asm.o"}, true) || !built.owns([]string{"go.o"}, true) || !built.owns([]string{"go.o"}, false) {
		t.Error("the compiler object and the assembled objects must be owned")
	}
	for _, tt := range []struct {
		inputs []string
		pack   bool
	}{
		{[]string{"asm.o"}, true},           // no compiler object
		{[]string{"go.o", "other.o"}, true}, // an object the seam never produced
		{[]string{"go.o", "asm.o"}, false},  // link takes one input
		{[]string{"go.o", "go.o"}, true},    // the compiler object once
		{[]string{"main.o"}, false},         // the .go -> .o rule does not apply once -o named the object
		{nil, true},
	} {
		if built.owns(tt.inputs, tt.pack) {
			t.Errorf("owns(%v, pack=%v) = true", tt.inputs, tt.pack)
		}
	}
	// Without -o the directory compile keeps the .go -> .o rule.
	directory := backendProgram{files: []string{"/t/b.go", "/t/a.go"}, artifact: "/w/main.o"}
	if !directory.owns([]string{"b.o"}, false) || directory.owns([]string{"a.o"}, false) || directory.owns([]string{"b.o", "asm.o"}, true) {
		t.Error("the directory rule must adopt exactly the first file's object")
	}
	if (backendProgram{}).owns([]string{"go.o"}, false) {
		t.Error("a program with no files owns nothing")
	}
}

// TestCompilerIdentityReadsUpstreamArgv pins the identity handoff (S165.0,
// D8): the -D / -p of upstream's own direct compile argv, last occurrence
// winning as the compiler's flag parsing has it, compile inputs excluded by
// name; every other native command carries none.
func TestCompilerIdentityReadsUpstreamArgv(t *testing.T) {
	for _, tt := range []struct {
		argv   []string
		inputs []string
		want   backendIdentity
	}{
		// compileFile / errorcheck / errorcheckoutput: -p=p
		{[]string{"go", "tool", "compile", "-e", "-p=p", "-importcfg=/i", "/t/x.go"}, []string{"/t/x.go"}, backendIdentity{Path: "p"}},
		{[]string{"go", "tool", "compile", "-p=p", "-d=panic", "-C", "-e", "-importcfg=/i", "-0", "-m", "-l", "/t/x.go"}, []string{"/t/x.go"}, backendIdentity{Path: "p"}},
		// a recipe's own -p after upstream's: the last one is gc's
		{[]string{"go", "tool", "compile", "-p=p", "-d=panic", "-C", "-e", "-importcfg=/i", "-+", "-p=runtime", "/t/x.go"}, []string{"/t/x.go"}, backendIdentity{Path: "runtime"}},
		{[]string{"go", "tool", "compile", "-p=p", "-e", "-importcfg=/i", "-p", "example.com/dotted", "/t/x.go"}, []string{"/t/x.go"}, backendIdentity{Path: "example.com/dotted"}},
		// compileInDir: -D test -p=main / -o a.a -p test/a
		{[]string{"go", "tool", "compile", "-e", "-D", "test", "-importcfg=/i", "-p=main", "/t/d/main.go"}, []string{"/t/d/main.go"}, backendIdentity{Base: "test", Path: "main"}},
		{[]string{"go", "tool", "compile", "-e", "-D", "test", "-importcfg=/i", "-o", "test/a.a", "-p", "test/a", "/t/d/a.go", "/t/d/b.go"}, []string{"/t/d/a.go", "/t/d/b.go"}, backendIdentity{Base: "test", Path: "test/a"}},
		// builddir: -p=main -D . -o go.o -asmhdr go_asm.h -symabis symabis
		{[]string{"go", "tool", "compile", "-p=main", "-e", "-D", ".", "-importcfg=/i", "-o", "go.o", "-asmhdr", "go_asm.h", "-symabis", "symabis", "/t/d/main.go"}, []string{"/t/d/main.go"}, backendIdentity{Base: ".", Path: "main"}},
		// an input named like a flag value is excluded by name, never read
		{[]string{"go", "tool", "compile", "-e", "-importcfg=/i", "-p", "-p", "-p"}, []string{"-p"}, backendIdentity{}},
		// an empty or dangling -p is no identity
		{[]string{"go", "tool", "compile", "-p=", "-e", "/t/x.go"}, []string{"/t/x.go"}, backendIdentity{}},
		{[]string{"go", "tool", "compile", "-e", "/t/x.go", "-p"}, []string{"/t/x.go"}, backendIdentity{}},
		// go run / go build / go tool asm / go tool link / a.exe: none
		{[]string{"go", "run", "-gcflags=-p=x", "/t/x.go"}, []string{"/t/x.go"}, backendIdentity{}},
		{[]string{"go", "build", "-o", "a.exe", "/t/x.go"}, []string{"/t/x.go"}, backendIdentity{}},
		{[]string{"go", "tool", "asm", "-p=main", "-gensymabis", "-o", "symabis", "/t/d/f.s"}, []string{"/t/d/f.s"}, backendIdentity{}},
		{[]string{"go", "tool", "link", "-o", "a.exe", "-importcfg=/i", "main.o"}, []string{"main.o"}, backendIdentity{}},
		{[]string{"/t/a.exe"}, nil, backendIdentity{}},
	} {
		if got := compilerIdentity(tt.argv, tt.inputs); got != tt.want {
			t.Errorf("compilerIdentity(%v) = %+v, want %+v", tt.argv, got, tt.want)
		}
	}
	if got := (backendIdentity{Base: "test", Path: "main"}).args(); strings.Join(got, " ") != "--go-import-base test --go-import-path main" {
		t.Errorf("args() = %v", got)
	}
	if got := (backendIdentity{Path: "p"}).args(); strings.Join(got, " ") != "--go-import-path p" {
		t.Errorf("args() = %v", got)
	}
	if got := (backendIdentity{}).args(); len(got) != 0 {
		t.Errorf("args() of no identity = %v", got)
	}
}
