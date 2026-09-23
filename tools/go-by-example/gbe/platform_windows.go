//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"unsafe"
)

const (
	lockfileExclusiveLock = 0x00000002
)

var (
	kernel32GBE       = syscall.NewLazyDLL("kernel32.dll")
	lockFileEx        = kernel32GBE.NewProc("LockFileEx")
	unlockFileEx      = kernel32GBE.NewProc("UnlockFileEx")
	generateCtrlEvent = kernel32GBE.NewProc("GenerateConsoleCtrlEvent")
	livenessWriters   sync.Map // raw handle -> sole owning *os.File
)

func executableFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}
func candidatePayloadPath(launcher string) string { return launcher }
func goExecutable(goroot string) string           { return filepath.Join(goroot, "bin", "go.exe") }
func linkSDK(link, target string) error {
	root := os.Getenv("SystemRoot")
	if root == "" {
		root = `C:\Windows`
	}
	return exec.Command(filepath.Join(root, "System32", "cmd.exe"), "/c", "mklink", "/J", link, target).Run()
}

func configureProcess(cmd *exec.Cmd, env map[string]string) func() error {
	attr := &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | windowsCreateSuspended}
	var rawHandle uintptr
	if raw := env["GBE_LIVENESS_HANDLE"]; raw != "" {
		value, err := strconv.ParseUint(raw, 10, 64)
		if err == nil {
			rawHandle = uintptr(value)
			attr.AdditionalInheritedHandles = []syscall.Handle{syscall.Handle(rawHandle)}
		}
	}
	cmd.SysProcAttr = attr
	return func() error {
		if rawHandle != 0 {
			if owner, ok := livenessWriters.LoadAndDelete(rawHandle); ok {
				_ = owner.(*os.File).Close()
			}
		}
		if cmd.Process == nil {
			return nil
		}
		return registerProcessTree(cmd)
	}
}

func configureDetached(cmd *exec.Cmd) {
	attr := &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000008}
	if raw := os.Getenv("GBE_LIVENESS_HANDLE"); raw != "" {
		if value, err := strconv.ParseUint(raw, 10, 64); err == nil {
			attr.AdditionalInheritedHandles = []syscall.Handle{syscall.Handle(uintptr(value))}
		}
	}
	cmd.SysProcAttr = attr
}

func signalProcessTree(name string, pid int) error {
	if name == "INT" || name == "TERM" {
		const ctrlBreakEvent = 1
		r, _, err := generateCtrlEvent.Call(ctrlBreakEvent, uintptr(uint32(pid)))
		if r != 0 {
			return nil
		}
		return err
	}
	return windowsTerminateTree(pid)
}
func processGone(err error) bool       { return err == nil || errors.Is(err, os.ErrProcessDone) }
func processPermission(err error) bool { return errors.Is(err, os.ErrPermission) }

func processStatus(state *os.ProcessState) (any, any) {
	if state == nil {
		return nil, nil
	}
	return Int(int64(state.ExitCode())), nil
}

func lockFile(f *os.File) (func(), error) {
	var overlapped syscall.Overlapped
	r, _, callErr := lockFileEx.Call(f.Fd(), lockfileExclusiveLock, 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if r == 0 {
		return nil, callErr
	}
	return func() { _, _, _ = unlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&overlapped))) }, nil
}

func platformStatIdentity(path string) (int64, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &info); err != nil {
		return 0, 0, err
	}
	ino := uint64(info.FileIndexHigh)<<32 | uint64(info.FileIndexLow)
	return int64(info.VolumeSerialNumber), int64(ino), nil
}

func prepareLiveness(_ string, env map[string]string) (*os.File, string, func(), error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, "", nil, err
	}
	handle := strconv.FormatUint(uint64(writer.Fd()), 10)
	env["GBE_LIVENESS_HANDLE"] = handle
	livenessWriters.Store(writer.Fd(), writer)
	return reader, "handle:" + handle, func() {
		_ = reader.Close()
		if owner, ok := livenessWriters.LoadAndDelete(writer.Fd()); ok {
			_ = owner.(*os.File).Close()
		}
	}, nil
}
