package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/polyglot"
)

func main() {
	if len(os.Args) != 2 {
		fail("usage: nanochat-stale-handle NANOCHAT_ROOT")
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		fail("nanochat root: %v", err)
	}
	python := os.Getenv("BASHPP_PYTHON")
	plan, err := polyglot.PlanImport(polyglot.ImportRequest{
		Source: filepath.Join(root, "sprint183-fixture.bpp"), Language: "python",
		Module: "nanochat.execution", Alias: "nano",
		Environ: []string{"BASHPP_PYTHON=" + python, "PATH=" + os.Getenv("PATH"), "PYTHONPATH=" + root},
	})
	if err != nil {
		fail("plan import: %v", err)
	}
	module := polyglot.StartImport(plan)
	defer module.Close()
	result, err := module.CallKeywords(context.Background(), "execute_code", []any{"print(6 * 7)"}, map[string]any{"timeout": 5.0})
	if err != nil {
		fail("execute_code: %v", err)
	}
	handle, ok := result.Value.(*polyglot.Handle)
	if !ok {
		fail("execute_code returned %T, want Python handle", result.Value)
	}
	success, err := module.GetAttr(context.Background(), handle, "success")
	if err != nil || success.Value != true {
		fail("success = %#v, err=%v", success.Value, err)
	}
	stdout, err := module.GetAttr(context.Background(), handle, "stdout")
	if err != nil || strings.TrimSpace(fmt.Sprint(stdout.Value)) != "42" {
		fail("stdout = %#v, err=%v", stdout.Value, err)
	}
	if err := module.Close(); err != nil {
		fail("restart worker: %v", err)
	}
	if _, err := module.GetAttr(context.Background(), handle, "success"); err == nil || !strings.Contains(err.Error(), "stale or foreign Python handle") {
		fail("stale handle error = %v", err)
	}
	fmt.Println("nanochat stale handle: PASS")
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "nanochat-stale-handle: "+format+"\n", args...)
	os.Exit(1)
}
