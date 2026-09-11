package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
)

func init() {
	addCommand.Flags().StringP("file", "f", "", "Path of the file to add")
	addCommand.Flags().String("as", "", "Override the automatic repository name, e.g. work/git/config")
	addCommand.Flags().StringP("copy-to", "c", "", "Path where the file should be copied before being added")
	addCommand.Flags().StringP("move-to", "m", "", "Move the file to this path before adding it")
	addCommand.MarkFlagsMutuallyExclusive("copy-to", "move-to")
	addCommand.Flags().BoolP("no-commit", "n", false, "Do not commit the changes")
	addCommand.Flags().Bool("force", false, "Overwrite a copy/move destination; repository entries are never overwritten")
	rootCmd.AddCommand(addCommand)
}

var addCommand = &cobra.Command{
	Aliases: []string{"a"},
	Use:     "add [file | -f <file>] [--as <repository-path>]",
	Short:   "Add config files to the repository",
	Long: `Add a config file with an automatically chosen repository name.

The filename is used when available. On collision, up to three parent directory
names are prepended, stopping before your home directory or filesystem root.
The chosen name is printed. Existing entries are never renamed.

Run add without a file in a terminal to open the file picker. If no automatic
name works, a terminal prompts for a name; noninteractive use returns an error.
Use --as to choose an exact repository name instead:
  oh-my-dot add ~/.ssh/config --as ssh/config

Folders organize stored files; they do not select groups or profiles.`,
	TraverseChildren: true,
	GroupID:          "dotfiles",
	Args:             cobra.MaximumNArgs(1),
	SilenceUsage:     true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := git.CheckRepoWritePermission(); err != nil {
			return fmt.Errorf("cannot write repository: %w", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		file, _ := cmd.Flags().GetString("file")
		force, _ := cmd.Flags().GetBool("force")
		mode := interactive.GetMode(cmd)
		if mode != interactive.ModeNonInteractive && file == "" && len(args) == 0 {
			currentDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			files, err := interactive.PromptFilePicker("Select file(s) to add:", currentDir)
			if err != nil {
				return fmt.Errorf("select files: %w", err)
			}
			as, _ := cmd.Flags().GetString("as")
			if as != "" && len(files) != 1 {
				return fmt.Errorf("--as requires exactly one selected file")
			}
			var failures []error
			for _, file := range files {
				if err := processAddFile(cmd, file, force); err != nil {
					failures = append(failures, fmt.Errorf("add %s: %w", file, err))
					if errors.Is(err, interactive.ErrCancelled) {
						break
					}
				}
			}
			return errors.Join(failures...)
		}
		if file == "" && len(args) > 0 {
			file = args[0]
		}
		if file == "" {
			return fmt.Errorf("no file specified; pass a file path, or run add in a terminal to select files")
		}
		if err := processAddFile(cmd, file, force); err != nil {
			return fmt.Errorf("add %s: %w", file, err)
		}
		return nil
	},
}

// processAddFile validates the final repository key and destination before any copy or move.
func processAddFile(cmd *cobra.Command, file string, force bool) error {
	file, err := fileops.ExpandPath(file)
	if err != nil {
		return fmt.Errorf("expand source: %w", err)
	}
	file, err = filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("resolve source: %w", err)
	}
	info, err := os.Lstat(file)
	if err != nil {
		return fmt.Errorf("inspect source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source must be a regular file (resolve symbolic links before adding)")
	}

	copyTo, _ := cmd.Flags().GetString("copy-to")
	moveTo, _ := cmd.Flags().GetString("move-to")
	destination := file
	if copyTo != "" || moveTo != "" {
		target := copyTo
		if target == "" {
			target = moveTo
		}
		target, err = fileops.ExpandPath(target)
		if err != nil {
			return fmt.Errorf("expand destination: %w", err)
		}
		destination, err = filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("resolve destination: %w", err)
		}
		if fileops.IsDir(destination) {
			destination = filepath.Join(destination, filepath.Base(file))
		}
		if !fileops.IsDir(filepath.Dir(destination)) {
			return fmt.Errorf("destination directory does not exist: %s", filepath.Dir(destination))
		}
		if destinationInfo, err := os.Lstat(destination); err == nil {
			if os.SameFile(info, destinationInfo) {
				return fmt.Errorf("source and destination refer to the same file")
			}
			if !destinationInfo.Mode().IsRegular() {
				return fmt.Errorf("destination must be a regular file: %s", destination)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect destination: %w", err)
		}
	}

	repoPath := viper.GetString("repo-path")
	links, err := symlink.GetLinkings()
	if err != nil {
		return fmt.Errorf("read linkings: %w", err)
	}
	// Check the destination independently of naming, so retries do not register
	// the same live file again under progressively longer names.
	if err := symlink.CheckDestinationAvailable(links, "", destination); err != nil {
		return fmt.Errorf("check destination: %w", err)
	}
	for _, candidate := range []string{file, destination} {
		if err := checkOutsideRepositoryStorage(repoPath, candidate); err != nil {
			return err
		}
	}
	key, err := selectAddKey(cmd, repoPath, destination, links)
	if err != nil {
		return err
	}
	normalizedPath, err := symlink.BuildLinkPath(destination)
	if err != nil {
		return fmt.Errorf("normalize destination: %w", err)
	}

	if copyTo != "" || moveTo != "" {
		if fileops.PathExists(destination) && !force {
			if !interactive.ShouldPrompt(cmd, false) {
				return fmt.Errorf("destination %s already exists; use --force to overwrite", destination)
			}
			overwrite, err := interactive.PromptConfirm("File " + destination + " already exists. Overwrite?")
			if err != nil {
				return fmt.Errorf("confirm overwrite: %w", err)
			}
			if !overwrite {
				return fmt.Errorf("skipped existing destination %s", destination)
			}
		}
		if copyTo != "" {
			err = fileops.CopyFile(file, destination)
		} else {
			err = os.Rename(file, destination)
		}
		if err != nil {
			return fmt.Errorf("prepare destination: %w", err)
		}
	}
	if err := git.LinkAndAddFileAs(destination, key); err != nil {
		return fmt.Errorf("store file: %w", err)
	}
	if err := symlink.AddLinking(key, normalizedPath); err != nil {
		return fmt.Errorf("save linking: %w", err)
	}
	noCommit, _ := cmd.Flags().GetBool("no-commit")
	if !noCommit {
		if err := git.Commit("Added " + file); err != nil {
			return fmt.Errorf("commit file: %w", err)
		}
	}
	fileops.ColorPrintfn(fileops.Green, "Added %s as %s", file, key)
	return nil
}

// checkOutsideRepositoryStorage resolves directory aliases before deciding whether
// a copy/move could modify the repository's own source files.
func checkOutsideRepositoryStorage(repoPath, file string) error {
	repoRoot, err := filepath.EvalSymlinks(repoPath)
	if err != nil {
		return fmt.Errorf("resolve repository directory: %w", err)
	}
	repoRoot, err = filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Errorf("resolve repository path: %w", err)
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(file))
	if err != nil {
		return fmt.Errorf("resolve source/destination directory: %w", err)
	}
	if pathWithinBase(filepath.Join(parent, filepath.Base(file)), filepath.Join(repoRoot, "files")) {
		return fmt.Errorf("source and destination must be outside the repository files directory: %s", file)
	}
	return nil
}
