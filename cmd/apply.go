package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/hooks"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/shell"
	"github.com/PatrickMatthiesen/oh-my-dot/internal/symlink"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	applyCommand.Flags().BoolP("verbose", "v", false, "Prints more information about the linking process")
	applyCommand.Flags().Bool("no-shell", false, "Skip shell hook application")

	rootCmd.AddCommand(applyCommand)
}

var applyCommand = &cobra.Command{
	Use:     "apply",
	Short:   "Apply the dotfiles and shell hooks to the system",
	Long:    `Applies the dotfiles to the system and installs shell integration hooks.`,
	GroupID: "dotfiles",
	Run: func(cmd *cobra.Command, args []string) {
		verbose, verr := cmd.Flags().GetBool("verbose")
		if verr != nil {
			fileops.ColorPrintfn(fileops.Red, "Error getting verbose flag: %s", verr)
			return
		}

		noShell, _ := cmd.Flags().GetBool("no-shell")
		repoPath := viper.GetString("repo-path")

		// Apply dotfiles
		fileops.ColorPrintln("Applying dotfiles...", fileops.Cyan)
		linkings, err := symlink.GetLinkings()
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error retrieving linkings: %s", err)
			return
		}

		missingFiles := 0
		linkedFiles := 0

		for file, link := range linkings {
			file = filepath.Join(repoPath, "files", file)
			if !fileops.IsFile(file) {
				missingFiles++
				fileops.ColorPrintfn(fileops.Red, "  Error: file %s does not exist", file)
				continue
			}

			// Expand normalized path (e.g., ~/... to /home/user/...)
			expandedLink, err := fileops.ExpandPath(link)
			if err != nil {
				missingFiles++
				fileops.ColorPrintfn(fileops.Red, "  Error expanding path %s: %s", link, err)
				continue
			}

			if fileops.IsFile(expandedLink) {
				if verbose {
					fileops.ColorPrintfn(fileops.Reset, "  Skipping %s: link already exists", expandedLink)
				}
				linkedFiles++
				continue
			}

			err = os.Link(file, expandedLink)
			if err != nil {
				missingFiles++
				fileops.ColorPrintfn(fileops.Red, "  Error creating hard link %s -> %s: %s", expandedLink, file, err)
				continue
			}
			linkedFiles++
		}

		if linkedFiles > 0 {
			fileops.ColorPrintfn(fileops.Green, "  ✓ %d files linked", linkedFiles)
		}

		if missingFiles > 0 {
			fileops.ColorPrintfn(fileops.Yellow, "  ✗ %d files could not be applied", missingFiles)
		}

		// Apply shell hooks if not disabled
		if !noShell {
			fmt.Println()
			fileops.ColorPrintln("Applying shell integration...", fileops.Cyan)

			shellsWithFeatures, err := shell.ListShellsWithFeatures(repoPath)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "  Error listing shells: %s", err)
				return
			}

			if len(shellsWithFeatures) == 0 {
				fileops.ColorPrintln("  (no shell features configured)", fileops.Yellow)
			} else {
				addedHooks := 0
				existingHooks := 0
				for _, shellName := range shellsWithFeatures {
					shellConfig, ok := shell.GetShellConfig(shellName)
					if !ok {
						if verbose {
							fileops.ColorPrintfn(fileops.Yellow, "  Skipping unsupported shell: %s", shellName)
						}
						continue
					}

					// Resolve profile path
					profilePath, err := shell.ResolveProfilePath(shellConfig)
					if err != nil {
						fileops.ColorPrintfn(fileops.Red, "  Error resolving profile path for %s: %s", shellName, err)
						continue
					}

					// Get init script path
					initScriptPath, err := shell.GetInitScriptPath(repoPath, shellName)
					if err != nil {
						fileops.ColorPrintfn(fileops.Red, "  Error getting init script path for %s: %s", shellName, err)
						continue
					}

					// Generate hook content
					hookContent := hooks.GenerateHook(shellName, initScriptPath)
					if hookContent == "" {
						if verbose {
							fileops.ColorPrintfn(fileops.Yellow, "  Skipping %s: no hook template", shellName)
						}
						continue
					}

					// Insert hook
					added, err := hooks.InsertHook(profilePath, hookContent)
					if err != nil {
						fileops.ColorPrintfn(fileops.Red, "  Error inserting hook for %s: %s", shellName, err)
						continue
					}

					if added {
						fileops.ColorPrintfn(fileops.Green, "  %s: Hook added to %s ✓", shellName, profilePath)
						addedHooks++
					} else {
						if verbose {
							// Print full message about existing hook
							fileops.ColorPrintfn(fileops.Reset, "  Skipping %s: hook already present in %s", shellName, profilePath)
						} else {
							// Just print shell name with checkmark
							fileops.ColorPrintfn(fileops.Green, "  ✓ %s", shellName)
						}
						existingHooks++
					}
				}
				// Summary
				if existingHooks > 0 && verbose {
					fileops.ColorPrintfn(fileops.Green, "  ✓ %d shell hooks already present", existingHooks)
				}

				// No summary needed since we show status per shell
			}
		}

		fmt.Println()
		fileops.ColorPrintln("Done!", fileops.Green)
	},
}
