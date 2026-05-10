package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
)

const releaseChecksumsFileName = "checksums.txt"

func init() {
	updateCommand.Flags().Bool("gh-auth", false, "Allow update checks to use the GitHub CLI auth token without prompting")
	rootCmd.AddCommand(updateCommand)
}

var updateCommand = &cobra.Command{
	Use:              "update [version]",
	Short:            "Update oh-my-dot to the latest or a specific version",
	Long:             `Update oh-my-dot binary to the latest version from GitHub releases, or to a specific version if provided.`,
	TraverseChildren: true,
	GroupID:          "basics",
	Args:             cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		// Get the current executable path
		executable, err := os.Executable()
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error getting executable path: %s", err)
			return
		}

		// Repository information
		repository := selfupdate.ParseSlug("PatrickMatthiesen/oh-my-dot")
		apiToken := resolveGitHubAPIToken(ctx, cmd)

		// Create GitHub source
		source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{APIToken: apiToken})
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error creating GitHub source: %s", err)
			return
		}

		// Create updater
		updater, err := selfupdate.NewUpdater(selfupdate.Config{
			Source:    source,
			Validator: &selfupdate.ChecksumValidator{UniqueFilename: releaseChecksumsFileName},
		})
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error creating updater: %s", err)
			return
		}

		// Check if a specific version is requested
		if len(args) > 0 {
			requestedVersion := args[0]

			// Validate version format
			if !isValidVersionFormat(requestedVersion) {
				fileops.ColorPrintfn(fileops.Red, "Invalid version format: %s", requestedVersion)
				fileops.ColorPrintfn(fileops.Yellow, "Version should be in format: v1.2.3 or 1.2.3")
				return
			}

			// Ensure version has 'v' prefix (with safety check for empty string)
			if len(requestedVersion) > 0 && requestedVersion[0] != 'v' {
				requestedVersion = "v" + requestedVersion
			}

			fileops.ColorPrintfn(fileops.Yellow, "Updating to %s...", requestedVersion)

			// Find the specific release
			release, found, err := updater.DetectVersion(ctx, repository, requestedVersion)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error finding version %s: %s", requestedVersion, err)
				printGitHubRateLimitHint(apiToken, err)
				fileops.ColorPrintfn(fileops.Yellow, "Please check your internet connection and try again")
				return
			}

			if !found {
				fileops.ColorPrintfn(fileops.Red, "Version %s not found in releases", requestedVersion)
				fileops.ColorPrintfn(fileops.Yellow, "Please check if the version exists in GitHub releases")
				return
			}

			// Update to the specific version
			err = updater.UpdateTo(ctx, release, executable)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error updating to version %s: %s", requestedVersion, err)

				// Provide helpful error messages for common issues
				if os.IsPermission(err) {
					fileops.ColorPrintfn(fileops.Yellow, "Permission denied. Try running with elevated privileges (sudo on Unix/Linux)")
				} else {
					fileops.ColorPrintfn(fileops.Yellow, "Please check the error message above and ensure the update can proceed")
				}
				return
			}

			fileops.ColorPrintfn(fileops.Green, "Successfully updated to %s!", requestedVersion)
			return
		}

		// Update to the latest version
		currentVersionStr := Version

		// Ensure current version has 'v' prefix
		if len(currentVersionStr) > 0 && currentVersionStr[0] != 'v' {
			currentVersionStr = "v" + currentVersionStr
		}

		if !semver.IsValid(currentVersionStr) {
			fileops.ColorPrintfn(fileops.Red, "Error parsing current version: %s", currentVersionStr)
			return
		}
		currentVersion := semver.Canonical(currentVersionStr)
		fileops.ColorPrintfn(fileops.Yellow, "Current version: %s", currentVersion)

		fileops.ColorPrintfn(fileops.Yellow, "Checking for updates...")

		// Find the latest release
		latest, found, err := updater.DetectLatest(ctx, repository)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error checking for updates: %s", err)
			printGitHubRateLimitHint(apiToken, err)
			fileops.ColorPrintfn(fileops.Yellow, "Please check your internet connection and try again")
			return
		}

		if !found {
			fileops.ColorPrintfn(fileops.Yellow, "No releases found")
			return
		}

		// Parse latest version
		latestVersionStr := latest.Version()
		if len(latestVersionStr) > 0 && latestVersionStr[0] != 'v' {
			latestVersionStr = "v" + latestVersionStr
		}
		if !semver.IsValid(latestVersionStr) {
			fileops.ColorPrintfn(fileops.Red, "Error parsing latest version: %s", latestVersionStr)
			return
		}
		latestVersion := semver.Canonical(latestVersionStr)

		// Compare versions
		if semver.Compare(latestVersion, currentVersion) <= 0 {
			fileops.ColorPrintfn(fileops.Green, "Already up to date (%s)", currentVersion)
			return
		}

		fileops.ColorPrintfn(fileops.Yellow, "Updating from %s to %s...", currentVersion, latestVersion)

		// Update to the latest release
		err = updater.UpdateTo(context.Background(), latest, executable)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error updating: %s", err)

			// Provide helpful error messages for common issues
			if os.IsPermission(err) {
				fileops.ColorPrintfn(fileops.Yellow, "Permission denied. Try running with elevated privileges (sudo on Unix/Linux)")
			}

			return
		}

		fileops.ColorPrintfn(fileops.Green, "Successfully updated to %s!", latestVersion)
	},
}

// isValidVersionFormat checks if the version string follows semantic versioning format
func isValidVersionFormat(version string) bool {
	if len(version) == 0 {
		return false
	}

	if version[0] != 'v' {
		version = "v" + version
	}

	core := version
	if suffixStart := strings.IndexAny(core, "-+"); suffixStart >= 0 {
		core = core[:suffixStart]
	}

	return strings.Count(core, ".") == 2 && semver.IsValid(version)
}
