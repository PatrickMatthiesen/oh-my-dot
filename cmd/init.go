package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/config"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
)

func init() {
	initcmd.Flags().StringP("remote", "r", "", "URL of the remote repository, (local paths are also supported)")
	// initcmd.Flags().SetInterspersed(true)
	viper.BindPFlag("remote-url", initcmd.Flags().Lookup("remote"))

	initcmd.Flags().StringP("folder", "f", config.GetDefaultRepoPath(), "Path to the root of the dotfiles repository")
	initcmd.MarkFlagDirname("folder")
	viper.BindPFlag("repo-path", initcmd.Flags().Lookup("folder"))

	initcmd.Flags().BoolP("force", "", false, "Force initialization if previously initialized") //  or if given directory is not empty?
	rootCmd.AddCommand(initcmd)
}

var initcmd = &cobra.Command{
	Aliases:      []string{"i"},
	Use:          "init [url] [folder]",
	Args:         cobra.MaximumNArgs(2),
	SilenceUsage: true,
	Short:        "Initialize dotfiles management",
	Long: `Initialize dotfiles management.
Makes a git repository and sets remote origin to the specified URL.
The clone is placed in $HOME/dotfiles by default, but can be changed with --folder <new path>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			viper.Set("repo-path", args[1])
		}
		if len(args) > 0 {
			viper.Set("remote-url", args[0])
		}

		if git.IsGitRepo(viper.GetString("repo-path")) {
			if err := git.InitFromExistingRepo(viper.GetString("repo-path")); err != nil {
				return fmt.Errorf("initialize existing repository: %w", err)
			}
			if len(args) > 0 || cmd.Flags().Changed("remote") {
				requested, _ := cmd.Flags().GetString("remote")
				if len(args) > 0 {
					requested = args[0]
				}
				if requested != viper.GetString("remote-url") {
					return fmt.Errorf("existing repository has a different origin; use git -C %q remote set-url origin %q to change it", viper.GetString("repo-path"), requested)
				}
			}
			viper.Set("initialized", true)
			if err := viper.WriteConfig(); err != nil {
				return fmt.Errorf("save initialization config: %w", err)
			}
			fileops.ColorPrintln("Dotfiles repo initialized 🎉🎉🎉", fileops.Green)
			return nil
		}

		// allow for the remote url to be set in args
		if viper.GetString("remote-url") == "" && len(args) > 0 {
			viper.Set("remote-url", args[0])
		}

		// If no remote URL is provided, handle based on mode
		if viper.GetString("remote-url") == "" {
			// Check if we should prompt
			if interactive.ShouldPrompt(cmd, false) {
				// Ask if user wants to use a remote repository
				useRemote, err := interactive.PromptConfirm("Do you want to use a remote repository?")
				if err != nil {
					return fmt.Errorf("initialization cancelled: %w", err)
				}

				if useRemote {
					// Prompt for remote URL
					remoteURL, err := interactive.PromptInput("Enter remote repository URL:", "")
					if err != nil {
						return fmt.Errorf("initialization cancelled: %w", err)
					}
					if remoteURL == "" {
						return fmt.Errorf("no remote URL provided")
					}
					viper.Set("remote-url", remoteURL)
				}
			} else {
				// Non-interactive mode: error
				fileops.ColorPrintln("No remote URL specified", fileops.Red)
				fileops.ColorPrintln("Use: "+cmd.Root().Name()+" init <url> or set --remote flag", fileops.Yellow)
				return fmt.Errorf("no remote URL specified")
			}
		}

		_, err := git.InitGitRepo(viper.GetString("repo-path"), viper.GetString("remote-url"))
		if err != nil {
			return fmt.Errorf("initialize git repository: %w", err)
		}

		// write the config to the config file
		viper.Set("initialized", true)
		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("save initialization config: %w", err)
		}
		fileops.ColorPrintln("Dotfiles repo initialized 🎉🎉🎉", fileops.Green)
		return nil
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := interactive.ValidateMode(cmd); err != nil {
			return fmt.Errorf("invalid interaction mode: %w", err)
		}
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return fmt.Errorf("read force flag: %w", err)
		}

		if viper.GetBool("initialized") && git.IsGitRepo(viper.GetString("repo-path")) && !force && len(args) < 2 && !cmd.Flags().Changed("folder") {
			fileops.ColorPrintln("Dotfiles repository has been initialized previously", fileops.Yellow)
			fileops.ColorPrintln("Use the --force flag to reinitialize the repository", fileops.Blue)
			os.Exit(0)
		}
		return nil
	},
	GroupID: "basics",
	Example: `oh-my-dot init github.com/username/dotfiles
oh-my-dot init -r github.com/username/dotfiles -f $HOME/myCoolDotfiles`,
}
