package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"log"

	"github.com/PatrickMatthiesen/oh-my-dot/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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
	file, err := os.ReadFile("template.go.tpl")
	if err != nil {
		log.Fatal(err)
	}

	cobra.AddTemplateFuncs(*templateColorMap)
	rootCmd.SetHelpTemplate(string(file))
	rootCmd.AddGroup(&cobra.Group{
		ID:    "Basics",
		Title: "Basic Commands",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "dotfiles",
		Title: "Dotfile:",
	})
	// rootCmd.SetUsageTemplate(helpTemplate)

	

	initcmd.SetHelpTemplate(string(file))

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

const helpTemplate = `Usage:
{{if .Runnable -}} {{.UseLine}} {{- end}}
{{if and .HasAvailableSubCommands (not (eq .Name "oh-my-dot"))}}
{{.CommandPath}} [command]
{{end -}}

{{ if .HasExample }}
{{blue}}Examples:{{reset}}
{{.Example}}
{{end -}}

{{- if .HasAvailableSubCommands}}{{ $cmds := .Commands}}

{{- if eq (len .Groups) 0}}
{{- /* If there are no groupe on the subcommand */-}}
Available Commands:{{range $cmds}}
	{{if (or .IsAvailableCommand (eq .Name "help")) -}}
		{{rpad .Name .NamePadding }} {{.Short}}
	{{end}}
{{end}}

{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
{{rpad .NameAndAliases .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
{{rpad .NameAndAliases .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
{{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}`
