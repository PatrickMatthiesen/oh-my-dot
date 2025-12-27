package cmd

import (
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/util"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	applyCommand.Flags().BoolP("verbose", "v", false, "Prints more information about the linking process")

	rootCmd.AddCommand(applyCommand)
}

var applyCommand = &cobra.Command{
	Use:     "apply",
	Short:   "Apply the dotfiles to the system",
	Long:    `Applies the dotfiles to the system.`,
	GroupID: "dotfiles",
	Run: func(cmd *cobra.Command, args []string) {
		linkings, err := util.GetLinkings()
		if err != nil {
			util.ColorPrintfn(util.Red, "Error%s retriveing linkings: %s", util.Reset, err)
			return
		}

		missingFiles := 0

		// Read verbose once
		verbose, verr := cmd.Flags().GetBool("verbose")
		if verr != nil {
			util.ColorPrintfn(util.Red, "Error%s getting verbose flag: %s", util.Reset, verr)
			return
		}

		for file, link := range linkings {
			file = filepath.Join(viper.GetString("repo-path"), "files", file)
			if !util.IsFile(file) {
				missingFiles++
				util.ColorPrintfn(util.Red, "Error%s file %s does not exist", util.Reset, file)
				continue
			}

			if util.IsFile(link) {
				if verbose {
					util.ColorPrintfn(util.Reset, "Skipping %s%s%s: link already exists", util.Blue, link, util.Reset)
				}
				continue
			}

			err = os.Link(file, link)
			if err != nil {
				missingFiles++
				util.ColorPrintfn(util.Red, "Error%s creating hard link %s -> %s: %s", util.Reset, link, file, err)
				continue
			}
		}

		util.ColorPrintln("Completed", util.Green)

		if missingFiles > 0 {
			util.ColorPrintfn(util.Yellow, "%d%s could not be applied", missingFiles, util.Reset)
			util.ColorPrintln("Check your permissions and try again with --verbose for more info", util.Yellow)
		}
	},
}
