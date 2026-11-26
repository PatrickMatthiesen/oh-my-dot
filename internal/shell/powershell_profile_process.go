package shell

import (
	"context"
	"os/exec"
)

// runShellProfileCommand expects an unstarted command with nil streams or
// direct file handles so descendants can inherit stdio without delaying Wait.
func runShellProfileCommand(ctx context.Context, cmd *exec.Cmd) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	terminate, release, err := startShellProfileCommand(cmd)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	if release != nil {
		defer release()
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		// Favor an already completed run so surviving background jobs
		// keep running after their profile has finished.
		select {
		case err := <-done:
			return err
		default:
		}
		terminate()
		<-done
		return ctx.Err()
	}
}
