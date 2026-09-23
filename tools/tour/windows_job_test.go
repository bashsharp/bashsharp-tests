//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWindowsExecutableRecognition(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if !isExecutable(exe) {
		t.Fatalf("native executable rejected: %s", exe)
	}
	if isExecutable(t.TempDir()) || isExecutable(filepath.Join(t.TempDir(), "missing.exe")) {
		t.Fatal("directory or missing executable accepted")
	}
}

func TestWindowsLineageAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lineage.jsonl")
	for i := int64(1); i <= 2; i++ {
		if err := appendLineage(path, map[string]any{"sequence": i}); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{\"sequence\":1}\n{\"sequence\":2}\n" {
		t.Fatalf("lineage append lost a record: %q", data)
	}
}

// A descendant must remain visible after its launcher has exited. The former
// leader-only check reported this tree empty and silently skipped cleanup.
func TestWindowsJobSurvivingDescendant(t *testing.T) {
	switch os.Getenv("WINDOWS_JOB_HELPER") {
	case "child":
		time.Sleep(30 * time.Second)
		return
	case "leader":
		child := exec.Command(os.Args[0], "-test.run=^TestWindowsJobSurvivingDescendant$")
		child.Env = append(os.Environ(), "WINDOWS_JOB_HELPER=child")
		if err := child.Start(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(os.Getenv("WINDOWS_JOB_PIDFILE"), []byte(strconv.Itoa(child.Process.Pid)), 0600); err != nil {
			t.Fatal(err)
		}
		_ = child.Process.Release()
		return
	}
	pidfile := filepath.Join(t.TempDir(), "child.pid")
	leader := exec.Command(os.Args[0], "-test.run=^TestWindowsJobSurvivingDescendant$")
	leader.Env = append(os.Environ(), "WINDOWS_JOB_HELPER=leader", "WINDOWS_JOB_PIDFILE="+pidfile)
	configureProcess(leader)
	if err := leader.Start(); err != nil {
		t.Fatal(err)
	}
	pid := leader.Process.Pid
	defer releaseProcessTree(pid)
	if err := registerProcessTree(leader); err != nil {
		_ = leader.Process.Kill()
		_ = leader.Wait()
		t.Fatal(err)
	}
	if err := leader.Wait(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(pidfile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := strconv.Atoi(strings.TrimSpace(string(raw))); err != nil {
		t.Fatal(err)
	}
	if !processTreeAlive(pid) {
		t.Fatal("surviving descendant was not counted after leader exit")
	}
	if err := windowsTerminateTree(pid); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for processTreeAlive(pid) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if processTreeAlive(pid) {
		t.Fatal("job still contains a process after termination")
	}
}
