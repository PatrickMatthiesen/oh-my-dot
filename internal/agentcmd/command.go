package agentcmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/spf13/cobra"
)

type commandState struct {
	aliasProvider    func() string
	repoPathProvider func() string

	flagName     string
	flagForce    bool
	flagInstall  string
	flagScope    string
	flagProject  string
	flagNoCommit bool
	flagLongInfo bool
}

// NewCommand builds the agent command tree.
func NewCommand(aliasProvider, repoPathProvider func() string) *cobra.Command {
	state := &commandState{
		aliasProvider:    aliasProvider,
		repoPathProvider: repoPathProvider,
	}

	agentCmd := &cobra.Command{
		Use:     "agent",
		Short:   "Manage agent configuration",
		Long:    `Manage portable AI agent configuration such as reusable skills.`,
		GroupID: "dotfiles",
	}

	skillsCmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage agent skills",
		Long: `Track reusable agent skills in your dotfiles repository and install them
into user or project scopes.`,
	}

	skillsAddCmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Add skills to oh-my-dot management",
		Long: `Add a skill directory to the managed oh-my-dot agent skills store.

The path may point at a single skill directory containing SKILL.md, at a
SKILL.md file, or at a parent directory containing multiple skill directories.

Examples:
  oh-my-dot agent skills add ./skills/code-review
  oh-my-dot agent skills add ./skills
  oh-my-dot agent skills add ./skills/code-review --install user
  oh-my-dot agent skills add ./skills/code-review --install project --project .`,
		Args: cobra.ExactArgs(1),
		RunE: state.runSkillsAdd,
	}

	skillsRemoveCmd := &cobra.Command{
		Use:   "remove <skill>",
		Short: "Remove a managed agent skill",
		Long: `Remove a skill from the managed oh-my-dot agent skills store.

This does not uninstall already-installed user or project copies.`,
		Args: cobra.ExactArgs(1),
		RunE: state.runSkillsRemove,
	}

	skillsListCmd := &cobra.Command{
		Use:   "list",
		Short: "List managed agent skills",
		Long:  `List agent skills tracked in the oh-my-dot repository.`,
		Args:  cobra.NoArgs,
		RunE:  state.runSkillsList,
	}

	skillsInfoCmd := &cobra.Command{
		Use:   "info <skill>",
		Short: "Show managed skill information",
		Long:  `Show the managed path and summary for an agent skill.`,
		Args:  cobra.ExactArgs(1),
		RunE:  state.runSkillsInfo,
	}

	skillsInstallCmd := &cobra.Command{
		Use:   "install <skill>",
		Short: "Install a managed skill into a scope",
		Long: `Install a managed skill into a Codex-compatible user or project scope.

Examples:
  oh-my-dot agent skills install code-review --scope user
  oh-my-dot agent skills install code-review --scope project --project .`,
		Args: cobra.ExactArgs(1),
		RunE: state.runSkillsInstall,
	}

	skillsUninstallCmd := &cobra.Command{
		Use:   "uninstall <skill>",
		Short: "Uninstall a skill from a scope",
		Long: `Remove an installed skill from a Codex-compatible user or project scope.

Examples:
  oh-my-dot agent skills uninstall code-review --scope user
  oh-my-dot agent skills uninstall code-review --scope project --project .`,
		Args: cobra.ExactArgs(1),
		RunE: state.runSkillsUninstall,
	}

	skillsAddCmd.Flags().StringVar(&state.flagName, "name", "", "Override skill name when adding a single skill")
	skillsAddCmd.Flags().BoolVar(&state.flagForce, "force", false, "Overwrite an existing managed skill")
	skillsAddCmd.Flags().StringVar(&state.flagInstall, "install", "", "Also install after adding (user or project)")
	skillsAddCmd.Flags().StringVar(&state.flagProject, "project", ".", "Project path for project-scope install")
	skillsAddCmd.Flags().BoolVar(&state.flagNoCommit, "no-commit", false, "Do not commit managed skill changes")

	skillsRemoveCmd.Flags().BoolVar(&state.flagNoCommit, "no-commit", false, "Do not commit managed skill changes")

	skillsInfoCmd.Flags().BoolVar(&state.flagLongInfo, "long", false, "Print the full SKILL.md content")

	skillsInstallCmd.Flags().StringVar(&state.flagScope, "scope", "", "Install scope (user or project)")
	skillsInstallCmd.Flags().StringVar(&state.flagProject, "project", ".", "Project path for project-scope install")
	skillsInstallCmd.Flags().BoolVar(&state.flagForce, "force", false, "Overwrite an existing installed skill")

	skillsUninstallCmd.Flags().StringVar(&state.flagScope, "scope", "", "Install scope (user or project)")
	skillsUninstallCmd.Flags().StringVar(&state.flagProject, "project", ".", "Project path for project-scope install")

	skillsCmd.AddCommand(skillsAddCmd, skillsRemoveCmd, skillsListCmd, skillsInfoCmd, skillsInstallCmd, skillsUninstallCmd)
	agentCmd.AddCommand(skillsCmd)

	return agentCmd
}

