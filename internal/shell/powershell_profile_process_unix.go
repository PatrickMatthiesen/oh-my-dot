//go:build unix

package shell

import (
	"fmt"
	"os/exec"
	"syscall"
)

func startShellProfileCommand(cmd *exec.Cmd) (terminate, release func(), err error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start shell process: %w", err)
	}
	return func() {
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			_ = cmd.Process.Kill()
		}
	}, nil, nil
}
