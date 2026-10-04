package main

import (
	"bytes"
	"context"
	goparser "go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestInterpretedCommandUsesPlainBshAndPreservesArguments(t *testing.T) {
	got := interpretedCommand("/candidate/bashy", "/work/main.bsh", []string{"-test.v"})
	if !reflect.DeepEqual(got, []string{"/candidate/bashy", "/work/main.bsh", "-test.v"}) {
		t.Fatal(got)
	}
}

func TestInterpretedTestEntryPreservesAssertions(t *testing.T) {
	source := []byte("package main\nimport \"testing\"\nfunc TestFail(t *testing.T) { t.Fatal(\"sentinel\") }\n")
	driver, err := testDriver(source, "main_test.go")
	if err != nil {
		t.Fatal(err)
	}
	entry, err := interpretedTestEntry(source, driver)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(entry, string(source[len("package main"):])) || strings.Count(entry, `"testing"`) != 1 || !strings.Contains(entry, "F: TestFail") {
		t.Fatal(entry)
	}
	if _, err := goparser.ParseFile(token.NewFileSet(), "entry.bsh", entry, 0); err != nil {
		t.Fatal(err)
	}
}

// Opt-in, bounded execution of three pinned examples through the same staging,
// test-driver and command builders as the gate. No full-corpus work is done.
func TestInterpretedBshSmoke(t *testing.T) {
	bashy := os.Getenv("S378_BASHY")
	if bashy == "" {
		t.Skip("set S378_BASHY for focused runtime verification")
	}
	withRoot(t)
	for _, rel := range []string{"examples/hello-world/hello-world.go", "examples/command-line-arguments/command-line-arguments.go", "examples/testing-and-benchmarking/main_test.go"} {
		t.Run(rel, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(ROOT, rel))
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			name := strings.TrimSuffix(filepath.Base(rel), ".go") + ".bsh"
			stageSources(dir, []string{rel, "", "", "", "", "none"}, name)
			entry := filepath.Join(dir, name)
			copied, _ := os.ReadFile(entry)
			if !bytes.Equal(source, copied) {
				t.Fatal("staged source bytes changed")
			}
			args := []string{}
			if strings.HasSuffix(rel, "main_test.go") {
				driver, err := testDriver(source, rel)
				if err != nil {
					t.Fatal(err)
				}
				text, err := interpretedTestEntry(source, driver)
				if err != nil {
					t.Fatal(err)
				}
				entry = filepath.Join(dir, "gbe_test_driver.bsh")
				if err := os.WriteFile(entry, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
				args = []string{"-test.v"}
			} else if strings.Contains(rel, "command-line-arguments") {
				args = []string{"foo", "bar", "baz"}
			}
			os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Focused smoke\n"), 0600)
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			argv := interpretedCommand(bashy, entry, args)
			cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "BASHY_OTEL_SPOOL="+filepath.Join(dir, "otel.jsonl"))
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%v: %v\n%s", argv, err, out)
			}
			switch {
			case strings.Contains(rel, "hello-world"):
				if string(out) != "hello world\n" {
					t.Fatalf("unexpected output: %s", out)
				}
			case strings.Contains(rel, "command-line-arguments"):
				if !bytes.Contains(out, []byte("[foo bar baz]\nbaz\n")) {
					t.Fatalf("lost program arguments: %s", out)
				}
			default:
				if !bytes.Contains(out, []byte("--- PASS: TestIntMinBasic")) || !bytes.Contains(out, []byte("--- PASS: TestIntMinTableDriven")) || !bytes.HasSuffix(out, []byte("PASS\n")) {
					t.Fatalf("test bodies did not pass: %s", out)
				}
			}
			t.Logf("plain .bsh invocation passed: %s", rel)
		})
	}
}
