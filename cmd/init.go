package cmd

import (
	// "fmt"
	"os"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/config"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/exitcodes"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	initcmd.Flags().StringP("remote", "r", "", "URL of the remote repository, (local paths are also supported)")
	// initcmd.Flags().SetInterspersed(true)
	viper.BindPFlag("remote-url", initcmd.Flags().Lookup("remote"))

	initcmd.Flags().StringP("folder", "f", config.GetDefaultRepoPath(), "Path to the root of the dotfiles repository")
	initcmd.MarkFlagDirname("folder")
	viper.BindPFlag("repo-path", initcmd.Flags().Lookup("folder"))

	initcmd.Flags().BoolP("force", "", false, "Force initialization if a priveously initialized") //  or if given directory is not empty?
	rootCmd.AddCommand(initcmd)
}

var initcmd = &cobra.Command{
	Aliases: []string{"i"},
	Use:     "init <url> [folder] [...flags]",
	Short:   "Initialize dotfiles management",
	Long: `Initialize dotfiles management.
Makes a git repository and sets remote origin to the specified URL.
The clone is placed in $HOME/dotfiles by default, but can be changed with --folder <new path>`,
	Run: func(cmd *cobra.Command, args []string) {
		if git.IsGitRepo(viper.GetString("repo-path")) {
			git.InitFromExistingRepo(viper.GetString("repo-path"))
			fileops.ColorPrintln("Dotfiles repo initialized 🎉🎉🎉", fileops.Green)
			viper.Set("initialized", true)
			viper.WriteConfig()
			return
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
					fileops.ColorPrintln("Cancelled", fileops.Yellow)
					os.Exit(exitcodes.Error)
					return
				}

				if useRemote {
					// Prompt for remote URL
					remoteURL, err := interactive.PromptInput("Enter remote repository URL:", "")
					if err != nil {
						fileops.ColorPrintln("Cancelled", fileops.Yellow)
						os.Exit(exitcodes.Error)
					}
					if remoteURL == "" {
						fileops.ColorPrintln("No remote URL provided", fileops.Red)
						os.Exit(exitcodes.MissingArgs)
					}
					viper.Set("remote-url", remoteURL)
				}
			} else {
				// Non-interactive mode: error
				fileops.ColorPrintln("No remote URL specified", fileops.Red)
				fileops.ColorPrintln("Use: "+cmd.Root().Name()+" init <url> or set --remote flag", fileops.Yellow)
				os.Exit(exitcodes.MissingArgs)
			}
		}

		_, err := git.InitGitRepo(viper.GetString("repo-path"), viper.GetString("remote-url"))
		fileops.CheckIfErrorWithMessage(err, "Error initializing git repository")

		fileops.ColorPrintln("Dotfiles repo initialized 🎉🎉🎉", fileops.Green)

		// write the config to the config file
		viper.Set("initialized", true)
		viper.WriteConfig()
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		force, err := cmd.Flags().GetBool("force")
		fileops.CheckIfErrorWithMessage(err, "Error getting force flag")

		if viper.IsSet("initialized") && !force {
			fileops.ColorPrintln("Dotfiles repository has been initialized previously", fileops.Yellow)
			fileops.ColorPrintln("Use the --force flag to reinitialize the repository", fileops.Blue)
			os.Exit(0)
		}
	},
	GroupID: "basics",
	Example: `oh-my-dot init github.com/username/dotfiles
oh-my-dot init -r github.com/username/dotfiles -f $HOME/myCoolDotfiles`,
}
