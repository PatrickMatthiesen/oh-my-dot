package agentcmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddSkillsAddsSingleSkillDirectory(t *testing.T) {
	repoPath := t.TempDir()
	sourceDir := createTestSkill(t, t.TempDir(), "code-review")

	added, err := AddSkills(repoPath, sourceDir, "", false)
	if err != nil {
		t.Fatalf("AddSkills() error = %v", err)
	}

	if len(added) != 1 || added[0].Name != "code-review" {
		t.Fatalf("AddSkills() = %v, want code-review", added)
	}

	targetReadme := filepath.Join(repoPath, "omd-agents", "skills", "code-review", "SKILL.md")
	content, err := os.ReadFile(targetReadme)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "# code-review\n\nTest skill.\n" {
		t.Fatalf("managed skill content = %q", string(content))
	}
}

func TestAddSkillsAddsMultipleSkillsFromParentDirectory(t *testing.T) {
	repoPath := t.TempDir()
	sourceRoot := t.TempDir()
	createTestSkill(t, sourceRoot, "alpha")
	createTestSkill(t, sourceRoot, "beta")

	added, err := AddSkills(repoPath, sourceRoot, "", false)
	if err != nil {
		t.Fatalf("AddSkills() error = %v", err)
	}

	if len(added) != 2 {
		t.Fatalf("AddSkills() added %d skills, want 2", len(added))
	}

	skills, err := ListSkills(repoPath)
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}
	if len(skills) != 2 || skills[0].Name != "alpha" || skills[1].Name != "beta" {
		t.Fatalf("ListSkills() = %v, want alpha and beta", skills)
	}
}

func TestAddSkillsRejectsNameOverrideForMultipleSkills(t *testing.T) {
	repoPath := t.TempDir()
	sourceRoot := t.TempDir()
	createTestSkill(t, sourceRoot, "alpha")
	createTestSkill(t, sourceRoot, "beta")

	_, err := AddSkills(repoPath, sourceRoot, "custom", false)
	if err == nil {
		t.Fatal("expected AddSkills() to reject --name for multiple skills")
	}
}

func TestInstallSkillProjectScope(t *testing.T) {
	repoPath := t.TempDir()
	projectPath := t.TempDir()
	sourceDir := createTestSkill(t, t.TempDir(), "code-review")

	if _, err := AddSkills(repoPath, sourceDir, "", false); err != nil {
		t.Fatalf("AddSkills() error = %v", err)
	}

	target, err := InstallSkill(repoPath, "code-review", "project", projectPath, false)
	if err != nil {
		t.Fatalf("InstallSkill() error = %v", err)
	}

	want := filepath.Join(projectPath, ".codex", "skills", "code-review")
	if target != want {
		t.Fatalf("InstallSkill() target = %q, want %q", target, want)
	}
	if _, err := os.Stat(filepath.Join(want, "SKILL.md")); err != nil {
		t.Fatalf("installed SKILL.md missing: %v", err)
	}
}

func TestUninstallSkillProjectScope(t *testing.T) {
	projectPath := t.TempDir()
	targetDir := filepath.Join(projectPath, ".codex", "skills", "code-review")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte("# code-review\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	removed, err := UninstallSkill("code-review", "project", projectPath)
	if err != nil {
		t.Fatalf("UninstallSkill() error = %v", err)
	}
	if removed != targetDir {
		t.Fatalf("UninstallSkill() removed = %q, want %q", removed, targetDir)
	}
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Fatalf("expected installed skill to be removed, stat error = %v", err)
	}
}

func createTestSkill(t *testing.T, parentDir, name string) string {
	t.Helper()

	skillDir := filepath.Join(parentDir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	content := "# " + name + "\n\nTest skill.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return skillDir
}
