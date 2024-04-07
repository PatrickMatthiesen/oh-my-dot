package cmd

import (
	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/cobra"
)

func init() {
	listCommand.Flags().BoolP("verbose", "v", false, "List all files and their linkings (TODO)")

	rootCmd.AddCommand(listCommand)
}

var listCommand = &cobra.Command{
	Aliases: []string{"ls"},
	Use:     "list",
	Short:   "List all files in the repository",
	Long:    `List all files in the repository.`,
	GroupID: "basics",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := util.ListFiles()
		if err != nil {
			util.ColorPrintfn(util.Red, "Error listing files: %s", err)
			return
		}

		//TODO add verbose flag to show all files and their linkings, depends on linkings being implemented

		util.ColorPrintfn(util.Cyan, "Files in repository:")
		for _, file := range f {
			util.ColorPrintfn(util.Green, file)
		}
	},
}