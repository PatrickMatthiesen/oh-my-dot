package cmd

import (
	"fmt"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	initcmd.Flags().StringP("url", "u", "", "URL to set as remote origin")
	initcmd.MarkFlagRequired("url")
	initcmd.Flags().SetInterspersed(true)
	viper.BindPFlag("remote-url", initcmd.Flags().Lookup("url"))

	initcmd.Flags().StringP("folder", "f", "", "Path to the root of the dotfiles repository")
	initcmd.MarkFlagDirname("folder")
	viper.BindPFlag("repo-path", initcmd.Flags().Lookup("folder"))

	// initcmd.Flags().BoolP("force", "f", false, "Force initialization even if the directory is not empty")
	rootCmd.AddCommand(initcmd)
}

var initcmd = &cobra.Command{
	Aliases: []string{"i"},
	Use:     "init",
	Short:   "Initialize dotfiles management",
	Long: `Initialize dotfiles management.
Makes a git repository and sets remote origin to the specified URL.
Default URL is $HOME/dotfiles but can be changed with the --url flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		// allow for the remote url to be set in args


		util.InitGitRepo(viper.GetString("repo-path"), viper.GetString("remote-url"))
		fmt.Println("Initialized dotfiles repo")

	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if viper.GetString("remote-url") == "" {
			viper.Set("remote-url", args[0])
		}
		if !util.IsGitRepo(viper.GetString("remote-url")) {
			return fmt.Errorf("URL does not link to git repository")
		}

		result, err := util.ReadyForClone(viper.GetString("remote-path"))
		util.CheckIfError(err)

		if !result {
			return fmt.Errorf("folder cannot be used for git clone, empty the folder to allow for cloning")
		}

		return nil
	},
	GroupID: "Basics",
	Example: "oh-my-dot init -u github.com/username/dotfiles -f $HOME/myCoolDotfiles",
}
