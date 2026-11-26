package shell

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ProfileUnixShellStartup measures native startup in a fresh shell process.
// Bash, zsh, and fish run in interactive non-login mode without a terminal.
// POSIX sh runs in interactive login mode so it reads ~/.profile.
// Startup files can have their usual side effects. The shell runs a command
// after startup and never enters the prompt loop.
func ProfileUnixShellStartup(ctx context.Context, shellName string) (ShellProfileReport, error) {
	executable, err := resolveUnixProfileExecutable(shellName)
	if err != nil {
		return ShellProfileReport{}, err
	}
	return profileUnixShellStartup(ctx, shellName, executable)
}

func profileUnixShellStartup(ctx context.Context, shellName, executable string) (ShellProfileReport, error) {
	ctx, cancel := context.WithTimeout(ctx, shellProfileTimeout)
	defer cancel()
	interrupted := func(err error) error { return fmt.Errorf("%s startup profiling interrupted: %w", shellName, err) }
	if err := ctx.Err(); err != nil {
		return ShellProfileReport{}, interrupted(err)
	}

	tempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		return ShellProfileReport{}, fmt.Errorf("resolve startup marker directory: %w", err)
	}
	dir, err := os.MkdirTemp(tempRoot, "omd-unix-startup-")
	if err != nil {
		return ShellProfileReport{}, fmt.Errorf("create startup marker directory: %w", err)
	}
	defer os.RemoveAll(dir)

	const marker = "omd-startup-complete"
	markerPath := filepath.Join(dir, "complete")
	quotedPath := shellSingleQuote(filepath.ToSlash(markerPath))
	printf := `\builtin \printf`
	mode := "-i"
	switch shellName {
	case "fish":
		printf = "builtin printf"
		quotedPath = fishProfileLiteral(markerPath)
	case "posix":
		mode = "-il"
		printf = `\command \printf`
	}
	// Keep the marker independent of changes to stdout or stderr in a profile.
	// Nil streams connect the child directly to the null device.
	script := printf + " '%s' '" + marker + "' > " + quotedPath
	cmd := exec.Command(executable, mode, "-c", script)

	start := time.Now()
	runErr := runShellProfileCommand(ctx, cmd)
	elapsed := time.Since(start)
	var interruption error
	var exitErr *exec.ExitError
	if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
		interruption = interrupted(runErr)
	} else if runErr != nil && !errors.As(runErr, &exitErr) {
		return ShellProfileReport{}, fmt.Errorf("run %s startup profiler with %s: %w", shellName, executable, runErr)
	}
	data, readErr := os.ReadFile(markerPath)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return ShellProfileReport{}, errors.Join(interruption, fmt.Errorf("read %s startup marker: %w", shellName, readErr))
	}
	if string(data) != marker {
		if interruption != nil {
			return ShellProfileReport{}, interruption
		}
		return ShellProfileReport{}, errors.Join(fmt.Errorf("%s exited before writing the startup completion marker", shellName), runErr)
	}
	info, err := os.Stat(markerPath)
	if err != nil {
		return ShellProfileReport{}, errors.Join(interruption, fmt.Errorf("inspect %s startup marker: %w", shellName, err))
	}
	// Use the write time so exit traps cannot inflate the startup measurement.
	elapsed = max(0, min(elapsed, info.ModTime().Sub(start)))
	return ShellProfileReport{
		Shell:      shellName,
		Executable: executable,
		TotalMS:    float64(elapsed) / float64(time.Millisecond),
		Profiles:   []ProfileTiming{},
		Warnings:   []string{},
	}, interruption
}
