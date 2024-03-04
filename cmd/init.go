package cmd

import (
	"fmt"
	
	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	initcmd.Flags().StringP("url", "u", "", "URL to set as remote origin")
	initcmd.MarkFlagsOneRequired("url")
	viper.BindPFlag("remote-url", initcmd.Flags().Lookup("url"))

	initcmd.Flags().StringP("folder", "f", "", "Path to the root of the dotfiles repository")
	initcmd.MarkFlagDirname("folder")
	viper.BindPFlag("repo-path", initcmd.Flags().Lookup("folder"))

	// initcmd.Flags().BoolP("force", "f", false, "Force initialization even if the directory is not empty")
	rootCmd.AddCommand(initcmd)
}

var initcmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize dotfiles management",
	Long: `Initialize dotfiles management.
Makes a git repository and sets remote origin to the specified URL.
Default URL is $HOME/dotfiles but can be changed with the --url flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		util.InitGitRepo(viper.GetString("repo-path"), viper.GetString("remote-url"))
		fmt.Println("Initialized dotfiles repo")


	},
	PreRun: func(cmd *cobra.Command, args []string) {
		// util.EnsureConfigFolder()
	},
}
