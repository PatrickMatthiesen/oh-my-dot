package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"log"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed templates/helpTemplate.go.tpl
var HelpTemplate string

//go:embed templates/usageTemplate.go.tpl
var UsageTemplate string

var rootCmd = &cobra.Command{
	Use:   "oh-my-dot",
	Short: "oh-my-dot is a tool to manage your dotfiles",
	Long: `oh-my-dot is a small and fast config management tool for your dotfiles, written in Go 😉
oh-my-dot uses git to manage your dotfiles, so you can easily push and pull your dotfiles to and from a remote repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		if !viper.IsSet("remote-url") || !viper.IsSet("repo-path") {
			fmt.Print("\n\n")
			util.ColorPrintfn("Run oh-my-dot init to initialize your dotfiles repository", util.Green)
			util.ColorPrintln("Use the --help flag for more information on the init command", util.Yellow)
		}
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Use == "oh-my-dot" {
			return
		}

		cmdName := cmd.Name()
		if !viper.IsSet("initialized") && !(cmdName == "init" || cmdName == "help") {
			util.ColorPrintln("Dotfiles repository has not been initialized", util.Yellow)
			util.ColorPrintln("Run oh-my-dot init to initialize your dotfiles repository", util.Yellow)
			os.Exit(1)
		}
	},
	Example: "oh-my-dot help init",
}

func Execute() *cobra.Command {
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
		} else {
			// Config file was found but another error was produced
		}
		// CreateConfigFile()
	}

	// fmt.Println("padding:", rootCmd.UsagePadding())
	// fmt.Println("help:", rootCmd.HelpTemplate())
	// fmt.Println("usage:", rootCmd.UsageTemplate())

	cobra.AddTemplateFuncs(*templateColorMap)
	rootCmd.SetHelpTemplate(HelpTemplate)
	rootCmd.SetUsageTemplate(UsageTemplate)

	rootCmd.AddGroup(&cobra.Group{
		ID:    "basics",
		Title: "Basic Commands",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "dotfiles",
		Title: "Dotfile:",
	})

	// rootCmd.SetOut(os.Stdout)
	// rootCmd.SetErr(os.Stderr)
	// fmt.Println(rootCmd.UsageString())
	// fmt.Println(rootCmd.HelpTemplate())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return rootCmd
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
