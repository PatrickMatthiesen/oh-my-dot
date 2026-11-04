package cmd

import (
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	gitconfig "github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
}

var configCmd = &cobra.Command{
	Use:   "config [key]",
	Short: "Show configuration values",
	Long: `Display configuration values. Run without arguments for current settings.

Readable keys: location, dotfiles, remote-url, initialized, allow-gh-auth, push-compact.
Writable keys (config set <key> <value>):
  remote-url      Origin URL; use an empty value to remove origin
  allow-gh-auth   true/false; controls release update checks only
  push-compact    auto/ask/off; repeated unpublished commit compaction

Examples:
  oh-my-dot config remote-url
  oh-my-dot config set remote-url git@github.com:you/dotfiles.git
  oh-my-dot config set push-compact ask`,
	Args: cobra.MaximumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return completeConfigKeys(false), cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			// Show all config values
			showAllConfig()
			return nil
		}

		// Show specific config value
		key := args[0]
		for _, spec := range configKeys {
			if spec.Key == key {
				showConfigValue(key)
				return nil
			}
		}
		return fmt.Errorf("unknown config key %q; run config --help for readable and writable keys", key)
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
	{Key: "remote-url", Help: "Origin URL (empty to remove origin)", Writable: true},
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
	fileops.ColorPrintfn(fileops.Green, "%t (update checks only)", allowGHAuth)

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
	case "remote-url":
		return updateRemoteURL(strings.TrimSpace(value))
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

// updateRemoteURL keeps the repository origin and application configuration in sync.
// It changes configuration only; the next remote operation checks access.
func updateRemoteURL(value string) error {
	if strings.ContainsAny(value, "\r\n\x00") {
		return fmt.Errorf("remote URL must not contain line breaks or NUL characters")
	}
	repo, err := gogit.PlainOpen(viper.GetString("repo-path"))
	if err != nil {
		return fmt.Errorf("open dotfiles repository: %w", err)
	}
	original, err := repo.Config()
	if err != nil {
		return fmt.Errorf("read repository configuration: %w", err)
	}
	updated, err := repo.Config()
	if err != nil {
		return fmt.Errorf("read repository configuration: %w", err)
	}
	if value == "" {
		delete(updated.Remotes, "origin")
		for _, branch := range updated.Branches {
			if branch.Remote == "origin" {
				branch.Remote = ""
				branch.Merge = plumbing.ReferenceName("")
			}
		}
	} else if origin, exists := updated.Remotes["origin"]; exists {
		origin.URLs = []string{value}
	} else {
		updated.Remotes["origin"] = &gitconfig.RemoteConfig{Name: "origin", URLs: []string{value}, Fetch: []gitconfig.RefSpec{"+refs/heads/*:refs/remotes/origin/*"}}
	}
	if err := repo.SetConfig(updated); err != nil {
		return fmt.Errorf("save origin configuration: %w", err)
	}
	previous := viper.GetString("remote-url")
	viper.Set("remote-url", value)
	if err := viper.WriteConfig(); err != nil {
		viper.Set("remote-url", previous)
		if rollbackErr := repo.SetConfig(original); rollbackErr != nil {
			return fmt.Errorf("save remote URL: %w; restore origin: %v", err, rollbackErr)
		}
		return fmt.Errorf("save remote URL: %w", err)
	}
	fileops.ColorPrintfn(fileops.Green, "Updated remote-url")
	return nil
}
