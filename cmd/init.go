package cmd

import (
	"fmt"
	"os"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	initcmd.Flags().StringP("remote", "r", "", "URL of the remote repository, (local paths are also supported)")
	// initcmd.Flags().SetInterspersed(true)
	viper.BindPFlag("remote-url", initcmd.Flags().Lookup("remote"))

	initcmd.Flags().StringP("folder", "f", util.DefaultRepoPath, "Path to the root of the dotfiles repository")
	initcmd.MarkFlagDirname("folder")
	viper.BindPFlag("repo-path", initcmd.Flags().Lookup("folder"))

	initcmd.Flags().BoolP("force", "", false, "Force initialization if a priveously initialized") //  or if given directory is not empty?
	rootCmd.AddCommand(initcmd)
}

var initcmd = &cobra.Command{
	Aliases: []string{"i"},
	Use:     "init <url> [folder] [...flags]",
	Short:   "Initialize dotfiles management",
	Long: `Initialize dotfiles management.
Makes a git repository and sets remote origin to the specified URL.
Default folder is $HOME/dotfiles but can be changed with the --folder flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		if util.IsGitRepo(viper.GetString("repo-path")) {
			util.InitFromExistingRepo(viper.GetString("repo-path"))
			fmt.Println("Initialized dotfiles repo 🎉🎉🎉")
			viper.Set("initialized", true)
			CreateConfigFile()
			return
		}

		// allow for the remote url to be set in args
		if viper.GetString("remote-url") == "" && len(args) > 0 {
			viper.Set("remote-url", args[0])
		} else if viper.GetString("remote-url") == "" {
			util.ColorPrintln("No remote URL specified", util.Red)
			return
		}

		_, err := util.InitGitRepo(viper.GetString("repo-path"), viper.GetString("remote-url"))
		util.CheckIfErrorWithMessage(err, "Error initializing git repository")

		fmt.Println("Initialized dotfiles repo")

		// write the config to the config file
		viper.Set("initialized", true)
		CreateConfigFile()
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		force, err := cmd.Flags().GetBool("force")
		util.CheckIfErrorWithMessage(err, "Error getting force flag")

		if viper.IsSet("initialized") && !force {
			util.ColorPrintln("Dotfiles repository has been initialized previously", util.Yellow)
			util.ColorPrintln("Use the --force flag to reinitialize the repository", util.Green)
			os.Exit(0)
		}
	},
	GroupID: "basics",
	Example: `oh-my-dot init github.com/username/dotfiles
oh-my-dot init -r github.com/username/dotfiles -f $HOME/myCoolDotfiles`,
}
