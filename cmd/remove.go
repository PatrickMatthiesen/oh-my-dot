package cmd

import (
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/exitcodes"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	removeCommand.Flags().StringP("file", "f", "", "Path of the file to remove")

	removeCommand.Flags().Bool("delete-linked", false, "Delete the symlinked file as well (removes both from repository and linked location)")
	removeCommand.Flags().Bool("keep-linked", false, "Keep the symlinked file (only remove from repository)")
	removeCommand.MarkFlagsMutuallyExclusive("delete-linked", "keep-linked")

	removeCommand.Flags().BoolP("yes", "y", false, "Auto-confirm deletion prompts")
	removeCommand.Flags().BoolP("no-commit", "n", false, "Don't commit changes")

	// Keep the old --source flag for backwards compatibility but hide it
	removeCommand.Flags().BoolP("source", "s", false, "")
	removeCommand.Flags().MarkHidden("source")

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

		// Check if interactive mode should be used
		mode := interactive.GetMode(cmd)
		if mode == interactive.ModeInteractive && file == "" && len(args) == 0 {
			// Get list of tracked files from linkings
			linkings, err := symlink.GetLinkings()
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error%s retrieving linkings: %s", fileops.Reset, err)
				os.Exit(exitcodes.Error)
				return
			}

			// Convert linkings map to slice of files
			options := make([]string, 0, len(linkings))
			for basename := range linkings {
				options = append(options, basename)
			}

			if len(options) == 0 {
				fileops.ColorPrintln("No files are currently tracked", fileops.Yellow)
				return
			}

			// Show multi-select for files to remove
			indices, err := interactive.PromptMultiSelect("Select file(s) to remove:", options)
			if err != nil {
				fileops.ColorPrintln("Cancelled", fileops.Yellow)
				os.Exit(exitcodes.Error)
				return
			}

			// Process each selected file
			for _, idx := range indices {
				processRemoveFile(cmd, options[idx])
			}
			return
		}

		if (err != nil || file == "") && len(args) == 0 {
			fileops.ColorPrintln("No file was specified", fileops.Red)
			if mode == interactive.ModeAuto {
				fileops.ColorPrintln("Use -i flag for interactive file picker", fileops.Yellow)
			}
			cmd.Help()
			os.Exit(exitcodes.MissingArgs)
			return
		}

		if file == "" {
			file = args[0]
		}

		processRemoveFile(cmd, file)
	},
}

// processRemoveFile handles removing a single file with prompts
func processRemoveFile(cmd *cobra.Command, file string) {
	deleteLinked, _ := cmd.Flags().GetBool("delete-linked")
	keepLinked, _ := cmd.Flags().GetBool("keep-linked")
	autoYes, _ := cmd.Flags().GetBool("yes")
	source, _ := cmd.Flags().GetBool("source") // backwards compatibility

	// Handle backwards compatibility: --source maps to --delete-linked
	if source {
		deleteLinked = true
	}

	// Default behavior: keep linked file
	shouldDeleteLinked := deleteLinked

	// Check if symlink exists and we need to prompt
	if !deleteLinked && !keepLinked && !autoYes {
		linkings, err := symlink.GetLinkings()
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error%s retrieving linkings: %s", fileops.Reset, err)
			return
		}

		// Look up by base name to match how linkings are stored
		link, ok := linkings[filepath.Base(file)]
		if ok && fileops.PathExists(link) {
			// Prompt for deletion
			if interactive.ShouldPrompt(cmd, false) {
				deleteLink, err := interactive.PromptConfirm("Also delete the linked file at " + link + "?")
				if err != nil {
					fileops.ColorPrintln("Skipping "+file, fileops.Yellow)
					return
				}
				shouldDeleteLinked = deleteLink
			}
			// In non-interactive mode, default is to keep the linked file
		}
	}

	// Delete the linked file if requested
	if shouldDeleteLinked {
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
				fileops.ColorPrintfn(fileops.Red, "Error%s removing linked file: %s", fileops.Reset, err)
				return
			}
			fileops.ColorPrintfn(fileops.Yellow, "Deleted linked file: %s", link)
		}
	}

	err := git.RemoveFile(file)
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
		repoPath := viper.GetString("repo-path")
		if repoPath != "" && git.IsGitRepo(repoPath) {
			err = git.Commit("Removed " + filepath.Base(file))
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error%s committing changes: %s", fileops.Reset, err)
				return
			}
		}
	}

	fileops.ColorPrintfn(fileops.Green, "Successfully removed %s from repository", file)
}
