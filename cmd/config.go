package cmd

import (
	"fmt"
	"strings"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
}

var configCmd = &cobra.Command{
	Use:   "config [key]",
	Short: "Show configuration values",
	Long:  `Display configuration values for oh-my-dot. Run without arguments to see all config values, or specify a key to see a specific value.`,
	Args:  cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return completeConfigKeys(false), cobra.ShellCompDirectiveNoFileComp
	},
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

var configSetCmd = &cobra.Command{
	Use:     "set <key> <value>",
	Aliases: []string{"update"},
	Short:   "Set a configuration value",
	Args:    cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		case 0:
			return completeConfigKeys(true), cobra.ShellCompDirectiveNoFileComp
		case 1:
			return completeConfigValues(args[0]), cobra.ShellCompDirectiveNoFileComp
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateConfigValue(args[0], args[1])
	},
}

type configKeySpec struct {
	Key      string
	Help     string
	Writable bool
	Values   []string
}

var configKeys = []configKeySpec{
	{Key: "location", Help: "Config file location"},
	{Key: "dotfiles", Help: "Dotfiles repository path"},
	{Key: "remote-url", Help: "Remote repository URL"},
	{Key: "initialized", Help: "Whether oh-my-dot has been initialized"},
	{Key: "allow-gh-auth", Help: "Allow GitHub CLI authentication for update checks", Writable: true, Values: []string{"true", "false"}},
	{Key: "push-compact", Help: "Push-time duplicate commit compaction mode", Writable: true, Values: []string{"auto", "ask", "off"}},
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

	allowGHAuth := viper.GetBool(allowGHAuthConfigKey)
	fileops.ColorPrintf(fileops.Blue, "  allow-gh-auth: ")
	fileops.ColorPrintfn(fileops.Green, "%t", allowGHAuth)

	fileops.ColorPrintf(fileops.Blue, "  push-compact: ")
	fileops.ColorPrintfn(fileops.Green, "%s", pushCompactMode())
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
	case "allow-gh-auth":
		fmt.Printf("%t\n", viper.GetBool(allowGHAuthConfigKey))
	case "push-compact":
		fmt.Println(pushCompactMode())
	default:
		fmt.Printf("Unknown config key: %s\n", key)
		fmt.Println("Valid keys: location, dotfiles, remote-url, initialized, allow-gh-auth, push-compact")
	}
}

func updateConfigValue(key, value string) error {
	switch key {
	case "allow-gh-auth":
		normalized := strings.TrimSpace(strings.ToLower(value))
		if normalized != "true" && normalized != "false" {
			return fmt.Errorf("allow-gh-auth must be true or false")
		}
		viper.Set(allowGHAuthConfigKey, normalized == "true")
	case "push-compact":
		normalized := strings.TrimSpace(strings.ToLower(value))
		if normalized != "auto" && normalized != "ask" && normalized != "off" {
			return fmt.Errorf("push-compact must be auto, ask, or off")
		}
		viper.Set(pushCompactConfigKey, normalized)
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fileops.ColorPrintfn(fileops.Green, "Updated %s", key)
	return nil
}

func completeConfigKeys(writableOnly bool) []string {
	completions := make([]string, 0, len(configKeys))
	for _, spec := range configKeys {
		if writableOnly && !spec.Writable {
			continue
		}

		completions = append(completions, spec.Key+"\t"+spec.Help)
	}

	return completions
}

func completeConfigValues(key string) []string {
	for _, spec := range configKeys {
		if spec.Key != key {
			continue
		}

		return spec.Values
	}

	return nil
}
