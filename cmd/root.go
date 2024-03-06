package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	_ "embed"


	"log"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed templates/template.go.tpl
var helpTemplate string

// TODO: make this configurable through the init command

var rootCmd = &cobra.Command{
	Use:   "oh-my-dot",
	Short: "oh-my-dot is a tool to manage your dotfiles",
	Long:  `oh-my-dot is a fast and small config management tool for your dotfiles, written in Go.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file not found; ignore error if desired
			} else {
				// Config file was found but another error was produced
			}
			// CreateConfigFile()
		}
		cmd.Help()
		if !viper.IsSet("remote-url") || !viper.IsSet("repo-path") {
			fmt.Println()
			util.ColorPrintln("Run oh-my-dot init to initialize your dotfiles repository", util.Green)
			util.ColorPrintln("Use the --help flag for more information on the init command", util.Yellow)
		}
	},
	Example: "oh-my-dot damn that is cool",
}

func Execute() {
	// fmt.Println("padding:", rootCmd.UsagePadding())
	// fmt.Println("rootCmd:", rootCmd.UsageTemplate())

	cobra.AddTemplateFuncs(*templateColorMap)
	rootCmd.SetHelpTemplate(string(helpTemplate))
	rootCmd.AddGroup(&cobra.Group{
		ID:    "Basics",
		Title: "Basic Commands",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "dotfiles",
		Title: "Dotfile:",
	})
	// rootCmd.SetUsageTemplate(helpTemplate)

	

	initcmd.SetHelpTemplate(string(helpTemplate))

	// fmt.Println(rootCmd.UsageString())
	// fmt.Println(rootCmd.HelpTemplate())

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

var templateColorMap = &template.FuncMap{
	"color":  util.SColorPrint,
	"red":    func() string { return util.Red },
	"green":  func() string { return util.Green },
	"yellow": func() string { return util.Yellow },
	"blue":   func() string { return util.Blue },
	"purple": func() string { return util.Purple },
	"cyan":   func() string { return util.Cyan },
	"white":  func() string { return util.White },
	"weird":  func() string { return util.WeirdColor },
	"reset":  func() string { return util.Reset },
}


