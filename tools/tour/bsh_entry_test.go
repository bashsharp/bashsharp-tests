package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestBshNativeCopyAndMutationDetection(t *testing.T) {
	dir, corpus := t.TempDir(), t.TempDir()
	data := []byte("package main\nfunc main() {}\n")
	item := Item{Path: "main.bsh", SHA256: sha256hex(data)}
	for _, p := range []string{filepath.Join(dir, item.Path), nativeSourcePath(filepath.Join(dir, item.Path)), filepath.Join(corpus, item.Path)} {
		if err := os.WriteFile(p, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := verifySources(dir, corpus, []Item{item}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(nativeSourcePath(filepath.Join(dir, item.Path)), []byte("tampered"), 0600)
	if err := verifySources(dir, corpus, []Item{item}); err == nil {
		t.Fatal("native oracle copy mutation escaped verification")
	}
}

func TestMaterializeCreatesByteExactNativeCopy(t *testing.T) {
	corpus, dir := t.TempDir(), t.TempDir()
	data, license := []byte("package main\nfunc main() {}\n"), []byte("license\n")
	os.WriteFile(filepath.Join(corpus, "main.bsh"), data, 0600)
	os.WriteFile(filepath.Join(corpus, "LICENSE"), license, 0600)
	item := Item{Path: "main.bsh", Bytes: int64(len(data)), SHA256: sha256hex(data)}
	spec := materializeSpec{corpusRoot: corpus, corpus: []string{"tour", "tour/LICENSE", "8", sha256hex(license)}, items: []Item{item}, goVersion: "go1.27.0", helper: []string{"example.test/helper", "v1.0.0", "", "h1:mod", "h1:zip"}, runtimeDep: map[string]any{"module": "mvdan.cc/sh/v3", "require_version": "v3.0.0", "dir": dir}}
	if err := materialize(dir, spec); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.bsh", "main.go"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("%s: %q %v", name, got, err)
		}
	}
	if _, err := os.Stat(filepath.Join(corpus, "main.go")); !os.IsNotExist(err) {
		t.Fatal("native copy escaped the scratch module")
	}
}

// Explicitly opt in: two programs, the real contract, no full gate run.
func TestTourBshSmoke(t *testing.T) {
	bashy, goBin := os.Getenv("S378_BASHY"), os.Getenv("S378_GO")
	if bashy == "" || goBin == "" {
		t.Skip("set S378_BASHY and S378_GO for focused runtime verification")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := loadContract(filepath.Join(root, "docs/tour/executor-contract.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"_content/tour/welcome/hello.bsh", "_content/tour/flowcontrol/for.bsh"} {
		t.Run(rel, func(t *testing.T) {
			dir := t.TempDir()
			source, err := os.ReadFile(filepath.Join(root, "tour", rel))
			if err != nil {
				t.Fatal(err)
			}
			os.WriteFile(filepath.Join(dir, "main.bsh"), source, 0600)
			os.WriteFile(filepath.Join(dir, "main.go"), source, 0600)
			os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Focused smoke\n"), 0600)
			module := "module tour.smoke\n\ngo 1.27\n\nrequire mvdan.cc/sh/v3 v3.0.0\nreplace mvdan.cc/sh/v3 => " + filepath.Join(root, "../sh") + "\n"
			os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600)
			var oracle []byte
			for _, mode := range []string{"baseline", "interpreted", "compiled"} {
				subs := substitutions("main.bsh", bashy, goBin, dir)
				var output []byte
				for _, stage := range contract[modeKey{"applicable_go_program", mode}].Stages {
					argv := mustRenderArgv(stage.ArgvTemplate, subs)
					ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
					cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
					cmd.Dir = dir
					cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOTOOLCHAIN=local", "GOROOT="+filepath.Dir(filepath.Dir(goBin)), "BASHY_OTEL_SPOOL="+filepath.Join(dir, "otel.jsonl"))
					output, err = cmd.CombinedOutput()
					cancel()
					if err != nil {
						t.Fatalf("%s %v: %v\n%s", mode, argv, err, output)
					}
				}
				if mode == "baseline" {
					oracle = output
				} else if !bytes.Equal(output, oracle) {
					t.Fatalf("%s output %q, native oracle %q", mode, output, oracle)
				}
				t.Logf("%s: %q", mode, output)
			}
		})
	}
}
