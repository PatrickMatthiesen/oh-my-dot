package cmd

import (
	"fmt"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
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
		f, err := git.ListFiles()
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error listing files: %s", err)
			return
		}

		if len(f) == 0 {
			fileops.ColorPrintln("No files in repository", fileops.Yellow)
			return
		}

		linkings, lerr := symlink.GetLinkings()
		if lerr != nil {
			fileops.ColorPrintfn(fileops.Red, "Error getting linkings: %s", lerr)
			return
		}

		fileops.ColorPrintfn(fileops.Cyan, "Files in repository:")
		for _, file := range f {
			if linkedPath, ok := linkings[file]; ok {
				s := fileops.SColorPrint(file, fileops.Green) +
					" -> " +
					fileops.SColorPrint(linkedPath, fileops.Blue)
				fmt.Println(s)
			} else {
				fileops.ColorPrintfn(fileops.Yellow, "%s (not linked)", file)
			}
		}
	},
}
