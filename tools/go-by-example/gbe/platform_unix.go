//go:build !windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func executableFile(path string) bool             { return syscall.Access(path, 1) == nil }
func candidatePayloadPath(launcher string) string { return launcher + ".real" }
func goExecutable(goroot string) string           { return filepath.Join(goroot, "bin", "go") }
func linkSDK(link, target string) error           { return os.Symlink(target, link) }

func configureProcess(cmd *exec.Cmd, _ map[string]string) func() error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return func() error { return nil }
}
func releaseProcessTree(int) {}

func configureDetached(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
func processTreeAlive(pgid int) bool {
	err := syscall.Kill(-pgid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
func signalProcessTree(name string, pgid int) error {
	sig := syscall.SIGKILL
	if name == "TERM" {
		sig = syscall.SIGTERM
	} else if name == "INT" {
		sig = syscall.SIGINT
	}
	return syscall.Kill(-pgid, sig)
}
func processGone(err error) bool       { return errors.Is(err, syscall.ESRCH) }
func processPermission(err error) bool { return errors.Is(err, syscall.EPERM) }

func processStatus(state *os.ProcessState) (any, any) {
	if ws, ok := state.Sys().(syscall.WaitStatus); ok {
		if ws.Signaled() {
			return nil, Int(int64(ws.Signal()))
		}
		return Int(int64(ws.ExitStatus())), nil
	}
	return nil, nil
}

func lockFile(f *os.File) (func(), error) {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return nil, err
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }, nil
}

func platformStatIdentity(path string) (int64, int64, error) {
	st, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, errors.New("no stat identity for " + path)
	}
	return int64(sys.Dev), int64(sys.Ino), nil
}

func prepareLiveness(adapterDir string, _ map[string]string) (*os.File, string, func(), error) {
	fifo := filepath.Join(adapterDir, "liveness.fifo")
	_ = os.Remove(fifo)
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		return nil, "", nil, err
	}
	reader, err := os.OpenFile(fifo, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		_ = os.Remove(fifo)
		return nil, "", nil, err
	}
	return reader, fifo, func() { _ = reader.Close(); _ = os.Remove(fifo) }, nil
}
