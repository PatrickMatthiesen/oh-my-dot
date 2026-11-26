//go:build unix || windows

package shell

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestPowerShellProfileCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := runShellProfileCommand(ctx, exec.Command(os.Args[0], "-test.run=^$")); err != nil {
		t.Fatalf("run ordinary command: %v", err)
	}
}

func TestPowerShellProfileCommandCancelsDescendants(t *testing.T) {
	stdout, childStdout, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdout.Close()
	defer childStdout.Close()
	childStdin, stdin, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer childStdin.Close()
	// Closing stdin also lets the fixture exit if the cancellation fails.
	defer stdin.Close()

	cmd := exec.Command(os.Args[0], "-test.run=^TestPowerShellProfileProcessChild$", "--", "parent")
	cmd.Stdin = childStdin
	cmd.Stdout = childStdout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		err := runShellProfileCommand(ctx, cmd)
		childStdout.Close()
		done <- err
	}()
	ready := make(chan error, 1)
	go func() {
		_, err := io.ReadFull(stdout, make([]byte, 1))
		ready <- err
	}()
	select {
	case err := <-ready:
		if err != nil {
			t.Fatalf("read descendant readiness: %v (command error: %v)", err, <-done)
		}
	case <-ctx.Done():
		t.Fatal("descendant did not start")
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled command returned %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("canceled command did not exit")
	}

	// Only the descendant can still hold stdout open at this point.
	closed := make(chan error, 1)
	go func() {
		_, err := io.Copy(io.Discard, stdout)
		closed <- err
	}()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("read descendant stdout: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("descendant still holds stdout open after cancellation")
	}
}

func TestPowerShellProfileProcessChild(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "--" {
		return
	}
	switch os.Args[len(os.Args)-1] {
	case "parent":
		cmd := exec.Command(os.Args[0], "-test.run=^TestPowerShellProfileProcessChild$", "--", "descendant")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	case "descendant":
		if _, err := os.Stdout.Write([]byte{1}); err != nil {
			os.Exit(1)
		}
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
}
