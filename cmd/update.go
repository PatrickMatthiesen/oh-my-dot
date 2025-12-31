package cmd

import (
	"context"
	"os"
	"regexp"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/blang/semver"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
)

func init() {
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
		// Get the current executable path
		executable, err := os.Executable()
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error getting executable path: %s", err)
			return
		}

		// Repository information
		repository := selfupdate.ParseSlug("PatrickMatthiesen/oh-my-dot")

		// Create GitHub source
		source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error creating GitHub source: %s", err)
			return
		}

		// Create updater
		updater, err := selfupdate.NewUpdater(selfupdate.Config{
			Source: source,
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
			release, found, err := updater.DetectVersion(context.Background(), repository, requestedVersion)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error finding version %s: %s", requestedVersion, err)
				fileops.ColorPrintfn(fileops.Yellow, "Please check your internet connection and try again")
				return
			}

			if !found {
				fileops.ColorPrintfn(fileops.Red, "Version %s not found in releases", requestedVersion)
				fileops.ColorPrintfn(fileops.Yellow, "Please check if the version exists in GitHub releases")
				return
			}

			// Update to the specific version
			err = updater.UpdateTo(context.Background(), release, executable)
			if err != nil {
				fileops.ColorPrintfn(fileops.Red, "Error updating to version %s: %s", requestedVersion, err)

				// Provide helpful error messages for common issues
				if os.IsPermission(err) {
					fileops.ColorPrintfn(fileops.Yellow, "Permission denied. Try running with elevated privileges (sudo on Unix/Linux)")
				} else {
					fileops.ColorPrintfn(fileops.Yellow, "Please ensure you have write permissions to the binary location")
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

		// Remove 'v' prefix for parsing
		versionToParse := currentVersionStr
		if len(versionToParse) > 0 && versionToParse[0] == 'v' {
			versionToParse = versionToParse[1:]
		}

		// Parse the current version
		currentVersion, err := semver.Parse(versionToParse)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error parsing current version: %s", err)
			return
		}
		fileops.ColorPrintfn(fileops.Yellow, "Current version: v%s", currentVersion)

		fileops.ColorPrintfn(fileops.Yellow, "Checking for updates...")

		release, found, err := updater.DetectVersion(context.Background(), repository, "0.0.23")
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error checking for updates: %s", err)
			fileops.ColorPrintfn(fileops.Yellow, "Please check your internet connection and try again")
			return
		}
		if found {
			fileops.ColorPrintfn(fileops.Yellow, "Found release %s for testing purposes", release.Version())
		}

		// Find the latest release
		latest, found, err := updater.DetectLatest(context.Background(), repository)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error checking for updates: %s", err)
			fileops.ColorPrintfn(fileops.Yellow, "Please check your internet connection and try again")
			return
		}

		if !found {
			fileops.ColorPrintfn(fileops.Yellow, "No releases found")
			return
		}

		// Parse latest version
		latestVersionStr := latest.Version()
		if len(latestVersionStr) > 0 && latestVersionStr[0] == 'v' {
			latestVersionStr = latestVersionStr[1:]
		}
		latestVersion, err := semver.Parse(latestVersionStr)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error parsing latest version: %s", err)
			return
		}

		// Compare versions
		if latestVersion.LTE(currentVersion) {
			fileops.ColorPrintfn(fileops.Green, "Already up to date (v%s)", currentVersion)
			return
		}

		fileops.ColorPrintfn(fileops.Yellow, "Updating from v%s to v%s...", currentVersion, latestVersion)

		// Update to the latest release
		err = updater.UpdateTo(context.Background(), latest, executable)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error updating: %s", err)

			// Provide helpful error messages for common issues
			if os.IsPermission(err) {
				fileops.ColorPrintfn(fileops.Yellow, "Permission denied. Try running with elevated privileges (sudo on Unix/Linux)")
			} else {
				fileops.ColorPrintfn(fileops.Yellow, "Please ensure you have write permissions to the binary location")
			}
			return
		}

		fileops.ColorPrintfn(fileops.Green, "Successfully updated to v%s!", latestVersion)
	},
}

// isValidVersionFormat checks if the version string follows semantic versioning format
func isValidVersionFormat(version string) bool {
	// Reject empty strings
	if len(version) == 0 {
		return false
	}

	// Match semantic versioning: v1.2.3 or 1.2.3
	// Also allows pre-release and build metadata: v1.2.3-alpha.1+build.123
	pattern := `^v?\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`
	matched, _ := regexp.MatchString(pattern, version)
	return matched
}
