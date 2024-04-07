package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/util"

	"github.com/spf13/cobra"
)

func init() {
	addCommand.Flags().StringP("file", "f", "", "Path of the file to add")

	addCommand.Flags().StringP("copy-to", "c", "", "Path where the file should be copied to before being added to the repository")
	addCommand.Flags().StringP("move-to", "m", "", "Move the file to the repository and link it to the given path")
	addCommand.MarkFlagsMutuallyExclusive("copy-to", "move-to")

	rootCmd.AddCommand(addCommand)
}

var addCommand = &cobra.Command{
	Aliases:          []string{"a"},
	Use:              "add [file | -f <file>]",
	Short:            "Add config files to the repository",
	Long:             `Adds config files to the repository.`,
	TraverseChildren: true,
	GroupID:          "dotfiles",
	Args: 		      cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file, err := cmd.Flags().GetString("file")
		if (err != nil || file == "") && len(args) == 0 {
			util.ColorPrintln("No file was specified", util.Red)
			cmd.Help()
			return
		}

		if file == "" && util.IsFile(args[0]) {
			file = args[0]
		}

		if !util.IsFile(file) {
			util.ColorPrintln("File does not exist", util.Red)
			return
		}

		copy, _ := cmd.Flags().GetString("copy-to")
		if copy != "" {
			copy, err = filepath.Abs(copy)
			if err != nil {
				util.ColorPrintfn(util.Red, "Error%s when adding %s to repository: %s", util.Reset, file, err)
				return
			}

			if util.IsDir(copy) {
				err = util.CopyFileToDir(file, copy)
			} else {
				err = util.CopyFile(file, copy)
			}

			if err != nil {
				util.ColorPrintfn(util.Red, "Error%s when adding %s to repository: %s", util.Reset, file, err)
				return
			}
			file = copy
		}

		move, _ := cmd.Flags().GetString("move-to")
		if move != "" {
			log.Println("Moving file to", move)
			move, err = filepath.Abs(move)
			if err != nil {
				util.ColorPrintfn(util.Red, "Error%s when adding %s to repository: %s", util.Reset, file, err)
				return
			}

			if util.IsDir(move) {
				move = filepath.Join(move, filepath.Base(file))
			}

			err = os.Rename(file, move)
			
			if err != nil {
				util.ColorPrintfn(util.Red, "Error%s when adding %s to repository: %s", util.Reset, file, err)
				return
			}

			file = move
		}


		err = util.LinkAndAddFile(file)
		if err != nil {
			util.ColorPrintfn(util.Red, "Error%s when adding %s to repository: %s", util.Reset, file, err)
			return
		}

		util.ColorPrintfn(util.Green, "Added %s to repository", file)
	},
}
