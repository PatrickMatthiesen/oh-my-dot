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

		WarnIfRemoteUpdatesSync(cmd)

		// Check remote push permissions (exit on error since push requires access)
		git.CheckRemoteAccessWithHelp(true)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if state, err := git.GetRemoteSyncState(); err == nil && state == git.RemoteSyncLocalAhead {
			fileops.ColorPrintln("Detected local committed changes. Pushing...", fileops.Cyan)
		}

		err := git.PushRepo()
		if err != nil {
			if git.IsSSHAgentError(err) {
				git.DisplaySSHAgentError(true)
			} else {
				fileops.ColorPrintfn(fileops.Red, "Error pushing changes: %s", err)
				os.Exit(1)
			}
			return
		}

		fileops.ColorPrintfn(fileops.Green, "Pushed changes to repository")
	},
}
