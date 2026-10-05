package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOracleSourceIdentityPreservesBytesAndPositions(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "source with spaces", "src", "oracle")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	program := "package main\nimport \"log\"\nfunc main() { log.SetFlags(log.Lshortfile); log.Print(\"sentinel\"); other() }\n"
	other := "package main\nimport \"log\"\nfunc other() { log.Print(\"unrelated\") }\n"
	for name, contents := range map[string]string{"program.go": program, "other.go": other, "go.mod": "module oracleprobe\n\ngo 1.27\n"} {
		if err := os.WriteFile(filepath.Join(src, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(root, "oracle")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	goBinary := filepath.Join(runtime.GOROOT(), "bin", "go")
	args := oracleBuildCommand(goBinary, src, "program.bsh", bin, false)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = src
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS=")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil || string(out) != "program.bsh:3: sentinel\nother.go:3: unrelated\n" {
		t.Fatalf("run: %v\n%s", err, out)
	}
	data, err := os.ReadFile(filepath.Join(src, "program.go"))
	if err != nil || string(data) != program {
		t.Fatal("oracle source bytes changed")
	}
}

func TestOracleSourceIdentityRejectsRecipeTampering(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src", "oracle")
	output := filepath.Join(filepath.Dir(filepath.Dir(src)), "bin", "oracle")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	for _, testRow := range []bool{false, true} {
		name := "program.bsh"
		if testRow {
			name = "main_test.bsh"
		}
		args := oracleBuildCommand("go", src, name, output, testRow)
		if !validOracleBuildCommand(args, src, name, testRow) {
			t.Fatal("valid command refused")
		}
		for _, mutate := range []func([]string) []string{
			func(a []string) []string {
				for i := range a {
					a[i] = strings.ReplaceAll(a[i], "=>"+name, "=>wrong.bsh")
				}
				return a
			},
			func(a []string) []string {
				for i := range a {
					if strings.HasPrefix(a[i], "-gcflags=") {
						a[i] = "-gcflags=-trimpath=" + src
					}
				}
				return a
			},
			func(a []string) []string {
				for i := range a {
					if strings.HasPrefix(a[i], "-gcflags=") {
						return append(a[:i], a[i+1:]...)
					}
				}
				return a
			},
			func(a []string) []string { return append(a, "-gcflags=-trimpath=other.go=>"+name) },
		} {
			if validOracleBuildCommand(mutate(append([]string(nil), args...)), src, name, testRow) {
				t.Fatal("tampered oracle identity accepted")
			}
		}
	}
}
