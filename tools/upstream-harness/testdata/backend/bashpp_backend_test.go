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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
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

func replaceCommandEnv(env []string, name, value string) []string {
	if env == nil {
		env = os.Environ()
	}
	prefix := name + "="
	result := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if !strings.HasPrefix(entry, prefix) {
			result = append(result, entry)
		}
	}
	return append(result, prefix+value)
}

func interpretedRecipeGOFLAGS(recipeFlags []string) (string, bool) {
	if len(recipeFlags) == 1 && recipeFlags[0] == "-gcflags=-d=converthash=qy" {
		return recipeFlags[0], true
	}
	return "", false
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

type companionFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Role   string `json:"role,omitempty"`
}

func companionProof(paths []string) ([]companionFile, error) {
	return companionProofRole(paths, "")
}

func companionProofRole(paths []string, role string) ([]companionFile, error) {
	proofs := make([]companionFile, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(data)
		proofs = append(proofs, companionFile{Path: path, SHA256: hex.EncodeToString(digest[:]), Role: role})
	}
	return proofs, nil
}

func absPackageFiles(dir string, files []string) []string {
	abs := make([]string, 0, len(files))
	for _, f := range files {
		abs = append(abs, filepath.Join(dir, f))
	}
	return abs
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

// rememberDirectoryPackage returns the packages preceding this compile phase.
func rememberDirectoryPackage(key string, current packageGroup) []packageGroup {
	var earlier []packageGroup
	if value, ok := backendPackages.Load(key); ok {
		earlier = value.([]packageGroup)
	}
	// errorcheckandrundir repeats upstream's ordered package list for its
	// run pass. Start again at the repeated identity: later packages belong
	// to the previous pass and may import this package themselves.
	for i, group := range earlier {
		if group.path == current.path {
			earlier = earlier[:i]
			break
		}
	}
	backendPackages.Store(key, append(append([]packageGroup(nil), earlier...), packageGroup{path: current.path, files: append([]string(nil), current.files...)}))
	return earlier
}

// directoryImportClosure selects only imported source-map entries. Native
// importcfg may name an object whose compilation failed: it is loaded only if
// imported. Passing every attempted source package to an eager checker would
// incorrectly re-check an unrelated rejected package in a later phase.
// Parsing only import headers never changes the diagnostic checker's verdict;
// if a header cannot be read, retain the full map so the checker owns the error.
func directoryImportClosure(base string, files []string, available []packageGroup) []packageGroup {
	byPath := make(map[string]packageGroup, len(available))
	for _, group := range available {
		byPath[group.path] = group
	}
	reached := map[string]bool{}
	var visit func([]string) bool
	visit = func(files []string) bool {
		for _, name := range files {
			f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
			if err != nil {
				return false
			}
			for _, spec := range f.Imports {
				imported, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return false
				}
				if strings.HasPrefix(imported, "./") || strings.HasPrefix(imported, "../") {
					imported = path.Join(base, imported)
				}
				group, exists := byPath[imported]
				if !exists || reached[imported] {
					continue
				}
				reached[imported] = true
				if !visit(group.files) {
					return false
				}
			}
		}
		return true
	}
	if !visit(files) {
		return available
	}
	var selected []packageGroup
	for _, group := range available {
		if reached[group.path] {
			selected = append(selected, group)
		}
	}
	return selected
}

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
	ImportPath     string
	Name           string
	Dir            string
	GoFiles        []string
	SFiles         []string
	CgoFiles       []string
	IgnoredGoFiles []string
	Standard       bool
	Module         *struct {
		Path string
		Main bool
	}
	Error *struct {
		Err string
	}
}

func goEnvCGOEnabled(goTool, dir string, env []string) (bool, error) {
	cmd := exec.Command(goTool, "env", "CGO_ENABLED")
	cmd.Dir, cmd.Env = dir, env
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "1", nil
}

func importsC(path string) bool {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return false
	}
	for _, spec := range f.Imports {
		if strings.Trim(spec.Path.Value, `"`) == "C" {
			return true
		}
	}
	return false
}

