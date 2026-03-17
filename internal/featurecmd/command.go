package featurecmd

import "github.com/spf13/cobra"

type commandState struct {
	aliasProvider    func() string
	repoPathProvider func() string

	flagShell       []string
	flagAll         bool
	flagStrategy    string
	flagOnCommand   []string
	flagOption      []string
	flagDisabled    bool
	flagForce       bool
	flagInteractive bool
}

// NewCommand builds the feature command tree and keeps cmd/feature.go as a thin entrypoint.
func NewCommand(aliasProvider, repoPathProvider func() string) *cobra.Command {
	state := &commandState{
		aliasProvider:    aliasProvider,
		repoPathProvider: repoPathProvider,
	}

	featureCmd := &cobra.Command{
		Use:     "feature",
		Short:   "Manage shell features",
		Long:    `Add, remove, enable, disable, and list shell features`,
		GroupID: "dotfiles",
	}

	featureAddCmd := &cobra.Command{
		Use:   "add [feature]",
		Short: "Add a shell feature",
		Long: `Add a shell feature to one or more shells.

Interactive mode (-i): Browse and select features from the catalog
Non-interactive: Specify feature name directly

The command will intelligently select which shell(s) to add the feature to based on:
- Feature compatibility (which shells support this feature)
- Your current shell
- Interactive prompts (if multiple options available)

Examples:
  oh-my-dot feature add -i                    # Browse catalog interactively
  oh-my-dot feature add git-prompt
  oh-my-dot feature add kubectl-completion --shell bash
  oh-my-dot feature add core-aliases --all`,
		Args: cobra.MaximumNArgs(1),
		RunE: state.runFeatureAdd,
	}

	featureRemoveCmd := &cobra.Command{
		Use:   "remove [feature]",
		Short: "Remove a shell feature",
		Long: `Remove a shell feature from one or more shells.

If this is the last feature in a shell, the shell integration will be automatically
cleaned up (hooks removed from profile, directory deleted).

Interactive mode (-i): Browse features to remove
Non-interactive: Specify feature name directly

Examples:
  oh-my-dot feature remove -i                     # Browse features interactively
  oh-my-dot feature remove git-prompt
  oh-my-dot feature remove kubectl-completion --shell bash
  oh-my-dot feature remove core-aliases --all`,
		Args: cobra.MaximumNArgs(1),
		RunE: state.runFeatureRemove,
	}

	featureListCmd := &cobra.Command{
		Use:   "list",
		Short: "List shell features",
		Long: `List all enabled shell features, optionally filtered by shell.

Examples:
  oh-my-dot feature list
  oh-my-dot feature list --shell bash`,
		Args: cobra.NoArgs,
		RunE: state.runFeatureList,
	}

	featureEnableCmd := &cobra.Command{
		Use:   "enable <feature>",
		Short: "Enable a disabled feature",
		Long: `Enable a previously disabled feature without re-adding it.

Examples:
  oh-my-dot feature enable git-prompt
  oh-my-dot feature enable kubectl-completion --shell bash`,
		Args: cobra.ExactArgs(1),
		RunE: state.runFeatureEnable,
	}

	featureDisableCmd := &cobra.Command{
		Use:   "disable <feature>",
		Short: "Disable a feature without removing it",
		Long: `Disable a feature without deleting its configuration file.
This allows you to temporarily turn off a feature.

Examples:
  oh-my-dot feature disable git-prompt
  oh-my-dot feature disable kubectl-completion --shell bash`,
		Args: cobra.ExactArgs(1),
		RunE: state.runFeatureDisable,
	}

	featureInfoCmd := &cobra.Command{
		Use:   "info <feature>",
		Short: "Show detailed information about a feature",
		Long: `Display metadata about a feature from the catalog, including:
- Description
- Default load strategy
- Supported shells
- Current configuration (if installed)

Examples:
  oh-my-dot feature info git-prompt
  oh-my-dot feature info kubectl-completion`,
		Args: cobra.ExactArgs(1),
		RunE: state.runFeatureInfo,
	}

	featureCmd.AddCommand(featureAddCmd, featureRemoveCmd, featureListCmd, featureEnableCmd, featureDisableCmd, featureInfoCmd)

	featureAddCmd.Flags().BoolVarP(&state.flagInteractive, "interactive", "i", false, "Browse and select features from catalog")
	featureAddCmd.Flags().StringSliceVar(&state.flagShell, "shell", nil, "Target specific shell(s)")
	featureAddCmd.Flags().BoolVar(&state.flagAll, "all", false, "Add to all supported shells")
	featureAddCmd.Flags().StringVar(&state.flagStrategy, "strategy", "", "Override load strategy (eager, defer, on-command)")
	featureAddCmd.Flags().StringSliceVar(&state.flagOnCommand, "on-command", nil, "Set trigger commands for on-command strategy")
	featureAddCmd.Flags().StringSliceVar(&state.flagOption, "option", nil, "Set feature option (key=value); repeatable")
	featureAddCmd.Flags().BoolVar(&state.flagDisabled, "disabled", false, "Add feature but keep it disabled")

	featureRemoveCmd.Flags().StringSliceVar(&state.flagShell, "shell", nil, "Target specific shell(s)")
	featureRemoveCmd.Flags().BoolVar(&state.flagAll, "all", false, "Remove from all shells")
	featureRemoveCmd.Flags().BoolVar(&state.flagForce, "force", false, "Skip confirmation prompts")
	featureRemoveCmd.Flags().BoolVarP(&state.flagInteractive, "interactive", "i", false, "Browse and select features to remove")

	featureListCmd.Flags().StringSliceVar(&state.flagShell, "shell", nil, "Filter by specific shell(s)")

	featureEnableCmd.Flags().StringSliceVar(&state.flagShell, "shell", nil, "Target specific shell(s)")
	featureEnableCmd.Flags().StringVar(&state.flagStrategy, "strategy", "", "Override load strategy")
	featureEnableCmd.Flags().StringSliceVar(&state.flagOnCommand, "on-command", nil, "Set trigger commands")

	featureDisableCmd.Flags().StringSliceVar(&state.flagShell, "shell", nil, "Target specific shell(s)")
	featureDisableCmd.Flags().BoolVar(&state.flagAll, "all", false, "Disable in all shells")

	return featureCmd
}
