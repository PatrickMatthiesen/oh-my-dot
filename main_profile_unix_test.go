//go:build unix

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestProfileProcessHandlesSignals(t *testing.T) {
	binDir := t.TempDir()
	binary := filepath.Join(binDir, "oh-my-dot")
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build test CLI: %v\n%s", err, output)
	}
	stub := "#!/bin/sh\nprintf '%s' \"$$\" > \"$OMD_TEST_PROFILE_PID\"\nexec sleep 60\n"
	if err := os.WriteFile(filepath.Join(binDir, "pwsh"), []byte(stub), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			root := t.TempDir()
			processTemp := t.TempDir()
			marker := filepath.Join(root, "profiler.pid")
			t.Setenv("OH_MY_DOT_CONFIG", filepath.Join(root, "config.json"))
			t.Setenv("OMD_TEST_PROFILE_PID", marker)
			t.Setenv("TMPDIR", processTemp)

			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, binary, "profile", "--shell", "powershell", "--full", "--json")
			command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- command.Wait() }()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			var childPID int
			for childPID == 0 {
				select {
				case err := <-done:
					t.Fatalf("command exited before starting profiler: %v, %s", err, stderr.String())
				case <-ctx.Done():
					t.Fatal("profiler did not start")
				case <-ticker.C:
					pid, _ := os.ReadFile(marker)
					childPID, _ = strconv.Atoi(string(pid))
				}
			}
			if err := syscall.Kill(-command.Process.Pid, sig); err != nil {
				t.Fatal(err)
			}
			var exitErr *exec.ExitError
			if err := <-done; !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				_ = syscall.Kill(childPID, syscall.SIGKILL)
				t.Fatalf("expected an ordinary failure exit after %s, got %v", sig, err)
			}
			if err := syscall.Kill(childPID, 0); !errors.Is(err, syscall.ESRCH) {
				_ = syscall.Kill(childPID, syscall.SIGKILL)
				t.Fatalf("profiler survived %s: %v", sig, err)
			}
			entries, err := os.ReadDir(processTemp)
			if err != nil || len(entries) != 0 {
				t.Fatalf("profiler left temporary files: %v, %v", entries, err)
			}
			var document struct {
				Results []struct{ Status string }
			}
			if err := json.Unmarshal(stdout.Bytes(), &document); err != nil || len(document.Results) != 1 || document.Results[0].Status != "failed" {
				t.Fatalf("expected one JSON failure report: %s, %s, %v", stdout.String(), stderr.String(), err)
			}
		})
	}
}
