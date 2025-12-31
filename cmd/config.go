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
		fileops.ColorPrintf(fileops.Blue, "  location: ")
		fileops.ColorPrintfn(fileops.Green, "%s", dotHome)
	}

	// Dotfiles folder location
	repoPath := viper.GetString("repo-path")
	if repoPath != "" {
		fileops.ColorPrintf(fileops.Blue, "  dotfiles: ")
		fileops.ColorPrintfn(fileops.Green, "%s", repoPath)
	}

	// Remote URL if set
	remoteURL := viper.GetString("remote-url")
	if remoteURL != "" {
		fileops.ColorPrintf(fileops.Blue, "  remote-url: ")
		fileops.ColorPrintfn(fileops.Green, "%s", remoteURL)
	}

	// Initialized status
	initialized := viper.GetBool("initialized")
	fileops.ColorPrintf(fileops.Blue, "  initialized: ")
	fileops.ColorPrintfn(fileops.Green, "%t", initialized)
}

func showConfigValue(key string) {
	// Show specific config value based on the key
	// Note: we dont use color in case the output is being piped
	switch key {
	case "initialized":
		initialized := viper.GetBool("initialized")
		fmt.Printf("%t\n", initialized)
		return
	case "location":
		value := viper.GetString("dot-home")
		if value != "" {
			fmt.Println(value)
		} else {
			fmt.Printf("%s is not set\n", key)
		}
	case "dotfiles":
		value := viper.GetString("repo-path")
		if value != "" {
			fmt.Println(value)
		} else {
			fmt.Printf("%s is not set\n", key)
		}
	case "remote-url":
		value := viper.GetString("remote-url")
		if value != "" {
			fmt.Println(value)
		} else {
			fmt.Printf("%s is not set\n", key)
		}
	default:
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Valid keys: location, dotfiles, remote-url, initialized")
	}
}
