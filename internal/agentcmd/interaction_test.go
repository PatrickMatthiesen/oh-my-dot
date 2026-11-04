package agentcmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
)

func TestSkillCommandsMissingInputNeverPromptInScripts(t *testing.T) {
	originalInput, originalSelect := promptSkillPath, selectSkillName
	t.Cleanup(func() { promptSkillPath, selectSkillName = originalInput, originalSelect })
	promptSkillPath = func(string, string) (string, error) { t.Fatal("path prompt in script"); return "", nil }
	selectSkillName = func(string, []string) (int, error) { t.Fatal("selection prompt in script"); return 0, nil }
	for _, mode := range []string{"CI", "environment", "flag", "pipe"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("CI", "")
			t.Setenv("OMDOT_NON_INTERACTIVE", "")
			switch mode {
			case "CI":
				t.Setenv("CI", "true")
			case "environment":
				t.Setenv("OMDOT_NON_INTERACTIVE", "1")
			case "pipe":
				input, output, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				previous := os.Stdin
				os.Stdin = input
				t.Cleanup(func() { os.Stdin = previous; input.Close(); output.Close() })
			}
			for _, operation := range []string{"add", "remove", "info", "apply", "unapply"} {
				t.Run(operation, func(t *testing.T) {
					repo := t.TempDir()
					project := t.TempDir()
					command := NewCommand(func() string { return "omd" }, func() string { return repo })
					command.PersistentFlags().Bool("no-interactive", false, "Disable prompts")
					args := []string{"skills", operation}
					if mode == "flag" {
						args = append(args, "--no-interactive")
					}
					if operation == "apply" || operation == "unapply" {
						args = append(args, "--project", project)
					}
					command.SetArgs(args)
					err := command.Execute()
					if err == nil || !strings.Contains(err.Error(), "required") {
						t.Fatalf("missing actionable error: %v", err)
					}
					entries, _ := os.ReadDir(repo)
					if len(entries) != 0 {
						t.Fatal("missing input mutated repository")
					}
					entries, _ = os.ReadDir(project)
					if len(entries) != 0 {
						t.Fatal("missing input mutated project")
					}
				})
			}
		})
	}
}

func TestSkillSelectorsUseManagedAndAppliedState(t *testing.T) {
	originalAllowed, originalInput, originalSelect := shouldPromptSkill, promptSkillPath, selectSkillName
	t.Cleanup(func() {
		shouldPromptSkill, promptSkillPath, selectSkillName = originalAllowed, originalInput, originalSelect
	})
	shouldPromptSkill = func(*cobra.Command, bool) bool { return true }
	repo := t.TempDir()
	project := t.TempDir()
	source := createTestSkill(t, t.TempDir(), "managed")
	if _, err := AddSkills(repo, source, "", false, false); err != nil {
		t.Fatal(err)
	}
	applied := filepath.Join(project, ".codex", "skills")
	createTestSkill(t, applied, "applied-only")
	state := &commandState{repoPathProvider: func() string { return repo }, flagScope: "project", flagProject: project}
	command := &cobra.Command{}
	promptSkillPath = func(question, defaultValue string) (string, error) { return source, nil }
	selected, err := state.resolveSkillArgument(command, nil, "add")
	if err != nil || selected != source {
		t.Fatalf("path selection %q %v", selected, err)
	}
	for _, operation := range []string{"remove", "info", "apply", "unapply"} {
		selectSkillName = func(question string, names []string) (int, error) {
			expected := "managed"
			if operation == "unapply" {
				expected = "applied-only"
			}
			if len(names) != 1 || names[0] != expected {
				t.Fatalf("%s candidates %v", operation, names)
			}
			return 0, nil
		}
		name, err := state.resolveSkillArgument(command, nil, operation)
		if err != nil || name == "" {
			t.Fatalf("%s selection %q %v", operation, name, err)
		}
	}
	// Explicit arguments never open a selector, even when a terminal is available.
	selectSkillName = func(string, []string) (int, error) { t.Fatal("prompt for complete command"); return 0, nil }
	if name, err := state.resolveSkillArgument(command, []string{"managed"}, "info"); err != nil || name != "managed" {
		t.Fatalf("direct selection %q %v", name, err)
	}
}

func TestMissingSkillSelectionCancellationPreservesFiles(t *testing.T) {
	originalAllowed, originalSelect := shouldPromptSkill, selectSkillName
	t.Cleanup(func() { shouldPromptSkill, selectSkillName = originalAllowed, originalSelect })
	shouldPromptSkill = func(*cobra.Command, bool) bool { return true }
	selectSkillName = func(string, []string) (int, error) { return 0, interactive.ErrCancelled }
	repo := t.TempDir()
	source := createTestSkill(t, t.TempDir(), "keep-me")
	if _, err := AddSkills(repo, source, "", false, false); err != nil {
		t.Fatal(err)
	}
	state := &commandState{repoPathProvider: func() string { return repo }, flagNoCommit: true}
	if err := state.runSkillsRemove(&cobra.Command{}, nil); !errors.Is(err, interactive.ErrCancelled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if _, err := os.Stat(skillReadmePath(managedSkillDirectory(repo, "keep-me"))); err != nil {
		t.Fatal("cancelled removal changed files")
	}
}

func setupAgentCommitRepo(t *testing.T, identity bool) (string, *gogit.Repository) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "none"))
	t.Setenv("GIT_AUTHOR_NAME", "")
	t.Setenv("GIT_AUTHOR_EMAIL", "")
	t.Setenv("CI", "true")
	path := t.TempDir()
	repo, err := gogit.PlainInit(path, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	if identity {
		config, err := repo.Config()
		if err != nil {
			t.Fatal(err)
		}
		config.User.Name = "Agent Test"
		config.User.Email = "agent@example.invalid"
		if err := repo.SetConfig(config); err != nil {
			t.Fatal(err)
		}
	}
	viper.Set("repo-path", path)
	return path, repo
}

