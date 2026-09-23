//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	lockfileExclusiveLock = 0x00000002
)

var (
	kernel32Tour = syscall.NewLazyDLL("kernel32.dll")
	lockFileEx   = kernel32Tour.NewProc("LockFileEx")
	unlockFileEx = kernel32Tour.NewProc("UnlockFileEx")
)

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | windowsCreateSuspended}
}
func candidatePayloadPath(launcher string) string { return launcher }
func goExecutable(goroot string) string           { return filepath.Join(goroot, "bin", "go.exe") }
func isExecutable(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}
func linkSDK(link, target string) error {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return exec.Command(filepath.Join(root, "System32", "cmd.exe"), "/c", "mklink", "/J", link, target).Run()
}

func killProcessTree(pid int) error      { return windowsTerminateTree(pid) }
func terminateProcessTree(pid int) error { return windowsTerminateTree(pid) }
func processGone(err error) bool         { return err == nil || errors.Is(err, os.ErrProcessDone) }
func terminateSelf() {
	if p, err := os.FindProcess(os.Getpid()); err == nil {
		_ = p.Kill()
	}
}

func processStatus(state *os.ProcessState) (any, any) {
	if state == nil {
		return nil, nil
	}
	return int64(state.ExitCode()), nil
}

func processExit(state *os.ProcessState) any {
	exit, _ := processStatus(state)
	return exit
}

func lockFile(f *os.File) (func(), error) {
	var overlapped syscall.Overlapped
	r, _, callErr := lockFileEx.Call(f.Fd(), lockfileExclusiveLock, 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if r == 0 {
		return nil, callErr
	}
	return func() { _, _, _ = unlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&overlapped))) }, nil
}
