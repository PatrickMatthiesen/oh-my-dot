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
	flagApply    bool
	flagRecurse  bool
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
		Use:     "skills",
		Aliases: []string{"skill"},
		Short:   "Manage agent skills",
		Long: `Track reusable agent skills in your dotfiles repository and apply them
into user or project scopes.`,
	}

	skillsAddCmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Add skills to oh-my-dot management",
		Long: `Add a skill directory to the managed oh-my-dot agent skills store.

The path may point at a single skill directory containing SKILL.md, at a
SKILL.md file, or at a parent directory containing immediate child skill directories.
Use --recurse to discover skill directories below that first level.

Examples:
  oh-my-dot agent skills add ./skills/code-review
  oh-my-dot agent skills add ./skills
  oh-my-dot agent skills add ./skills --recurse
  oh-my-dot agent skills add ./skills/code-review --apply
  oh-my-dot agent skills add ./skills/code-review --apply --scope user`,
		Args: cobra.ExactArgs(1),
		RunE: silenceUsageOnRun(state.runSkillsAdd),
	}

	skillsRemoveCmd := &cobra.Command{
		Use:   "remove <skill>",
		Short: "Remove a managed agent skill",
		Long: `Remove a skill from the managed oh-my-dot agent skills store.

This does not unapply already-applied user or project copies.`,
		Args: cobra.ExactArgs(1),
		RunE: silenceUsageOnRun(state.runSkillsRemove),
	}

	skillsListCmd := &cobra.Command{
		Use:   "list",
		Short: "List managed agent skills",
		Long:  `List agent skills tracked in the oh-my-dot repository.`,
		Args:  cobra.NoArgs,
		RunE:  silenceUsageOnRun(state.runSkillsList),
	}

	skillsInfoCmd := &cobra.Command{
		Use:   "info <skill>",
		Short: "Show managed skill information",
		Long:  `Show the managed path and summary for an agent skill.`,
		Args:  cobra.ExactArgs(1),
		RunE:  silenceUsageOnRun(state.runSkillsInfo),
	}

	skillsApplyCmd := &cobra.Command{
		Use:   "apply <skill>",
		Short: "Apply a managed skill into a scope",
		Long: `Apply a managed skill into a Codex-compatible user or project scope.

Project scope is used by default.

Examples:
  oh-my-dot agent skill apply code-review
  oh-my-dot agent skill apply code-review --scope user
  oh-my-dot agent skill apply code-review --scope project --project .`,
		Args: cobra.ExactArgs(1),
		RunE: silenceUsageOnRun(state.runSkillsApply),
	}

	skillsUnapplyCmd := &cobra.Command{
		Use:   "unapply <skill>",
		Short: "Unapply a skill from a scope",
		Long: `Remove an applied skill from a Codex-compatible user or project scope.

Project scope is used by default.

Examples:
  oh-my-dot agent skill unapply code-review
  oh-my-dot agent skill unapply code-review --scope user
  oh-my-dot agent skill unapply code-review --scope project --project .`,
		Args: cobra.ExactArgs(1),
		RunE: silenceUsageOnRun(state.runSkillsUnapply),
	}

	skillsAddCmd.Flags().StringVar(&state.flagName, "name", "", "Override skill name when adding a single skill")
	skillsAddCmd.Flags().BoolVar(&state.flagForce, "force", false, "Overwrite an existing managed skill")
	skillsAddCmd.Flags().BoolVar(&state.flagApply, "apply", false, "Also apply after adding")
	skillsAddCmd.Flags().BoolVarP(&state.flagRecurse, "recurse", "r", false, "Recursively search for skill directories")
	skillsAddCmd.Flags().StringVar(&state.flagScope, "scope", "project", "Apply scope when using --apply (project or user)")
	skillsAddCmd.Flags().StringVar(&state.flagProject, "project", ".", "Project path for project-scope apply")
	skillsAddCmd.Flags().BoolVar(&state.flagNoCommit, "no-commit", false, "Do not commit managed skill changes")

	skillsRemoveCmd.Flags().BoolVar(&state.flagNoCommit, "no-commit", false, "Do not commit managed skill changes")

	skillsInfoCmd.Flags().BoolVar(&state.flagLongInfo, "long", false, "Print the full SKILL.md content")

	skillsApplyCmd.Flags().StringVar(&state.flagScope, "scope", "project", "Apply scope (project or user)")
	skillsApplyCmd.Flags().StringVar(&state.flagProject, "project", ".", "Project path for project-scope apply")
	skillsApplyCmd.Flags().BoolVar(&state.flagForce, "force", false, "Overwrite an existing applied skill")

	skillsUnapplyCmd.Flags().StringVar(&state.flagScope, "scope", "project", "Apply scope (project or user)")
	skillsUnapplyCmd.Flags().StringVar(&state.flagProject, "project", ".", "Project path for project-scope apply")

	skillsCmd.AddCommand(skillsAddCmd, skillsRemoveCmd, skillsListCmd, skillsInfoCmd, skillsApplyCmd, skillsUnapplyCmd)
	agentCmd.AddCommand(skillsCmd)

	return agentCmd
}

func silenceUsageOnRun(runE func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return runE(cmd, args)
	}
}

func (state *commandState) runSkillsAdd(cmd *cobra.Command, args []string) error {
	repoPath := state.repoPathProvider()

	addedSkills, err := AddSkills(repoPath, args[0], state.flagName, state.flagForce, state.flagRecurse)
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

	if state.flagApply {
		for _, skill := range addedSkills {
			target, err := ApplySkill(repoPath, skill.Name, state.flagScope, state.flagProject, state.flagForce)
			if err != nil {
				return err
			}
			fileops.ColorPrintfn(fileops.Green, "Applied %s to %s", skill.Name, target)
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

func (state *commandState) runSkillsApply(cmd *cobra.Command, args []string) error {
	target, err := ApplySkill(state.repoPathProvider(), args[0], state.flagScope, state.flagProject, state.flagForce)
	if err != nil {
		return err
	}

	fileops.ColorPrintfn(fileops.Green, "Applied %s to %s", args[0], target)
	return nil
}

func (state *commandState) runSkillsUnapply(cmd *cobra.Command, args []string) error {
	target, err := UnapplySkill(args[0], state.flagScope, state.flagProject)
	if err != nil {
		return err
	}

	fileops.ColorPrintfn(fileops.Green, "Unapplied %s from %s", args[0], target)
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
