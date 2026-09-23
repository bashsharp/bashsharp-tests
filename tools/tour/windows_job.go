//go:build windows

package main

// A new process group is not a Windows process tree. Keep an owned Job Object
// for each capture, and do not let the child run until it belongs to that job.
// The job retains descendants after the leader exits, including detached ones.

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

const (
	windowsCreateSuspended = 0x00000004
	processSetQuota        = 0x0100
	processTerminate       = 0x0001
	threadSuspendResume    = 0x0002
)

type windowsJobAccounting struct {
	TotalUserTime, TotalKernelTime                                                 int64
	ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime                             int64
	TotalPageFaultCount, TotalProcesses, ActiveProcesses, TotalTerminatedProcesses uint32
}

type windowsThreadEntry struct {
	Size, Usage, ThreadID, OwnerProcessID uint32
	BasePri, DeltaPri                     int32
	Flags                                 uint32
}

var (
	jobDLL             = syscall.NewLazyDLL("kernel32.dll")
	createJobObject    = jobDLL.NewProc("CreateJobObjectW")
	assignProcessToJob = jobDLL.NewProc("AssignProcessToJobObject")
	terminateJob       = jobDLL.NewProc("TerminateJobObject")
	queryJob           = jobDLL.NewProc("QueryInformationJobObject")
	thread32First      = jobDLL.NewProc("Thread32First")
	thread32Next       = jobDLL.NewProc("Thread32Next")
	openThread         = jobDLL.NewProc("OpenThread")
	resumeThread       = jobDLL.NewProc("ResumeThread")
	windowsJobs        sync.Map // pid -> syscall.Handle; held through the final leak check
)

func windowsCallError(name string, e error) error {
	if e == nil || e == syscall.Errno(0) {
		return fmt.Errorf("%s failed", name)
	}
	return fmt.Errorf("%s: %w", name, e)
}

func registerProcessTree(cmd *exec.Cmd) error {
	pid := cmd.Process.Pid
	job, _, e := createJobObject.Call(0, 0)
	if job == 0 {
		return windowsCallError("CreateJobObjectW", e)
	}
	closeJob := true
	defer func() {
		if closeJob {
			_ = syscall.CloseHandle(syscall.Handle(job))
		}
	}()
	process, err := syscall.OpenProcess(processSetQuota|processTerminate, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("open suspended process: %w", err)
	}
	defer syscall.CloseHandle(process)
	if ok, _, e := assignProcessToJob.Call(job, uintptr(process)); ok == 0 {
		return windowsCallError("AssignProcessToJobObject", e)
	}
	if err := resumeProcess(pid); err != nil {
		_, _, _ = terminateJob.Call(job, 1)
		return err
	}
	windowsJobs.Store(pid, syscall.Handle(job))
	closeJob = false
	return nil
}

func resumeProcess(pid int) error {
	snapshot, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return fmt.Errorf("snapshot suspended threads: %w", err)
	}
	defer syscall.CloseHandle(snapshot)
	entry := windowsThreadEntry{Size: uint32(unsafe.Sizeof(windowsThreadEntry{}))}
	for ok, _, _ := thread32First.Call(uintptr(snapshot), uintptr(unsafe.Pointer(&entry))); ok != 0; ok, _, _ = thread32Next.Call(uintptr(snapshot), uintptr(unsafe.Pointer(&entry))) {
		if entry.OwnerProcessID != uint32(pid) {
			continue
		}
		thread, _, e := openThread.Call(threadSuspendResume, 0, uintptr(entry.ThreadID))
		if thread == 0 {
			return windowsCallError("OpenThread", e)
		}
		count, _, e := resumeThread.Call(thread)
		_ = syscall.CloseHandle(syscall.Handle(thread))
		if count == ^uintptr(0) {
			return windowsCallError("ResumeThread", e)
		}
		return nil
	}
	return fmt.Errorf("suspended process %d has no main thread", pid)
}

func processTreeAlive(pid int) bool {
	stored, ok := windowsJobs.Load(pid)
	if !ok {
		return true
	} // missing ownership cannot certify an empty tree
	job := stored.(syscall.Handle)
	var accounting windowsJobAccounting
	const jobObjectBasicAccountingInformation = 1
	r, _, _ := queryJob.Call(uintptr(job), jobObjectBasicAccountingInformation,
		uintptr(unsafe.Pointer(&accounting)), unsafe.Sizeof(accounting), 0)
	return r == 0 || accounting.ActiveProcesses != 0 // query failure fails closed
}

func windowsTerminateTree(pid int) error {
	stored, ok := windowsJobs.Load(pid)
	if !ok {
		return fmt.Errorf("process %d has no owned job", pid)
	}
	r, _, e := terminateJob.Call(uintptr(stored.(syscall.Handle)), 1)
	if r == 0 {
		return windowsCallError("TerminateJobObject", e)
	}
	return nil
}

func releaseProcessTree(pid int) {
	if stored, ok := windowsJobs.LoadAndDelete(pid); ok {
		_ = syscall.CloseHandle(stored.(syscall.Handle))
	}
}
