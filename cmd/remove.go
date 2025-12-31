package cmd

import (
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"

	"github.com/spf13/cobra"
)

func init() {
	removeCommand.Flags().StringP("file", "f", "", "Path of the file to remove")

	removeCommand.Flags().BoolP("source", "s", false, fileops.SColorPrintf("Delete the source file as well. %sNotice%s removes the file from the repository and the linked location.", fileops.Yellow, fileops.Reset))

	removeCommand.Flags().BoolP("no-commit", "n", false, "Don't commit changes")

	rootCmd.AddCommand(removeCommand)
}

var removeCommand = &cobra.Command{
	Aliases: []string{"rm", "delete"},
	Use:     "remove [file | -f <file>]",
	Short:   "Remove config files from the repository",
	Long:    `Removes config files from the repository.`,
	GroupID: "dotfiles",
	Args:    cobra.MaximumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		// Check write permissions on the repository
		if err := git.CheckRepoWritePermission(); err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error: %s", err)
			os.Exit(1)
		}

		// Check remote push permissions
		if err := git.CheckRemotePushPermission(); err != nil {
			fileops.ColorPrintfn(fileops.Yellow, "Warning: Unable to verify remote push access: %s", err)
			fileops.ColorPrintln("You may not be able to push changes to the remote repository.", fileops.Yellow)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		file, err := cmd.Flags().GetString("file")
		if (err != nil || file == "") && len(args) == 0 {
			fileops.ColorPrintln("No file was specified", fileops.Red)
			cmd.Help()
			return
		}

		if file == "" {
			file = args[0]
		}

		source, _ := cmd.Flags().GetBool("source")
		if source {
			linkings, err := symlink.GetLinkings()
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error%s retrieving linkings: %s", fileops.Reset, err)
				return
			}

			// Look up by base name to match how linkings are stored (basename -> absolute path)
			link, ok := linkings[filepath.Base(file)]
			if ok {
				err = os.Remove(link)
				if err != nil {
					fileops.ColorPrintfn(fileops.Red, "Error%s removing source file: %s", fileops.Reset, err)
					return
				}
			}
		}

		err = git.RemoveFile(file)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error%s when removing %s from repository: %s", fileops.Reset, file, err)
			return
		}

		err = symlink.RemoveLinking(filepath.Base(file))
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error%s removing linking: %s", fileops.Reset, err)
			return
		}

		noCommit, _ := cmd.Flags().GetBool("no-commit")
		if !noCommit {
			err = git.Commit("Removed " + filepath.Base(file))
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error%s committing changes: %s", fileops.Reset, err)
				return
			}
		}

		fileops.ColorPrintfn(fileops.Green, "Successfully removed %s from repository", file)
	},
}
