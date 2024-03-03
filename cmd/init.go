package cmd

import (
	"errors"
	"fmt"
	// "os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// addCommand.Flags().StringP("file", "f", "", "file to add")
	rootCmd.AddCommand(initcmd)
}

var initcmd = &cobra.Command{
	Use: "init",
	Short: "Initialize dotfiles management",
	Long: `Initialize dotfiles management.
Makes a git repository and sets remote origin to the specified URL.
Default URL is $HOME/dotfiles but can be changed with the --url flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		rootGitRepoPath := viper.GetString("repo-path")
		fmt.Println("rootGitRepoPath:", rootGitRepoPath)
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if !viper.IsSet("dot-home") {

			fmt.Println("No config file found")
			return errors.New("no config file found")
		} // find a way to make sure that the config file is created here

		return nil
	},
}