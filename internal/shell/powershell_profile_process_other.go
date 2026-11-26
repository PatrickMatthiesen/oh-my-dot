//go:build !unix && !windows

package shell

import (
	"fmt"
	"os/exec"
)

func startShellProfileCommand(cmd *exec.Cmd) (terminate, release func(), err error) {
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start shell process: %w", err)
	}
	return func() { _ = cmd.Process.Kill() }, nil, nil
}