func ignoredCgoFiles(pkg goListPackage) []string {
	var files []string
	for _, name := range pkg.IgnoredGoFiles {
		if importsC(filepath.Join(pkg.Dir, name)) {
			files = append(files, name)
		}
	}
	return files
}

// resolveModuleProgram asks the pinned go command what "." is in dir: the main
// package's Go files (absolute) and every in-module dependency as an ordered
// --go-package entry (go list -deps emits dependencies before dependents).
// Assembly and cgo sources selected by go list are authenticated companions of
// their same package. Cgo sources are accepted only when the selected Go
// environment has CGO_ENABLED=1; otherwise they remain an explicit refusal.
func resolveModuleProgram(goTool, dir string, env []string) (files, mapArgs []string, record map[string]any, err error) {
	if goTool == "" {
		return nil, nil, nil, fmt.Errorf("resolving a module program requires BASHPP_TESTDIR_GO")
	}
	cgoEnabled, envErr := goEnvCGOEnabled(goTool, dir, env)
	if envErr != nil {
		return nil, nil, nil, fmt.Errorf("go env CGO_ENABLED failed in %s: %v", dir, envErr)
	}
	cmd := exec.Command(goTool, "list", "-json", "-deps", "-e", ".")
	cmd.Dir, cmd.Env = dir, env
	out, err := cmd.Output()
	record = map[string]any{"go_list": append([]string{goTool}, cmd.Args[1:]...), "dir": dir, "cgo_enabled": cgoEnabled}
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
		if !cgoEnabled {
			if cgoFiles := ignoredCgoFiles(pkg); len(cgoFiles) != 0 {
				return nil, nil, record, fmt.Errorf("module package %s has cgo inputs %v with CGO_ENABLED=0", pkg.ImportPath, cgoFiles)
			}
		}
		if len(pkg.CgoFiles) != 0 && !cgoEnabled {
			return nil, nil, record, fmt.Errorf("module package %s has cgo inputs %v with CGO_ENABLED=0", pkg.ImportPath, pkg.CgoFiles)
		}
		if pkg.Error != nil {
			return nil, nil, record, fmt.Errorf("go list package %s: %s", pkg.ImportPath, pkg.Error.Err)
		}
		abs := absPackageFiles(pkg.Dir, append(append([]string(nil), pkg.GoFiles...), pkg.CgoFiles...))
		companionPaths := absPackageFiles(pkg.Dir, pkg.SFiles)
		cgoPaths := absPackageFiles(pkg.Dir, pkg.CgoFiles)
		for _, path := range companionPaths {
			rel, relErr := filepath.Rel(pkg.Dir, path)
			if relErr != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
				return nil, nil, record, fmt.Errorf("module package %s has unauthenticated assembly companion %q", pkg.ImportPath, path)
			}
		}
		companionProofs, proofErr := companionProof(companionPaths)
		if proofErr != nil {
			return nil, nil, record, fmt.Errorf("module package %s assembly companion proof failed: %v", pkg.ImportPath, proofErr)
		}
		cgoProofs, proofErr := companionProofRole(cgoPaths, "cgo")
		if proofErr != nil {
			return nil, nil, record, fmt.Errorf("module package %s cgo companion proof failed: %v", pkg.ImportPath, proofErr)
		}
		companionProofs = append(companionProofs, cgoProofs...)
		companionPaths = append(companionPaths, cgoPaths...)
		unit := map[string]any{"path": pkg.ImportPath, "files": abs, "dir": pkg.Dir}
		if len(companionProofs) != 0 {
			unit["companions"] = companionPaths
			unit["companion_proof"] = companionProofs
		}
		if pkg.Name == "main" && pkg.Dir == dir {
			p := pkg
			main = &p
			files = abs
			if len(companionProofs) != 0 {
				record["companions"] = companionPaths
				record["companion_proof"] = companionProofs
			}
			continue
		}
		mapArgs = append(mapArgs, "--go-package", pkg.ImportPath+"="+strings.Join(abs, ","))
		packages = append(packages, unit)
	}
	if main == nil || len(files) == 0 {
		return nil, nil, record, fmt.Errorf("go list found no main package in %s", dir)
	}
	if len(packages) != 0 {
		mapArgs = append([]string{"--go-import-path", main.ImportPath}, mapArgs...)
	}
	record["base"] = ""
	record["path"] = main.ImportPath
	record["module"] = main.Module.Path
	record["packages"] = packages
	return files, mapArgs, record, nil
}

