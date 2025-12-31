package cmd

import (
	"os"
	"regexp"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
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
		const githubRepo = "PatrickMatthiesen/oh-my-dot"

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

			// Detect the specific version
			release, found, err := selfupdate.DetectVersion(githubRepo, requestedVersion)
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
			err = selfupdate.UpdateTo(release.AssetURL, executable)
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
		
		// Remove 'v' prefix if present for parsing
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

		fileops.ColorPrintfn(fileops.Yellow, "Checking for updates...")

		// Detect the latest release
		latest, found, err := selfupdate.DetectLatest(githubRepo)
		if err != nil {
			fileops.ColorPrintfn(fileops.Red, "Error checking for updates: %s", err)
			fileops.ColorPrintfn(fileops.Yellow, "Please check your internet connection and try again")
			return
		}

		if !found {
			fileops.ColorPrintfn(fileops.Yellow, "No releases found for %s", githubRepo)
			return
		}

		// Compare versions
		if latest.Version.LTE(currentVersion) {
			fileops.ColorPrintfn(fileops.Green, "Already up to date (v%s)", currentVersion)
			return
		}

		fileops.ColorPrintfn(fileops.Yellow, "Updating from v%s to v%s...", currentVersion, latest.Version)

		// Perform the update
		err = selfupdate.UpdateTo(latest.AssetURL, executable)
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

		fileops.ColorPrintfn(fileops.Green, "Successfully updated to v%s!", latest.Version)
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
