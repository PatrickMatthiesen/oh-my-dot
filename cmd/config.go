package cmd

import (
	"fmt"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:     "config [key]",
	Short:   "Show configuration values",
	Long:    `Display configuration values for oh-my-dot. Run without arguments to see all config values, or specify a key to see a specific value.`,
	GroupID: "basics",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Show all config values
			showAllConfig()
			return
		}

		// Show specific config value
		key := args[0]
		showConfigValue(key)
	},
}

func showAllConfig() {
	fileops.ColorPrintfn(fileops.Cyan, "Configuration:")
	fmt.Println()

	// Config folder location
	dotHome := viper.GetString("dot-home")
	if dotHome != "" {
		fileops.ColorPrintfn(fileops.Green, "  location: %s", dotHome)
	}

	// Dotfiles folder location
	repoPath := viper.GetString("repo-path")
	if repoPath != "" {
		fileops.ColorPrintfn(fileops.Green, "  dotfiles: %s", repoPath)
	}

	// Remote URL if set
	remoteURL := viper.GetString("remote-url")
	if remoteURL != "" {
		fileops.ColorPrintfn(fileops.Green, "  remote-url: %s", remoteURL)
	}

	// Initialized status
	initialized := viper.GetBool("initialized")
	fileops.ColorPrintfn(fileops.Green, "  initialized: %t", initialized)
}

func showConfigValue(key string) {
	var value string
	var found bool

	switch key {
	case "location":
		value = viper.GetString("dot-home")
		found = true
	case "dotfiles":
		value = viper.GetString("repo-path")
		found = true
	case "remote-url":
		value = viper.GetString("remote-url")
		found = true
	case "initialized":
		initialized := viper.GetBool("initialized")
		fileops.ColorPrintfn(fileops.Green, "%t", initialized)
		return
	default:
		fileops.ColorPrintfn(fileops.Red, "Unknown config key: %s", key)
		fileops.ColorPrintfn(fileops.Yellow, "Valid keys: location, dotfiles, remote-url, initialized")
		return
	}

	if found && value != "" {
		fileops.ColorPrintfn(fileops.Green, "%s", value)
	} else if found {
		fileops.ColorPrintfn(fileops.Yellow, "%s is not set", key)
	}
}