// nativeModuleCommands emits each go-list unit separately, preserving the
// original module identity and dependency order instead of flattening types.
func nativeModuleCommands(dir, tool, shellrt string, record map[string]any, mainFiles []string) (commands [][]string, artifacts, maps []string, mainDir string, err error) {
	module, _ := record["module"].(string)
	mainPath, _ := record["path"].(string)
	if module == "" || strings.ContainsAny(module, "\\\"\n\r\t ") {
		return nil, nil, nil, "", fmt.Errorf("invalid module identity %q", module)
	}
	packages, _ := record["packages"].([]map[string]any)
	units := append(append([]map[string]any(nil), packages...), map[string]any{"path": mainPath, "files": mainFiles})
	if companions, _ := record["companions"].([]string); len(companions) != 0 {
		units[len(units)-1]["companions"] = companions
	}
	if proofs, _ := record["companion_proof"].([]companionFile); len(proofs) != 0 {
		units[len(units)-1]["companion_proof"] = proofs
	}
	var deps []string
	seen := map[string]bool{}
	for _, unit := range units {
		path, _ := unit["path"].(string)
		files, _ := unit["files"].([]string)
		companions, _ := unit["companions"].([]string)
		proofs, _ := unit["companion_proof"].([]companionFile)
		cgoCompanions := map[string]bool{}
		for _, proof := range proofs {
			if proof.Role == "cgo" {
				cgoCompanions[proof.Path] = true
			}
		}
		rel := strings.TrimPrefix(path, module)
		if path != module && !strings.HasPrefix(path, module+"/") || strings.Contains(rel, "\\") || strings.Contains(rel, "//") || strings.Contains(rel, "/../") || strings.HasSuffix(rel, "/..") || rel != "" && filepath.Clean(strings.TrimPrefix(rel, "/")) != strings.TrimPrefix(rel, "/") || seen[path] || len(files) == 0 {
			return nil, nil, nil, "", fmt.Errorf("invalid module unit %q", path)
		}
		seen[path] = true
		unitDir := filepath.Join(dir, strings.TrimPrefix(rel, "/"))
		if err := os.MkdirAll(unitDir, 0o700); err != nil {
			return nil, nil, nil, "", err
		}
		for _, companion := range companions {
			if cgoCompanions[companion] {
				continue
			}
			relCompanion, err := filepath.Rel(filepath.Dir(files[0]), companion)
			if err != nil || relCompanion == "." || strings.HasPrefix(relCompanion, ".."+string(filepath.Separator)) || relCompanion == ".." || filepath.IsAbs(relCompanion) {
				return nil, nil, nil, "", fmt.Errorf("assembly companion %q is not inside package unit %q", companion, path)
			}
			target := filepath.Join(unitDir, relCompanion)
			if filepath.Base(target) == "main.go" {
				return nil, nil, nil, "", fmt.Errorf("assembly companion %q would overwrite generated source", companion)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				return nil, nil, nil, "", err
			}
			in, err := os.Open(companion)
			if err != nil {
				return nil, nil, nil, "", err
			}
			out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
			if err != nil {
				in.Close()
				return nil, nil, nil, "", err
			}
			_, copyErr := io.Copy(out, in)
			closeErr := out.Close()
			in.Close()
			if copyErr != nil {
				return nil, nil, nil, "", copyErr
			}
			if closeErr != nil {
				return nil, nil, nil, "", closeErr
			}
		}
		generated := filepath.Join(unitDir, "main.go")
		mapFile := generated + ".map"
		args := []string{tool, "transpile", "--bashpp", "--source=go", "--go-native-unit", "--go-import-path", path}
		args = append(args, deps...)
		args = append(args, goFileArgs(files)...)
		args = append(args, "-o", generated, "--map", mapFile)
		commands = append(commands, args)
		artifacts = append(artifacts, generated)
		maps = append(maps, mapFile)
		deps = append(deps, "--go-package", path+"="+strings.Join(files, ","))
		mainDir = unitDir
	}
	body := fmt.Sprintf("module %s\n\ngo 1.27\n\nrequire mvdan.cc/sh/v3 v3.13.1\nreplace mvdan.cc/sh/v3 => %s\n", module, shellrt)
	err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(body), 0o600)
	return
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
		available := rememberDirectoryPackage(key, packageGroup{path: pkg.Path, files: compileInputs})
		earlier := directoryImportClosure(pkg.Base, compileInputs, available)
		mapArgs = []string{"--go-import-base", pkg.Base, "--go-import-path", pkg.Path}
		for _, group := range earlier {
			mapArgs = append(mapArgs, "--go-package", group.path+"="+strings.Join(group.files, ","))
		}
		packages := make([]map[string]any, 0, len(earlier))
		for _, group := range earlier {
			packages = append(packages, map[string]any{"path": group.path, "files": nonNil(group.files)})
		}
		availableMap := make([]map[string]any, 0, len(available))
		for _, group := range available {
			availableMap = append(availableMap, map[string]any{"path": group.path, "files": nonNil(group.files)})
		}
		packageMap = map[string]any{"base": pkg.Base, "path": pkg.Path, "packages": packages, "available": availableMap, "selection": "import-closure"}
		// The last directory package upstream compiles is the program a later
		// link/execute phase of this test acts on (rundir, errorcheckandrundir).
		// The program carries upstream's identity and its import closure —
		// a single-package program too (S165.0, D8): its execute phase runs
		// under the same `-p main` its compile phase was checked under.
		program := backendProgram{files: append([]string(nil), compileInputs...), mapArgs: append([]string(nil), mapArgs...), identity: identity}
		backendPrograms.Store(key, program)
		deviations = append(deviations,
			fmt.Sprintf("upstream package identity -D %s -p %s and the %d transitively imported earlier package(s) of this test are handed to Bash++ as an explicit package map; the complete available map is recorded separately; relative imports are never resolved on disk", pkg.Base, pkg.Path, len(earlier)))
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
		goFlags, transportPolicy := interpretedRecipeGOFLAGS(recipeFlags)
		if transportPolicy {
			runDeviations = append(append([]string(nil), deviations...),
				"the exact supported unscoped compiler conversion policy is transported to both direct Go-source commands as GOFLAGS=-gcflags=-d=converthash=qy; recipe_flags retains the upstream spelling")
		} else if len(recipeFlags) != 0 {
			runDeviations = append(append([]string(nil), deviations...),
				"upstream go-command recipe flags have no representation in the direct Go-source interpreter and remain explicit evidence only")
		}
		*step.cmd = *shellCommand(step.cmd,
			append([]string{tool}, checkArgs...),
			append([]string{tool}, runArgs...))
		if transportPolicy {
			step.cmd.Env = replaceCommandEnv(step.cmd.Env, "GOFLAGS", goFlags)
		}
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
		if moduleMap != nil {
			commands, artifacts, maps, mainDir, err := nativeModuleCommands(moduleDir, tool, shellrt, moduleMap, sourceFiles)
			if err != nil {
				step.backendErr = err
				return
			}
			buildArgs := append([]string{goTool, "build", "-C", mainDir}, recipeFlags...)
			buildArgs = append(buildArgs, "-o", artifact, ".")
			commands = append(commands, buildArgs, append([]string{artifact}, programArgv...))
			*step.cmd = *shellCommand(step.cmd, commands...)
			step.artifacts = append(artifacts, artifact)
			step.maps = maps
			t.backendEventMap(mode, action, phase, "transpile-build-run", compileInputs, programArgv, recipeFlags, nativeArgv, step.artifacts, step.maps,
				append(deviations, "each module package is emitted as a native unit at its original import path; upstream go-command recipe flags and environment are preserved"), moduleMap)
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

// Replaying errorcheckandrundir must not import a package into itself or
// carry the previous pass's main package backward into its dependencies.
func TestDirectoryPackageReplay(t *testing.T) {
	key := t.Name()
	defer backendPackages.Delete(key)
	for round := 0; round < 2; round++ {
		for i, path := range []string{"test/a", "test/b", "main"} {
			earlier := rememberDirectoryPackage(key, packageGroup{path: path, files: []string{path + ".go"}})
			if len(earlier) != i {
				t.Fatalf("round %d path %s: previous=%v", round, path, earlier)
			}
			for j, g := range earlier {
				if g.path != []string{"test/a", "test/b"}[j] {
					t.Fatalf("dependency order=%v", earlier)
				}
			}
		}
	}
}

func TestDirectoryPackageReplacementAndIsolation(t *testing.T) {
	key, other := t.Name(), t.Name()+"/other"
	defer backendPackages.Delete(key)
	defer backendPackages.Delete(other)
	original := []string{"old.go"}
	rememberDirectoryPackage(key, packageGroup{path: "test/a", files: original})
	original[0] = "mutated.go"
	before := rememberDirectoryPackage(key, packageGroup{path: "test/b", files: []string{"b.go"}})
	if before[0].files[0] != "old.go" {
		t.Fatal("input slice aliases registry")
	}
	if got := rememberDirectoryPackage(other, packageGroup{path: "main", files: []string{"main.go"}}); len(got) != 0 {
		t.Fatal("root state leaked")
	}
	// A recompile supplies the exact new unit. Never concatenate conflicting
	// old/new definitions; duplicate definitions inside this unit remain input
	// to the unchanged checker/compiler, rather than being deduplicated here.
	replacement := []string{"new.go", "conflict.go", "new.go"}
	if got := rememberDirectoryPackage(key, packageGroup{path: "test/a", files: replacement}); len(got) != 0 {
		t.Fatalf("stale pass dependencies=%v", got)
	}
	replacement[0] = "changed.go"
	got := rememberDirectoryPackage(key, packageGroup{path: "main", files: []string{"main.go"}})
	if len(got) != 1 || strings.Join(got[0].files, ",") != "new.go,conflict.go,new.go" {
		t.Fatalf("replacement changed or merged: %v", got)
	}
}

func TestDirectoryPackageRecompileReadsCurrentSource(t *testing.T) {
	key := t.Name()
	defer backendPackages.Delete(key)
	path := filepath.Join(t.TempDir(), "a.go")
	for _, source := range []string{"package a; const Value = 1", "package a; const Value = 2"} {
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		rememberDirectoryPackage(key, packageGroup{path: "test/a", files: []string{path}})
		dependencies := rememberDirectoryPackage(key, packageGroup{path: "main", files: []string{"main.go"}})
		if len(dependencies) != 1 || len(dependencies[0].files) != 1 {
			t.Fatalf("dependencies=%v", dependencies)
		}
		got, err := os.ReadFile(dependencies[0].files[0])
		if err != nil || string(got) != source {
			t.Fatalf("source=%q err=%v", got, err)
		}
	}
}

func TestNativeModuleCommands(t *testing.T) {
	dir := t.TempDir()
	record := map[string]any{"module": "example.test/m", "path": "example.test/m/cmd/app", "packages": []map[string]any{{"path": "example.test/m/a", "files": []string{"/original/a.go"}}}}
	commands, artifacts, maps, mainDir, err := nativeModuleCommands(dir, "bashy", "/runtime", record, []string{"/original/main.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 2 || len(artifacts) != 2 || len(maps) != 2 || mainDir != filepath.Join(dir, "cmd/app") {
		t.Fatalf("unexpected units: %v %v %v %s", commands, artifacts, maps, mainDir)
	}
	first := strings.Join(commands[0], " ")
	second := strings.Join(commands[1], " ")
	if !strings.Contains(first, "--go-native-unit --go-import-path example.test/m/a") || strings.Contains(first, "--go-package") || !strings.Contains(second, "--go-package example.test/m/a=/original/a.go") || strings.Contains(second, "--go-import-base") {
		t.Fatalf("unit routing: %v", commands)
	}
	body, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(body), "module example.test/m\n") {
		t.Fatalf("module: %s", body)
	}
	for _, path := range []string{"other/a", "example.test/m/../escape", "example.test/m/a/..", "example.test/m//a", "example.test/m/a"} {
		record["path"] = path
		if _, _, _, _, err := nativeModuleCommands(t.TempDir(), "bashy", "/runtime", record, []string{"/original/main.go"}); err == nil {
			t.Errorf("accepted invalid/duplicate unit %q", path)
		}
	}
}

// Sprint: #249; Story: #715; Story-ID: 90f96d4f4dae
func TestResolveModuleProgramAcceptsAuthenticatedAssemblyCompanion(t *testing.T) {
	goTool := os.Getenv("BASHPP_TESTDIR_GO")
	if goTool == "" {
		var err error
		goTool, err = exec.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test/asmrun\n\ngo 1.20\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	main := []byte("package main\n\nfunc f() int64\n\nfunc main() { if f() != 42 { panic(\"f\") } }\n")
	if err := os.WriteFile(filepath.Join(dir, "main.go"), main, 0o600); err != nil {
		t.Fatal(err)
	}
	writeAsm := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeAsm("f_arm64.s", "TEXT ·f(SB),4,$0-8\n\tMOVD\t$42, R0\n\tMOVD\tR0, ret+0(FP)\n\tRET\n")
	writeAsm("f_amd64.s", "TEXT ·f(SB),4,$0-8\n\tMOVQ\t$42, ret+0(FP)\n\tRET\n")
	files, mapArgs, record, err := resolveModuleProgram(goTool, dir, append(os.Environ(), "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0"))
	if err != nil {
		t.Fatal(err)
	}
	companion := filepath.Join(dir, "f_"+runtime.GOARCH+".s")
	if strings.Join(files, ",") != filepath.Join(dir, "main.go") || len(mapArgs) != 0 {
		t.Fatalf("files/map=%v/%v", files, mapArgs)
	}
	gotCompanions, _ := record["companions"].([]string)
	gotProof, _ := record["companion_proof"].([]companionFile)
	if !reflect.DeepEqual(gotCompanions, []string{companion}) || !validCompanionRecords(gotProof, []string{companion}) {
		t.Fatalf("companion evidence = %#v %#v, want %s", gotCompanions, gotProof, companion)
	}
	moduleDir := t.TempDir()
	_, _, _, mainDir, err := nativeModuleCommands(moduleDir, "bashy", "/runtime", record, files)
	if err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(mainDir, filepath.Base(companion))
	if _, err := os.Stat(copied); err != nil {
		t.Fatalf("compiled unit lacks copied companion %s: %v", copied, err)
	}
	if _, err := os.Stat(filepath.Join(mainDir, "bashpp_asmdecls.go")); !os.IsNotExist(err) {
		t.Fatalf("compiled unit generated duplicate bodyless declaration stub: %v", err)
	}
}

// Sprint: #249; Story: #715; Story-ID: 90f96d4f4dae
func TestResolveModuleProgramAcceptsAuthenticatedCgoPackageWhenEnabled(t *testing.T) {
	goTool := os.Getenv("BASHPP_TESTDIR_GO")
	if goTool == "" {
		var err error
		goTool, err = exec.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
	}
	dir := cgoModuleFixture(t)
	files, mapArgs, record, err := resolveModuleProgram(goTool, dir, append(os.Environ(), "CGO_ENABLED=1"))
	if err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(dir, "main.go")
	cgoFile := filepath.Join(dir, "bad", "bad.go")
	if !reflect.DeepEqual(files, []string{main}) {
		t.Fatalf("main files = %v, want %s", files, main)
	}
	wantMap := "example.test/cgorun/bad=" + cgoFile
	if strings.Join(mapArgs, " ") != "--go-import-path example.test/cgorun --go-package "+wantMap {
		t.Fatalf("map args = %v", mapArgs)
	}
	packages, _ := record["packages"].([]map[string]any)
	if len(packages) != 1 {
		t.Fatalf("packages = %#v", packages)
	}
	gotFiles, _ := packages[0]["files"].([]string)
	gotCompanions, _ := packages[0]["companions"].([]string)
	gotProof, _ := packages[0]["companion_proof"].([]companionFile)
	if !reflect.DeepEqual(gotFiles, []string{cgoFile}) || !reflect.DeepEqual(gotCompanions, []string{cgoFile}) || !validCompanionRecords(gotProof, []string{cgoFile}) || gotProof[0].Role != "cgo" {
		t.Fatalf("cgo package evidence files=%#v companions=%#v proof=%#v", gotFiles, gotCompanions, gotProof)
	}
	moduleDir := t.TempDir()
	commands, _, _, _, err := nativeModuleCommands(moduleDir, "bashy", "/runtime", record, files)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(commands[0], " "); !strings.Contains(got, "--go-native-unit --go-import-path example.test/cgorun/bad --go-file "+cgoFile) {
		t.Fatalf("cgo unit command = %v", commands[0])
	}
	if _, err := os.Stat(filepath.Join(moduleDir, "bad", "bad.go")); !os.IsNotExist(err) {
		t.Fatalf("cgo source was copied beside generated unit: %v", err)
	}
}

// Sprint: #249; Story: #715; Story-ID: 90f96d4f4dae
func TestResolveModuleProgramRefusesCgoPackageWhenDisabled(t *testing.T) {
	goTool := os.Getenv("BASHPP_TESTDIR_GO")
	if goTool == "" {
		var err error
		goTool, err = exec.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
	}
	_, _, _, err := resolveModuleProgram(goTool, cgoModuleFixture(t), append(os.Environ(), "CGO_ENABLED=0"))
	if err == nil || !strings.Contains(err.Error(), "CGO_ENABLED=0") || !strings.Contains(err.Error(), "bad.go") {
		t.Fatalf("resolveModuleProgram error = %v, want cgo disabled refusal", err)
	}
}

// Sprint: #249; Story: #715; Story-ID: 90f96d4f4dae
func TestCgoNativeUnitTranspileBlockedUntilFrontendFakeImportC(t *testing.T) {
	t.Skip("blocker: sh D2 exposes cgo support through gosource.Options.FakeImportC, but bashsharp --source=go --go-native-unit has no authenticated flag/path to enable it; leaf b5-r1 fails before output with `could not import C (package \"C\" not found in go list output)`")
}

func cgoModuleFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "bad"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test/cgorun\n\ngo 1.20\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	main := []byte("package main\n\nimport \"example.test/cgorun/bad\"\n\nfunc main() { bad.Alloc() }\n")
	if err := os.WriteFile(filepath.Join(dir, "main.go"), main, 0o600); err != nil {
		t.Fatal(err)
	}
	bad := []byte("package bad\n\n// #include <stdlib.h>\nimport \"C\"\n\nfunc Alloc() { C.malloc(1) }\n")
	if err := os.WriteFile(filepath.Join(dir, "bad", "bad.go"), bad, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func validCompanionRecords(got []companionFile, paths []string) bool {
	if len(got) != len(paths) {
		return false
	}
	for i, proof := range got {
		if proof.Path != paths[i] || len(proof.SHA256) != 64 {
			return false
		}
	}
	return true
}

// Sprint: #249; Story: #715; Story-ID: 90f96d4f4dae
func TestResolveModuleProgramStillRefusesCgo(t *testing.T) {
	goTool := os.Getenv("BASHPP_TESTDIR_GO")
	if goTool == "" {
		var err error
		goTool, err = exec.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test/cgo\n\ngo 1.20\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport \"C\"\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := resolveModuleProgram(goTool, dir, append(os.Environ(), "CGO_ENABLED=1"))
	if err == nil || !strings.Contains(err.Error(), "non-Go inputs [main.go]") {
		t.Fatalf("resolveModuleProgram cgo error = %v, want existing non-Go refusal", err)
	}
}

func TestS243DirectoryImportClosure(t *testing.T) {
	dir := t.TempDir()
	write := func(name, source string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	good := packageGroup{path: "test/good", files: []string{write("good.go", "package good; const N = 1")}}
	bad := packageGroup{path: "test/bad", files: []string{write("bad.go", "package bad; var N int = `bad`")}}
	via := packageGroup{path: "test/via", files: []string{write("via.go", "package via; import _ `./bad`")}}
	available := []packageGroup{good, bad, via}
	for _, tc := range []struct{ name, source, want string }{
		{"unused rejected package", "package main; import _ `./good`", "test/good"},
		{"direct rejected dependency", "package main; import _ `./bad`", "test/bad"},
		{"absolute rejected dependency", "package main; import _ `test/bad`", "test/bad"},
		{"transitive rejected dependency", "package main; import _ `./via`", "test/bad,test/via"},
		{"broken import header", "package main; import", "test/good,test/bad,test/via"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := write("main.go", tc.source)
			got := directoryImportClosure("test", []string{root}, available)
			var paths []string
			for _, p := range got {
				paths = append(paths, p.path)
			}
			if strings.Join(paths, ",") != tc.want {
				t.Fatalf("selected %v, want %s", paths, tc.want)
			}
			if tool := os.Getenv("BASHPP_TESTDIR_TOOL"); tool != "" {
				args := []string{"--bashpp", "--source=go", "--check", "--go-import-base", "test", "--go-import-path", "main"}
				for _, p := range got {
					args = append(args, "--go-package", p.path+"="+strings.Join(p.files, ","))
				}
				args = append(args, "--go-file", root)
				out, err := exec.Command(tool, args...).CombinedOutput()
				wantFailure := tc.name != "unused rejected package"
				if (err != nil) != wantFailure {
					t.Fatalf("check: %v output=%s; want failure=%v", err, out, wantFailure)
				}
			}
		})
	}
}

func TestS243InterpretedRecipeGOFLAGS(t *testing.T) {
	for _, test := range []struct {
		name  string
		flags []string
		want  bool
	}{
		{"default", nil, false},
		{"exact supported policy", []string{"-gcflags=-d=converthash=qy"}, true},
		{"suffix", []string{"-gcflags=-d=converthash=qySuffix"}, false},
		{"package pattern", []string{"-gcflags=example.com/unrelated=-d=converthash=qy"}, false},
		{"additional recipe flag", []string{"-gcflags=-d=converthash=qy", "-race"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := interpretedRecipeGOFLAGS(test.flags)
			if ok != test.want {
				t.Fatalf("interpretedRecipeGOFLAGS(%q) = %q, %v", test.flags, got, ok)
			}
			if ok && got != "-gcflags=-d=converthash=qy" {
				t.Fatalf("transported GOFLAGS = %q", got)
			}
		})
	}
	env := replaceCommandEnv([]string{"A=1", "GOFLAGS=-gcflags=-d=converthash=xx", "B=2"}, "GOFLAGS", "-gcflags=-d=converthash=qy")
	if got := strings.Join(env, "|"); got != "A=1|B=2|GOFLAGS=-gcflags=-d=converthash=qy" {
		t.Fatalf("replaced environment = %q", got)
	}
}
