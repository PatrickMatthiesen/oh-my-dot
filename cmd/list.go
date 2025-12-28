package cmd

import (
	"fmt"

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

		if len(f) == 0 {
			util.ColorPrintln("No files in repository", util.Yellow)
			return
		}

		linkings, lerr := util.GetLinkings()
		if lerr != nil {
			util.ColorPrintfn(util.Red, "Error getting linkings: %s", lerr)
			return
		}

		util.ColorPrintfn(util.Cyan, "Files in repository:")
		for _, file := range f {
			if linkedPath, ok := linkings[file]; ok {
				s := util.SColorPrint(file, util.Green) +
					" -> " +
					util.SColorPrint(linkedPath, util.Blue)
				fmt.Println(s)
			} else {
				util.ColorPrintfn(util.Yellow, "%s (not linked)", file)
			}
		}
	},
}
