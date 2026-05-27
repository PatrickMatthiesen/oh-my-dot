package agentcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyDefaultsToProjectScope(t *testing.T) {
	repoPath := t.TempDir()
	projectPath := t.TempDir()
	sourceDir := createTestSkill(t, t.TempDir(), "frontend-design")
	if _, err := AddSkills(repoPath, sourceDir, "", false, false); err != nil {
		t.Fatalf("AddSkills() error = %v", err)
	}

	cmd := NewCommand(func() string { return "oh-my-dot" }, func() string { return repoPath })
	cmd.SetArgs([]string{"skill", "apply", "frontend-design", "--project", projectPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	targetReadme := filepath.Join(projectPath, ".codex", "skills", "frontend-design", "SKILL.md")
	if _, err := os.Stat(targetReadme); err != nil {
		t.Fatalf("project-scoped skill was not applied: %v", err)
	}
}

func TestUnapplyDefaultsToProjectScope(t *testing.T) {
	projectPath := t.TempDir()
	targetDir := filepath.Join(projectPath, ".codex", "skills", "frontend-design")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte("# frontend-design\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := NewCommand(func() string { return "oh-my-dot" }, func() string { return t.TempDir() })
	cmd.SetArgs([]string{"skills", "unapply", "frontend-design", "--project", projectPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Fatalf("expected project-scoped skill to be unapplied, stat error = %v", err)
	}
}

func TestAddRuntimeErrorSuppressesUsage(t *testing.T) {
	repoPath := t.TempDir()
	sourceDir := createTestSkill(t, t.TempDir(), "frontend-design")
	if _, err := AddSkills(repoPath, sourceDir, "", false, false); err != nil {
		t.Fatalf("AddSkills() error = %v", err)
	}

	cmd := NewCommand(func() string { return "oh-my-dot" }, func() string { return repoPath })
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"skills", "add", sourceDir, "--no-commit"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected duplicate add to fail")
	}

	combined := output.String() + err.Error()
	if strings.Contains(combined, "Usage:") {
		t.Fatalf("runtime error should not print usage, got %q", combined)
	}
	if !strings.Contains(combined, "destination already exists:") {
		t.Fatalf("expected formatted destination conflict, got %q", combined)
	}
}
