//go:build !windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func configureProcess(cmd *exec.Cmd)              { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func registerProcessTree(*exec.Cmd) error         { return nil }
func releaseProcessTree(int)                      {}
func candidatePayloadPath(launcher string) string { return launcher + ".real" }
func goExecutable(goroot string) string           { return filepath.Join(goroot, "bin", "go") }
func linkSDK(link, target string) error           { return os.Symlink(target, link) }

func processTreeAlive(pgid int) bool {
	err := syscall.Kill(-pgid, 0)
	return err == nil || !errors.Is(err, syscall.ESRCH)
}

func killProcessTree(pgid int) error      { return syscall.Kill(-pgid, syscall.SIGKILL) }
func terminateProcessTree(pgid int) error { return syscall.Kill(-pgid, syscall.SIGTERM) }
func processGone(err error) bool          { return errors.Is(err, syscall.ESRCH) }
func terminateSelf()                      { _ = syscall.Kill(os.Getpid(), syscall.SIGTERM) }

func processStatus(state *os.ProcessState) (any, any) {
	if ws, ok := state.Sys().(syscall.WaitStatus); ok {
		if ws.Exited() {
			return int64(ws.ExitStatus()), nil
		}
		if ws.Signaled() {
			return nil, int64(ws.Signal())
		}
	}
	return nil, nil
}

func processExit(state *os.ProcessState) any {
	exit, signal := processStatus(state)
	if signal != nil {
		return int64(128) + signal.(int64)
	}
	return exit
}

func lockFile(f *os.File) (func(), error) {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return nil, err
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }, nil
}
