package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/exitcodes"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"

	"github.com/spf13/cobra"
)

func init() {
	addCommand.Flags().StringP("file", "f", "", "Path of the file to add")

	addCommand.Flags().StringP("copy-to", "c", "", "Path where the file should be copied to before being added to the repository")
	addCommand.Flags().StringP("move-to", "m", "", "Move the file to the repository and link it to the given path")
	addCommand.MarkFlagsMutuallyExclusive("copy-to", "move-to")

	addCommand.Flags().BoolP("no-commit", "n", false, "Do not commit the changes")
	addCommand.Flags().Bool("force", false, "Overwrite existing files without prompting")

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
	PreRun: func(cmd *cobra.Command, args []string) {
		// Check write permissions on the repository
		if err := git.CheckRepoWritePermission(); err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error: %s", err)
			os.Exit(1)
		}

		// Check remote push permissions (warning only, don't exit)
		git.CheckRemoteAccessWithHelp(false)
	},
	Run: func(cmd *cobra.Command, args []string) {
		file, err := cmd.Flags().GetString("file")
		forceOverwrite, _ := cmd.Flags().GetBool("force")

		// Check if interactive mode should be used
		mode := interactive.GetMode(cmd)
		if mode == interactive.ModeInteractive && file == "" && len(args) == 0 {
			// Show file picker starting from current directory
			currentDir, _ := os.Getwd()
			files, err := interactive.PromptFilePicker("Select file(s) to add:", currentDir)
			if err != nil {
				fileops.ColorPrintln("Cancelled", fileops.Yellow)
				os.Exit(exitcodes.Error)
				return
			}

			// Process each selected file
			for _, f := range files {
				processAddFile(cmd, f, forceOverwrite)
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

		if file == "" && fileops.IsFile(args[0]) {
			file = args[0]
		}

		if !fileops.IsFile(file) {
			fileops.ColorPrintln("File does not exist", fileops.Red)
			os.Exit(exitcodes.Error)
			return
		}

		processAddFile(cmd, file, forceOverwrite)
	},
}

// processAddFile handles adding a single file with conflict resolution
func processAddFile(cmd *cobra.Command, file string, forceOverwrite bool) {
	copy, _ := cmd.Flags().GetString("copy-to")
	if copy != "" {
		var err error
		copy, err = filepath.Abs(copy)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error%s when adding %s: %s", fileops.Reset, file, err)
			return
		}

		// Check if target file exists
		targetFile := copy
		if fileops.IsDir(copy) {
			targetFile = filepath.Join(copy, filepath.Base(file))
		}

		if fileops.PathExists(targetFile) && !forceOverwrite {
			// Prompt for overwrite confirmation
			if interactive.ShouldPrompt(cmd, false) {
				overwrite, err := interactive.PromptConfirm("File " + targetFile + " already exists. Overwrite?")
				if err != nil || !overwrite {
					fileops.ColorPrintln("Skipping "+file, fileops.Yellow)
					return
				}
			} else {
				// Non-interactive mode: error on conflict
				fileops.ColorPrintfn(fileops.Red, "Error: File %s already exists. Use --force to overwrite", targetFile)
				os.Exit(exitcodes.Conflict)
			}
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
		var err error
		move, err = filepath.Abs(move)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error%s when adding %s: %s", fileops.Reset, file, err)
			return
		}

		if fileops.IsDir(move) {
			move = filepath.Join(move, filepath.Base(file))
		}

		// Check if target file exists
		if fileops.PathExists(move) && !forceOverwrite {
			// Prompt for overwrite confirmation
			if interactive.ShouldPrompt(cmd, false) {
				overwrite, err := interactive.PromptConfirm("File " + move + " already exists. Overwrite?")
				if err != nil || !overwrite {
					fileops.ColorPrintln("Skipping "+file, fileops.Yellow)
					return
				}
			} else {
				// Non-interactive mode: error on conflict
				fileops.ColorPrintfn(fileops.Red, "Error: File %s already exists. Use --force to overwrite", move)
				os.Exit(exitcodes.Conflict)
			}
		}

		err = os.Rename(file, move)

		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error%s when adding %s: %s", fileops.Reset, file, err)
			return
		}

		file = move
	}

	err := git.LinkAndAddFile(file)
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
}
