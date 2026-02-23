package cmd

import (
	"os"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pullCommand)
}

var pullCommand = &cobra.Command{
	Aliases: []string{"pl"},
	Use:     "pull",
	Short:   "Pull changes from the remote repository",
	Long:    `Pull changes from the remote repository.`,
	GroupID: "dotfiles",
	PreRun: func(cmd *cobra.Command, args []string) {
		git.CheckRemoteAccessWithHelp(true)
	},
	Run: func(cmd *cobra.Command, args []string) {
		state, err := git.GetRemoteSyncState()
		if err == nil {
			switch state {
			case git.RemoteSyncUpToDate:
				fileops.ColorPrintfn(fileops.Green, "Already up to date")
				return
			case git.RemoteSyncLocalAhead:
				fileops.ColorPrintfn(fileops.Cyan, "Local repository is ahead of remote. Nothing to pull.")
				return
			}
		}

		updated, err := git.PullRepo()
		if err != nil {
			if git.IsSSHAgentError(err) {
				git.DisplaySSHAgentError(true)
			} else if state == git.RemoteSyncDiverged {
				fileops.ColorPrintfn(fileops.Red, "Local and remote history diverged. Resolve conflicts and retry pull: %s", err)
				os.Exit(1)
			}
			fileops.ColorPrintfn(fileops.Red, "Error pulling changes: %s", err)
			os.Exit(1)
		}

		if !updated {
			fileops.ColorPrintfn(fileops.Green, "Already up to date")
			return
		}

		fileops.ColorPrintfn(fileops.Green, "Pulled latest changes from repository")
	},
}
