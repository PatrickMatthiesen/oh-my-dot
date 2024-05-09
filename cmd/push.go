package cmd

import (
	"github.com/PatrickMatthiesen/oh-my-dot/util"
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
		err := util.PushRepo()
		if err != nil {
			util.ColorPrintfn(util.Red, "Error pushing changes: %s", err)
			return
		}

		util.ColorPrintfn(util.Green, "Pushed changes to repository")
	},
}
