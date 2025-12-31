package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	// "log"

	"github.com/PatrickMatthiesen/oh-my-dot/internal/fileops"
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
			fileops.ColorPrintfn("Run "+cmd.Root().Name()+" init to initialize your dotfiles repository", fileops.Green)
			fileops.ColorPrintln("Use the --help flag for more information on the init command", fileops.Yellow)
		}
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cmd.Use == "oh-my-dot" || cmd.Use == cmd.Root().Name() {
			return
		}

		cmdName := cmd.Name()
		if !viper.IsSet("initialized") && !(cmdName == "init" || cmdName == "help") {
			fileops.ColorPrintln("Dotfiles repository has not been initialized", fileops.Yellow)
			fileops.ColorPrintln("Run "+cmd.Root().Name()+" init to initialize your dotfiles repository", fileops.Yellow)
			os.Exit(1)
		}
	},
}

func Execute(funcs ...func(*cobra.Command)) error {
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
		} else {
			// Config file was found but another error was produced
		}
		// CreateConfigFile()
	}

	// Get the actual invoked command name from os.Args[0]
	// This allows users to use aliases (symlinks, shortcuts, etc.) and see them in help
	invokedAs := filepath.Base(os.Args[0])
	rootCmd.Use = invokedAs
	
	// Set the example to use the actual invoked name
	rootCmd.Example = invokedAs + " help init"
	
	// Update examples in subcommands to use the invoked name
	for _, cmd := range rootCmd.Commands() {
		if cmd.Example != "" {
			// Replace "oh-my-dot" with the actual invoked name in examples
			cmd.Example = strings.ReplaceAll(cmd.Example, "oh-my-dot", invokedAs)
		}
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

	// Apply optional functions to rootCmd
	for _, f := range funcs {
		f(rootCmd)
	}

	return rootCmd.Execute()
}


var templateColorMap = &template.FuncMap{
	"color":  fileops.SColorPrint,
	"red":    func() string { return fileops.Red },
	"green":  func() string { return fileops.Green },
	"yellow": func() string { return fileops.Yellow },
	"blue":   func() string { return fileops.Blue },
	"purple": func() string { return fileops.Purple },
	"cyan":   func() string { return fileops.Cyan },
	"white":  func() string { return fileops.White },
	"weird":  func() string { return fileops.WeirdColor },
	"reset":  func() string { return fileops.Reset },
}
