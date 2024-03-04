package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"log"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TODO: make this configurable through the init command

var rootCmd = &cobra.Command{
	Use:   "oh-my-dot [command] [flags]",
	Short: "oh-my-dot is a tool to manage your dotfiles",
	Long:  `oh-my-dot is a fast and small config management tool for your dotfiles, written in Go.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file not found; ignore error if desired
			} else {
				// Config file was found but another error was produced
			}
			CreateConfigFile()
		}
		cmd.Help()
		if viper.IsSet("remote-url") && viper.IsSet("repo-path") {
			fmt.Println()
			util.ColorPrint("Run oh-my-dot init to initialize your dotfiles repository", util.Green)
			util.ColorPrint("Use the --help flag for more information on the init command", util.Yellow)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func CreateConfigFile() {
	log.Println("No config file found")
	log.Println("Making a new one")

	configFile := viper.GetString("dot-home")
	configDir := filepath.Dir(configFile)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0600)
	}

	err := viper.WriteConfigAs(configFile)
	if err != nil {
		log.Println("Error creating config file")
		log.Println(err)
		os.Exit(1)
	}

	log.Println("Config file created at " + configFile)
}
