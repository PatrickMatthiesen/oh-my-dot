package agentcmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddSkillsAddsSingleSkillDirectory(t *testing.T) {
	repoPath := t.TempDir()
	sourceDir := createTestSkill(t, t.TempDir(), "code-review")

	added, err := AddSkills(repoPath, sourceDir, "", false, false)
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

func TestAddSkillsAcceptsSkillReadmePath(t *testing.T) {
	repoPath := t.TempDir()
	sourceDir := createTestSkill(t, t.TempDir(), "code-review")

	added, err := AddSkills(repoPath, filepath.Join(sourceDir, "SKILL.md"), "", false, false)
	if err != nil {
		t.Fatalf("AddSkills() error = %v", err)
	}

	if len(added) != 1 || added[0].Name != "code-review" {
		t.Fatalf("AddSkills() = %v, want code-review", added)
	}
}

func TestAddSkillsAddsMultipleSkillsFromParentDirectory(t *testing.T) {
	repoPath := t.TempDir()
	sourceRoot := t.TempDir()
	createTestSkill(t, sourceRoot, "alpha")
	createTestSkill(t, sourceRoot, "beta")

	added, err := AddSkills(repoPath, sourceRoot, "", false, false)
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

func TestAddSkillsDoesNotRecurseByDefault(t *testing.T) {
	repoPath := t.TempDir()
	sourceRoot := t.TempDir()
	createTestSkill(t, sourceRoot, "alpha")
	createTestSkill(t, filepath.Join(sourceRoot, "nested"), "beta")

	added, err := AddSkills(repoPath, sourceRoot, "", false, false)
	if err != nil {
		t.Fatalf("AddSkills() error = %v", err)
	}

	if len(added) != 1 || added[0].Name != "alpha" {
		t.Fatalf("AddSkills() = %v, want only alpha", added)
	}

	if _, err := os.Stat(filepath.Join(repoPath, "omd-agents", "skills", "beta")); !os.IsNotExist(err) {
		t.Fatalf("nested skill should not be added without recurse, stat error = %v", err)
	}
}

func TestAddSkillsRecurseAddsNestedSkills(t *testing.T) {
	repoPath := t.TempDir()
	sourceRoot := t.TempDir()
	createTestSkill(t, sourceRoot, "alpha")
	createTestSkill(t, filepath.Join(sourceRoot, "nested"), "beta")

	added, err := AddSkills(repoPath, sourceRoot, "", false, true)
	if err != nil {
		t.Fatalf("AddSkills() error = %v", err)
	}

	if len(added) != 2 || added[0].Name != "alpha" || added[1].Name != "beta" {
		t.Fatalf("AddSkills() = %v, want alpha and beta", added)
	}
}

func TestAddSkillsRejectsDuplicateDiscoveredNamesBeforeCopying(t *testing.T) {
	repoPath := t.TempDir()
	sourceRoot := t.TempDir()
	createTestSkill(t, filepath.Join(sourceRoot, "one"), "duplicate")
	createTestSkill(t, filepath.Join(sourceRoot, "two"), "duplicate")

	_, err := AddSkills(repoPath, sourceRoot, "", false, true)
	if err == nil {
		t.Fatal("expected AddSkills() to reject duplicate discovered names")
	}

	managedSkillsDir := filepath.Join(repoPath, "omd-agents", "skills")
	if _, statErr := os.Stat(managedSkillsDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected duplicate preflight to avoid copying skills, stat error = %v", statErr)
	}
}

func TestAddSkillsRejectsNameOverrideForMultipleSkills(t *testing.T) {
	repoPath := t.TempDir()
	sourceRoot := t.TempDir()
	createTestSkill(t, sourceRoot, "alpha")
	createTestSkill(t, sourceRoot, "beta")

	_, err := AddSkills(repoPath, sourceRoot, "custom", false, false)
	if err == nil {
		t.Fatal("expected AddSkills() to reject --name for multiple skills")
	}
}

func TestApplySkillProjectScope(t *testing.T) {
	repoPath := t.TempDir()
	projectPath := t.TempDir()
	sourceDir := createTestSkill(t, t.TempDir(), "code-review")

	if _, err := AddSkills(repoPath, sourceDir, "", false, false); err != nil {
		t.Fatalf("AddSkills() error = %v", err)
	}

	target, err := ApplySkill(repoPath, "code-review", "project", projectPath, false)
	if err != nil {
		t.Fatalf("ApplySkill() error = %v", err)
	}

	want := filepath.Join(projectPath, ".codex", "skills", "code-review")
	if target != want {
		t.Fatalf("ApplySkill() target = %q, want %q", target, want)
	}
	if _, err := os.Stat(filepath.Join(want, "SKILL.md")); err != nil {
		t.Fatalf("applied SKILL.md missing: %v", err)
	}
}

func TestUnapplySkillProjectScope(t *testing.T) {
	projectPath := t.TempDir()
	targetDir := filepath.Join(projectPath, ".codex", "skills", "code-review")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "SKILL.md"), []byte("# code-review\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	removed, err := UnapplySkill("code-review", "project", projectPath)
	if err != nil {
		t.Fatalf("UnapplySkill() error = %v", err)
	}
	if removed != targetDir {
		t.Fatalf("UnapplySkill() removed = %q, want %q", removed, targetDir)
	}
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Fatalf("expected applied skill to be removed, stat error = %v", err)
	}
}

func TestUnapplySkillRejectsDirectoryWithoutSkillReadme(t *testing.T) {
	projectPath := t.TempDir()
	targetDir := filepath.Join(projectPath, ".codex", "skills", "code-review")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "notes.txt"), []byte("not a skill\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := UnapplySkill("code-review", "project", projectPath)
	if err == nil {
		t.Fatal("expected UnapplySkill() to reject non-skill directory")
	}

	if _, err := os.Stat(targetDir); err != nil {
		t.Fatalf("non-skill directory should not be removed, stat error = %v", err)
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
