package cmd

import (
	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/cobra"
)

func init() {
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

		util.ColorPrintfn(util.Cyan, "Files in repository:")
		for _, file := range f {
			util.ColorPrintfn(util.Green, file)
		}
	},
}