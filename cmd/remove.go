package cmd

import (
	"os"

	"github.com/PatrickMatthiesen/oh-my-dot/util"

	"github.com/spf13/cobra"
)

func init() {
	removeCommand.Flags().StringP("file", "f", "", "Path of the file to remove")

	removeCommand.Flags().BoolP("source", "s", false, util.SColorPrintf("Delete the source file as well. %sNotice%s removes the file from the repository and the linked location.", util.Yellow, util.Reset))

	addCommand.Flags().BoolP("no-commit", "n", false, "Dont commit changes")

	rootCmd.AddCommand(removeCommand)
}

var removeCommand = &cobra.Command{
	Aliases: []string{"rm", "delete"},
	Use:     "remove [file | -f <file>]",
	Short:   "Remove config files from the repository",
	Long:    `Removes config files from the repository.`,
	GroupID: "dotfiles",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		file, err := cmd.Flags().GetString("file")
		if (err != nil || file == "") && len(args) == 0 {
			util.ColorPrintln("No file was specified", util.Red)
			cmd.Help()
			return
		}

		if file == "" {
			file = args[0]
		}

		source, _ := cmd.Flags().GetBool("source")
		if source {
			linkings, err := util.GetLinkings()
			if err != nil {
				util.ColorPrintfn(util.Red, "Error%s retriveing linkings: %s", util.Reset, err)
				return
			}

			link, ok := linkings[file]
			if ok {
				err = os.Remove(link)
				if err != nil {
					util.ColorPrintfn(util.Red, "Error%s removing source file: %s", util.Reset, err)
					return
				}
			}
		}

		err = util.RemoveFile(file)
		if err != nil {
			util.ColorPrintfn(util.Red, "Error%s when removing %s from repository: %s", util.Reset, file, err)
			return
		}

		err = util.RemoveLinking(file)
		if err != nil {
			util.ColorPrintfn(util.Red, "Error%s removing linking: %s", util.Reset, err)
			return
		}

		noCommit, _ := cmd.Flags().GetBool("no-commit")
		if !noCommit {
			err = util.Commit("Removed " + file)
			if err != nil {
				util.ColorPrintfn(util.Red, "Error%s committing changes: %s", util.Reset, err)
				return
			}
		}

		util.ColorPrintfn(util.Green, "Successfully removed %s from repository", file)
	},
}