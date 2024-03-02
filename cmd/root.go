package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// TODO: make this configurable through the init command
var rootGitRepoPath = "C:\\Users\\patr7\\dotfiles"
var rootGitRepoURL = "https://github.com/PatrickMatthiesen/dotfiles"

var rootCmd = &cobra.Command{
	Use:   "oh-my-dot",
	Short: "oh-my-dot is a tool to manage your dotfiles",
	Long:  `oh-my-dot is a fast and small config management tool for your dotfiles, written in Go.`,
	// Run: func(cmd *cobra.Command, args []string) {
	// 	print("Hello, World!")
	// },
	// DisableFlagParsing: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