func (state *commandState) runSkillsAdd(cmd *cobra.Command, args []string) error {
	repoPath := state.repoPathProvider()

	addedSkills, err := AddSkills(repoPath, args[0], state.flagName, state.flagForce)
	if err != nil {
		return err
	}

	for _, skill := range addedSkills {
		fileops.ColorPrintfn(fileops.Green, "Added agent skill %s", skill.Name)
	}

	if !state.flagNoCommit {
		if err := autoCommitAgentSkillChanges("Add agent skills"); err != nil {
			return fmt.Errorf("failed to commit agent skill changes: %w", err)
		}
	}

	if state.flagInstall != "" {
		for _, skill := range addedSkills {
			target, err := InstallSkill(repoPath, skill.Name, state.flagInstall, state.flagProject, state.flagForce)
			if err != nil {
				return err
			}
			fileops.ColorPrintfn(fileops.Green, "Installed %s to %s", skill.Name, target)
		}
	}

	fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", state.aliasProvider())
	return nil
}

func (state *commandState) runSkillsRemove(cmd *cobra.Command, args []string) error {
	if err := RemoveSkill(state.repoPathProvider(), args[0]); err != nil {
		return err
	}

	fileops.ColorPrintfn(fileops.Green, "Removed managed agent skill %s", args[0])
	if !state.flagNoCommit {
		if err := autoCommitAgentSkillChanges("Remove agent skill: " + args[0]); err != nil {
			return fmt.Errorf("failed to commit agent skill changes: %w", err)
		}
	}

	fileops.ColorPrintfn(fileops.Cyan, "Run '%s push' to push committed changes", state.aliasProvider())
	return nil
}

func (state *commandState) runSkillsList(cmd *cobra.Command, args []string) error {
	skills, err := ListSkills(state.repoPathProvider())
	if err != nil {
		return err
	}

	if len(skills) == 0 {
		fileops.ColorPrintln("No managed agent skills found", fileops.Yellow)
		return nil
	}

	for _, skill := range skills {
		fmt.Fprintln(cmd.OutOrStdout(), skill.Name)
	}
	return nil
}

func (state *commandState) runSkillsInfo(cmd *cobra.Command, args []string) error {
	skill, err := GetSkill(state.repoPathProvider(), args[0])
	if err != nil {
		return err
	}

	content, err := os.ReadFile(skillReadmePath(skill.Path))
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", skillReadmeName, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", skill.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Path: %s\n", skill.Path)

	if state.flagLongInfo {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprint(cmd.OutOrStdout(), string(content))
		return nil
	}

	summary := summarizeSkillReadme(string(content))
	if summary != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Summary: %s\n", summary)
	}
	return nil
}

func (state *commandState) runSkillsInstall(cmd *cobra.Command, args []string) error {
	if state.flagScope == "" {
		return fmt.Errorf("--scope is required (user or project)")
	}

	target, err := InstallSkill(state.repoPathProvider(), args[0], state.flagScope, state.flagProject, state.flagForce)
	if err != nil {
		return err
	}

	fileops.ColorPrintfn(fileops.Green, "Installed %s to %s", args[0], target)
	return nil
}

func (state *commandState) runSkillsUninstall(cmd *cobra.Command, args []string) error {
	if state.flagScope == "" {
		return fmt.Errorf("--scope is required (user or project)")
	}

	target, err := UninstallSkill(args[0], state.flagScope, state.flagProject)
	if err != nil {
		return err
	}

	fileops.ColorPrintfn(fileops.Green, "Uninstalled %s from %s", args[0], target)
	return nil
}

func summarizeSkillReadme(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "---") {
			continue
		}
		return trimmed
	}
	return ""
}

func autoCommitAgentSkillChanges(commitMessage string) error {
	committed, err := git.StageAndCommitAgentSkillChanges(commitMessage)
	if err != nil {
		return err
	}

	if committed {
		fileops.ColorPrintln("Agent skill changes committed.", fileops.Green)
	} else {
		fileops.ColorPrintln("No committable agent skill changes detected.", fileops.Yellow)
	}

	return nil
}