func TestAgentMutationPreflightPreservesFiles(t *testing.T) {
	for _, operation := range []string{"add", "remove"} {
		for _, failure := range []string{"identity", "staged file", "agent edit"} {
			t.Run(operation+"/"+failure, func(t *testing.T) {
				path, repo := setupAgentCommitRepo(t, failure != "identity")
				source := createTestSkill(t, t.TempDir(), "existing")
				if operation == "remove" || failure == "agent edit" {
					if _, err := AddSkills(path, source, "", false, false); err != nil {
						t.Fatal(err)
					}
					if failure != "identity" {
						wt, _ := repo.Worktree()
						if err := wt.AddWithOptions(&gogit.AddOptions{All: true}); err != nil {
							t.Fatal(err)
						}
						if _, err := wt.Commit("initial", &gogit.CommitOptions{}); err != nil {
							t.Fatal(err)
						}
					}
				}
				if failure == "staged file" {
					os.WriteFile(filepath.Join(path, "unrelated"), []byte("preserve"), 0600)
					wt, _ := repo.Worktree()
					if _, err := wt.Add("unrelated"); err != nil {
						t.Fatal(err)
					}
				}
				if failure == "agent edit" {
					if err := os.WriteFile(skillReadmePath(managedSkillDirectory(path, "existing")), []byte("private edit"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				adding := createTestSkill(t, t.TempDir(), "new-skill")
				command := NewCommand(func() string { return "omd" }, func() string { return path })
				args := []string{"skills", operation, "existing"}
				if operation == "add" {
					args[2] = adding
				}
				command.SetArgs(args)
				err := command.Execute()
				if err == nil {
					t.Fatal("unsafe mutation succeeded")
				}
				if failure == "identity" && !strings.Contains(err.Error(), "user.name") {
					t.Fatalf("missing identity guidance %v", err)
				}
				if _, err := os.Stat(managedSkillDirectory(path, "new-skill")); !os.IsNotExist(err) {
					t.Fatal("failed preflight copied skill")
				}
				if operation == "remove" {
					content, readErr := os.ReadFile(skillReadmePath(managedSkillDirectory(path, "existing")))
					if readErr != nil {
						t.Fatalf("failed preflight deleted skill: %v", readErr)
					}
					if failure == "agent edit" && string(content) != "private edit" {
						t.Fatal("failed preflight changed edited skill")
					}
				}
			})
		}
	}
}

func TestSkillNoCommitAddAndRemoveAllowCallerManagedStaging(t *testing.T) {
	path, repo := setupAgentCommitRepo(t, false)
	os.WriteFile(filepath.Join(path, "unrelated"), []byte("staged"), 0600)
	wt, _ := repo.Worktree()
	if _, err := wt.Add("unrelated"); err != nil {
		t.Fatal(err)
	}
	source := createTestSkill(t, t.TempDir(), "sample")
	for _, args := range [][]string{{"skills", "add", source, "--no-commit"}, {"skills", "remove", "sample", "--no-commit"}} {
		command := NewCommand(func() string { return "omd" }, func() string { return path })
		command.SetArgs(args)
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	status, err := wt.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status["unrelated"].Staging != gogit.Added {
		t.Fatal("caller staging was changed")
	}
	if _, err := repo.Head(); err == nil {
		t.Fatal("--no-commit created history")
	}
}

func TestAgentSkillAutoCommitPreservesUnrelatedWorktreeFiles(t *testing.T) {
	path, repo := setupAgentCommitRepo(t, true)
	unrelated := filepath.Join(path, "unrelated")
	if err := os.WriteFile(unrelated, []byte("keep local"), 0600); err != nil {
		t.Fatal(err)
	}
	source := createTestSkill(t, t.TempDir(), "sample")
	for _, args := range [][]string{{"skills", "add", source}, {"skills", "remove", "sample"}} {
		command := NewCommand(func() string { return "omd" }, func() string { return path })
		command.SetArgs(args)
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		head, err := repo.Head()
		if err != nil {
			t.Fatal(err)
		}
		commit, err := repo.CommitObject(head.Hash())
		if err != nil {
			t.Fatal(err)
		}
		tree, err := commit.Tree()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tree.File("unrelated"); err == nil {
			t.Fatal("automatic skill commit included unrelated local file")
		}
		if content, err := os.ReadFile(unrelated); err != nil || string(content) != "keep local" {
			t.Fatalf("unrelated file changed: %q %v", content, err)
		}
	}
	if _, err := os.Stat(managedSkillDirectory(path, "sample")); !os.IsNotExist(err) {
		t.Fatal("managed skill remained after removal")
	}
}
