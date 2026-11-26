package shell

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func startShellProfileCommand(cmd *exec.Cmd) (terminate, release func(), err error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create shell job: %w", err)
	}
	releaseJob := func() { _ = windows.CloseHandle(job) }

	// Assign the process before it can launch children. Closing the job
	// handle on ordinary exit leaves any background processes running.
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_SUSPENDED}
	if err := cmd.Start(); err != nil {
		releaseJob()
		return nil, nil, fmt.Errorf("start suspended shell process: %w", err)
	}
	if err := assignAndResumeShellProfileProcess(job, uint32(cmd.Process.Pid)); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		releaseJob()
		return nil, nil, err
	}
	return func() {
		if err := windows.TerminateJobObject(job, 1); err != nil {
			_ = cmd.Process.Kill()
		}
	}, releaseJob, nil
}

func assignAndResumeShellProfileProcess(job windows.Handle, pid uint32) error {
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return fmt.Errorf("open suspended shell process: %w", err)
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		return fmt.Errorf("assign shell process to job: %w", err)
	}

	// os/exec closes the primary thread handle. A suspended process has
	// only its initial thread, which we can open by its thread ID.
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return fmt.Errorf("take Windows thread snapshot: %w", err)
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	for err = windows.Thread32First(snapshot, &entry); err == nil; err = windows.Thread32Next(snapshot, &entry) {
		if entry.OwnerProcessID != pid {
			continue
		}
		thread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
		if err != nil {
			return fmt.Errorf("open suspended shell thread: %w", err)
		}
		defer windows.CloseHandle(thread)
		if _, err := windows.ResumeThread(thread); err != nil {
			return fmt.Errorf("resume shell thread: %w", err)
		}
		return nil
	}
	return fmt.Errorf("find initial shell thread: %w", err)
}
