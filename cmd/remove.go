package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/git"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/interactive"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/repopath"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"
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
	Long: `Remove a file by its repository path or an unambiguous filename.
If several files share a name, specify the full repository path, such as
work/config. Use ./config to select a flat file when nested matches also exist.`,
	GroupID: "dotfiles",
	Args:    cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := git.CheckRepoWritePermission(); err != nil {
			return fmt.Errorf("cannot remove repository files: %w", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		file, err := cmd.Flags().GetString("file")
		if err != nil {
			return fmt.Errorf("read file flag: %w", err)
		}
		mode := interactive.GetMode(cmd)
		if mode == interactive.ModeInteractive && file == "" && len(args) == 0 {
			options, err := git.ListFiles()
			if err != nil {
				return fmt.Errorf("list repository files: %w", err)
			}
			sort.Strings(options)
			if len(options) == 0 {
				fileops.ColorPrintln("No files are currently tracked", fileops.Yellow)
				return nil
			}
			indices, err := interactive.PromptMultiSelect("Select file(s) to remove:", options)
			if err != nil {
				return fmt.Errorf("select files to remove: %w", err)
			}
			var failures []error
			for _, idx := range indices {
				// The picker has already selected an exact key, including flat keys
				// that would be ambiguous when entered as a basename shorthand.
				if err := processRemoveResolvedFile(cmd, options[idx]); err != nil {
					failures = append(failures, fmt.Errorf("remove %s: %w", options[idx], err))
					if errors.Is(err, interactive.ErrCancelled) {
						break
					}
				}
			}
			return errors.Join(failures...)
		}
		if file == "" {
			if len(args) == 0 {
				return fmt.Errorf("no file was specified")
			}
			file = args[0]
		}
		return processRemoveFile(cmd, file)
	},
}

// resolveRemoveFile resolves a selector to an exact repository key.
func resolveRemoveFile(selector string) (string, error) {
	repoPath := viper.GetString("repo-path")
	files, err := git.ListFiles()
	if err != nil {
		return "", fmt.Errorf("list repository files: %w", err)
	}
	qualified := strings.ContainsAny(selector, `/\`)
	key := filepath.ToSlash(selector)
	// An explicit ./ prefix can distinguish a flat key from basename shorthand.
	key = strings.TrimPrefix(key, "./")
	if filepath.IsAbs(selector) {
		key, err = filepath.Rel(filepath.Join(repoPath, "files"), selector)
		if err != nil {
			return "", fmt.Errorf("resolve repository file %q: %w", selector, err)
		}
		key = filepath.ToSlash(key)
		qualified = true
	}
	if qualified {
		if _, err := repopath.Resolve(repoPath, key); err != nil {
			return "", fmt.Errorf("invalid repository file %q: %w", selector, err)
		}
		for _, file := range files {
			if file == key {
				return key, nil
			}
		}
	} else {
		var matches []string
		for _, file := range files {
			if filepath.Base(filepath.FromSlash(file)) == key {
				matches = append(matches, file)
			}
		}
		sort.Strings(matches)
		if len(matches) > 1 {
			return "", fmt.Errorf("filename %q is ambiguous; specify a repository path (use ./%s for a flat file): %s", selector, key, strings.Join(matches, ", "))
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
	}
	return "", fmt.Errorf("file %q is not in the repository", selector)
}

// processRemoveFile resolves and removes a single file with prompts.
func processRemoveFile(cmd *cobra.Command, file string) error {
	key, err := resolveRemoveFile(file)
	if err != nil {
		return fmt.Errorf("resolve file to remove: %w", err)
	}
	return processRemoveResolvedFile(cmd, key)
}

func processRemoveResolvedFile(cmd *cobra.Command, key string) error {
	// Validate before deleting anything at the linked destination.
	repoPath := viper.GetString("repo-path")
	repositoryFile, err := repopath.Resolve(repoPath, key)
	if err != nil {
		return fmt.Errorf("validate repository file %q: %w", key, err)
	}
	info, err := os.Lstat(repositoryFile)
	if err != nil {
		return fmt.Errorf("inspect repository file %q: %w", key, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("repository file %q is not a regular file", key)
	}
	deleteLinked, _ := cmd.Flags().GetBool("delete-linked")
	keepLinked, _ := cmd.Flags().GetBool("keep-linked")
	autoYes, _ := cmd.Flags().GetBool("yes")
	source, _ := cmd.Flags().GetBool("source")
	if source {
		deleteLinked = true
	}
	linkings, err := symlink.GetLinkings()
	if err != nil {
		return fmt.Errorf("read linkings: %w", err)
	}
	link, linked := linkings[key]
	if linked {
		link, err = fileops.ExpandPath(link)
		if err != nil {
			return fmt.Errorf("expand linked path for %q: %w", key, err)
		}
	}
	shouldDeleteLinked := deleteLinked
	if !deleteLinked && !keepLinked && !autoYes && linked && fileops.PathExists(link) && interactive.ShouldPrompt(cmd, false) {
		shouldDeleteLinked, err = interactive.PromptConfirm("The repository copy will be removed.\nAlso delete the local file at " + link + "?")
		if err != nil {
			return fmt.Errorf("confirm linked file deletion: %w", err)
		}
	}
	if shouldDeleteLinked && linked {
		if err := os.Remove(link); err != nil {
			return fmt.Errorf("remove linked file %q: %w", link, err)
		}
		fileops.ColorPrintfn(fileops.Yellow, "Deleted linked file: %s", link)
	}
	if err := git.RemoveFile(key); err != nil {
		return fmt.Errorf("remove repository file %q: %w", key, err)
	}
	if err := symlink.RemoveLinking(key); err != nil {
		return fmt.Errorf("remove linking for %q: %w", key, err)
	}
	noCommit, _ := cmd.Flags().GetBool("no-commit")
	if !noCommit && repoPath != "" && git.IsGitRepo(repoPath) {
		if err := git.Commit("Removed " + key); err != nil {
			return fmt.Errorf("commit removal of %q: %w", key, err)
		}
	}
	fileops.ColorPrintfn(fileops.Green, "Successfully removed %s from repository", key)
	if linked && !shouldDeleteLinked {
		fileops.ColorPrintfn(fileops.Cyan, "Kept local file: %s", link)
	}
	return nil
}
