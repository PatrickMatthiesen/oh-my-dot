package cmd

import (
	// "fmt"

	"github.com/PatrickMatthiesen/oh-my-dot/util"

	// "github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	addCommand.Flags().StringP("file", "f", "", "file to add") //
	// addCommand.MarkFlagRequired("file")
	viper.BindPFlag("file", addCommand.Flags().Lookup("file"))
	rootCmd.AddCommand(addCommand)
}

var addCommand = &cobra.Command{
	Use:              "add [file]",
	Short:            "Add config files to the repository",
	Long:             `Add config files to the repository.`,
	TraverseChildren: true,
	GroupID: 		  "dotfiles",
	Run: func(cmd *cobra.Command, args []string) {
		vi := viper.GetString("file")
		if vi == "" {
			vi = args[0]
		}
		
		err := util.MoveAndAddFile(vi)
		if err != nil {
			util.ColorPrintfn(util.Red, "Error adding %s to repository: %s", vi, err)
			return
		}

		util.ColorPrintfn(util.Green, "Added %s to repository", vi)
	},
}

