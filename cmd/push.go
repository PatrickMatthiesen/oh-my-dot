package cmd

import (
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
	Run: func(cmd *cobra.Command, args []string) {
		err := git.PushRepo()
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error pushing changes: %s", err)
			return
		}

		fileops.ColorPrintfn(fileops.Green, "Pushed changes to repository")
	},
}
