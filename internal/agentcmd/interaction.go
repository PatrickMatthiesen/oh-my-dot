package agentcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	"github.com/spf13/cobra"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
)

var shouldPromptSkill = interactive.ShouldPrompt
var promptSkillPath = interactive.PromptInput
var selectSkillName = interactive.PromptSelect

func (state *commandState) resolveSkillArgument(cmd *cobra.Command, args []string, operation string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	if !shouldPromptSkill(cmd, false) {
		if operation == "add" {
			return "", fmt.Errorf("skill path required; use agent skills add <path>, or run it in a terminal to enter a path")
		}
		return "", fmt.Errorf("skill name required; use agent skills %s <skill>, or run it in a terminal to select one", operation)
	}
	if operation == "add" {
		source, err := promptSkillPath("Path to a skill or skills directory:", "")
		if err != nil {
			return "", fmt.Errorf("select skill source: %w", err)
		}
		if strings.TrimSpace(source) == "" {
			return "", fmt.Errorf("skill path is required")
		}
		return source, nil
	}
	var skills []Skill
	var err error
	if operation == "unapply" {
		var root string
		root, err = targetSkillsDirectory(state.flagScope, state.flagProject)
		if err != nil {
			return "", err
		}
		skills, err = listSkillDirectory(root)
	} else {
		skills, err = ListSkills(state.repoPathProvider())
	}
	if err != nil {
		return "", err
	}
	if len(skills) == 0 {
		if operation == "unapply" {
			return "", fmt.Errorf("no applied skills found in %s scope", state.flagScope)
		}
		return "", fmt.Errorf("no managed agent skills found; run agent skills add <path> first")
	}
	names := make([]string, len(skills))
	for i, skill := range skills {
		names[i] = skill.Name
	}
	index, err := selectSkillName("Select a skill to "+operation+":", names)
	if err != nil {
		return "", fmt.Errorf("select agent skill: %w", err)
	}
	if index < 0 || index >= len(names) {
		return "", fmt.Errorf("no agent skill selected")
	}
	return names[index], nil
}

// preflightAgentMutation prevents a broad agent-store commit from including
// earlier work or failing only after skill files were copied or deleted.
func preflightAgentMutation(repoPath string, noCommit bool) error {
	if noCommit {
		return nil
	}
	if err := git.CheckCommitSafety(repoPath); err != nil {
		return err
	}
	repo, err := git.GetRepository(repoPath)
	if err != nil {
		return err
	}
	defer repo.Close()
	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("inspect managed agent files: %w", err)
	}
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("inspect managed agent changes: %w", err)
	}
	for path, change := range status {
		if change.Staging == gogit.Unmodified && change.Worktree == gogit.Unmodified {
			continue
		}
		if strings.HasPrefix(filepath.ToSlash(path), agentDirectoryName+"/") {
			return fmt.Errorf("managed agent files already have uncommitted changes at %s; commit or stash them first, or use --no-commit to manage the changes yourself", path)
		}
	}
	return nil
}

func listSkillDirectory(root string) ([]Skill, error) {
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []Skill{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read applied skills: %w", err)
	}
	skills := make([]Skill, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && sourceContainsSkill(filepath.Join(root, entry.Name())) {
			skills = append(skills, Skill{Name: entry.Name(), Path: filepath.Join(root, entry.Name())})
		}
	}
	return skills, nil
}
