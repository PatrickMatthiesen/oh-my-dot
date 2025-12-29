package cmd

import (
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"

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
		linkings, err := symlink.GetLinkings()
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error%s retrieving linkings: %s", fileops.Reset, err)
			return
		}

		missingFiles := 0

		verbose, verr := cmd.Flags().GetBool("verbose")
		if verr != nil {
			fileops.ColorPrintfn(fileops.Red, "Error%s getting verbose flag: %s", fileops.Reset, verr)
			return
		}

		for file, link := range linkings {
			file = filepath.Join(viper.GetString("repo-path"), "files", file)
			if !fileops.IsFile(file) {
				missingFiles++
				fileops.ColorPrintfn(fileops.Red, "Error%s file %s does not exist", fileops.Reset, file)
				continue
			}

			if fileops.IsFile(link) {
				if verbose {
					fileops.ColorPrintfn(fileops.Reset, "Skipping %s%s%s: link already exists", fileops.Blue, link, fileops.Reset)
				}
				continue
			}

			err = os.Link(file, link)
			if err != nil {
				missingFiles++
				fileops.ColorPrintfn(fileops.Red, "Error%s creating hard link %s -> %s: %s", fileops.Reset, link, file, err)
				continue
			}
		}

		fileops.ColorPrintln("Completed", fileops.Green)

		if missingFiles > 0 {
			fileops.ColorPrintfn(fileops.Yellow, "%d%s could not be applied", missingFiles, fileops.Reset)
			fileops.ColorPrintln("Check your permissions and try again with --verbose for more info", fileops.Yellow)
		}
	},
}
