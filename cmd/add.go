package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"

	"github.com/spf13/cobra"
)

func init() {
	addCommand.Flags().StringP("file", "f", "", "Path of the file to add")

	addCommand.Flags().StringP("copy-to", "c", "", "Path where the file should be copied to before being added to the repository")
	addCommand.Flags().StringP("move-to", "m", "", "Move the file to the repository and link it to the given path")
	addCommand.MarkFlagsMutuallyExclusive("copy-to", "move-to")

	addCommand.Flags().BoolP("no-commit", "n", false, "Do not commit the changes")

	rootCmd.AddCommand(addCommand)
}

var addCommand = &cobra.Command{
	Aliases:          []string{"a"},
	Use:              "add [file | -f <file>]",
	Short:            "Add config files to the repository",
	Long:             `Adds config files to the repository.`,
	TraverseChildren: true,
	GroupID:          "dotfiles",
	Args:             cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file, err := cmd.Flags().GetString("file")
		if (err != nil || file == "") && len(args) == 0 {
			fileops.ColorPrintln("No file was specified", fileops.Red)
			cmd.Help()
			return
		}

		if file == "" && fileops.IsFile(args[0]) {
			file = args[0]
		}

		if !fileops.IsFile(file) {
			fileops.ColorPrintln("File does not exist", fileops.Red)
			return
		}

		copy, _ := cmd.Flags().GetString("copy-to")
		if copy != "" {
			copy, err = filepath.Abs(copy)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error%s when adding %s: %s", fileops.Reset, file, err)
				return
			}

			if fileops.IsDir(copy) {
				err = fileops.CopyFileToDir(file, copy)
			} else {
				err = fileops.CopyFile(file, copy)
			}

			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error%s when adding %s: %s", fileops.Reset, file, err)
				return
			}
			file = copy
		}

		move, _ := cmd.Flags().GetString("move-to")
		if move != "" {
			log.Println("Moving file to", move)
			move, err = filepath.Abs(move)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error%s when adding %s: %s", fileops.Reset, file, err)
				return
			}

			if fileops.IsDir(move) {
				move = filepath.Join(move, filepath.Base(file))
			}

			err = os.Rename(file, move)

			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error%s when adding %s: %s", fileops.Reset, file, err)
				return
			}

			file = move
		}

		err = git.LinkAndAddFile(file)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error%s when adding %s: %s", fileops.Reset, file, err)
			return
		}

		absFilePath, _ := filepath.Abs(file)
		err = symlink.AddLinking(filepath.Base(file), absFilePath)
		if err != nil {
			return
		}

		noCommit, _ := cmd.Flags().GetBool("no-commit")
		if !noCommit {
			err = git.Commit("Added " + file)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error%s when adding and commiting %s: %s", fileops.Reset, file, err)
				return
			}
		}

		fileops.ColorPrintfn(fileops.Green, "Added%s %s", fileops.Reset, file)
	},
}
