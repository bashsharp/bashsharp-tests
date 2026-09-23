//go:build windows

// Windows runner equivalent for launch.go. The gate passes an anonymous-pipe
// handle instead of a FIFO. The handle is explicitly inherited by the program
// under test, while this launcher waits and returns the program's exit status.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	argv := os.Args[1:]
	if len(argv) < 4 || argv[2] != "--" || !strings.HasPrefix(argv[0], "handle:") {
		fmt.Fprintln(os.Stderr, "usage: gbe-launch handle:HANDLE PIDFILE -- PROGRAM [ARG...]")
		os.Exit(2)
	}
	raw, err := strconv.ParseUint(strings.TrimPrefix(argv[0], "handle:"), 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gbe-launch: invalid liveness handle: %v\n", err)
		os.Exit(2)
	}
	handle := syscall.Handle(uintptr(raw))
	liveness := os.NewFile(uintptr(handle), "gbe-liveness")
	if liveness == nil {
		fmt.Fprintln(os.Stderr, "gbe-launch: invalid liveness handle")
		os.Exit(2)
	}

	command := argv[3:]
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags:              syscall.CREATE_NEW_PROCESS_GROUP,
		AdditionalInheritedHandles: []syscall.Handle{handle},
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "gbe-launch: cannot execute %s: %v\n", command[0], err)
		os.Exit(127)
	}
	_ = liveness.Close()
	pidfile := argv[1]
	if err := os.WriteFile(pidfile+".tmp", []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o600); err != nil {
		_ = cmd.Process.Kill()
		fmt.Fprintf(os.Stderr, "gbe-launch: cannot publish pid: %v\n", err)
		os.Exit(2)
	}
	if err := os.Rename(pidfile+".tmp", pidfile); err != nil {
		_ = cmd.Process.Kill()
		fmt.Fprintf(os.Stderr, "gbe-launch: cannot publish pid: %v\n", err)
		os.Exit(2)
	}
	if err := cmd.Wait(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "gbe-launch: wait failed: %v\n", err)
		os.Exit(127)
	}
}
