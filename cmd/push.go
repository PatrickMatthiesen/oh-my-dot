package cmd

import (
	"os"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pushCommand)
}

var pushCommand = &cobra.Command{
	Aliases:          []string{"p"},
	Use:              "push",
	Short:            "Push changes to the remote repository",
	Long:             `Push changes to the remote repository.`,
	TraverseChildren: true,
	GroupID:          "dotfiles",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Check write permissions on the repository
		if err := git.CheckRepoWritePermission(); err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error: %s", err)
			os.Exit(1)
		}

		// Check remote push permissions
		if err := git.CheckRemotePushPermission(); err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error: %s", err)
			fileops.ColorPrintln("Cannot push to remote repository. Please check your credentials and network connection.", fileops.Red)
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := git.PushRepo()
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error pushing changes: %s", err)
			return
		}

		fileops.ColorPrintfn(fileops.Green, "Pushed changes to repository")
	},
}
