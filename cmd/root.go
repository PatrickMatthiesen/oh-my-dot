package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

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
		// Skip this check if we're running the root command itself
		if cmd == cmd.Root() {
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
	
	// Sanitize the invoked name to prevent control characters or special sequences
	// This protects against malicious symlinks with control characters
	invokedAs = strings.Map(func(r rune) rune {
		// Only allow letters, digits, hyphen, underscore, and dot
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			return r
		}
		return -1 // Drop other characters
	}, invokedAs)
	
	// strip dotfile extensions like .exe, .bat, .cmd on Windows
	invokedAs = strings.TrimSuffix(invokedAs, filepath.Ext(invokedAs))

	// Fallback to "oh-my-dot" if the sanitized name is empty
	if invokedAs == "" {
		invokedAs = "oh-my-dot"
	}
	
	rootCmd.Use = invokedAs
	
	// Update the root command example dynamically
	// Try to find the init command and create an example, fallback to a generic example if not found
	if initCmd, _, err := rootCmd.Find([]string{"init"}); err == nil {
		rootCmd.Example = invokedAs + " help " + initCmd.Name()
	} else {
		rootCmd.Example = invokedAs + " help [command]"
	}
	
	// Update examples in all subcommands to use the invoked name
	// Skip if we're already using the default name to avoid unnecessary work
	if invokedAs != "oh-my-dot" {
		// This walks through all commands recursively to handle any nested commands
		var updateExamples func(*cobra.Command)
		updateExamples = func(cmd *cobra.Command) {
			if cmd.Example != "" {
				// Replace "oh-my-dot" at word boundaries to avoid incorrect replacements in URLs or paths
				// This ensures we only replace the command name, not parts of other text
				cmd.Example = strings.ReplaceAll(cmd.Example, "oh-my-dot ", invokedAs+" ")
				cmd.Example = strings.ReplaceAll(cmd.Example, "oh-my-dot\n", invokedAs+"\n")
			}
			for _, subCmd := range cmd.Commands() {
				updateExamples(subCmd)
			}
		}
		// Update all subcommands, but not the root command (we set it explicitly above)
		for _, cmd := range rootCmd.Commands() {
			updateExamples(cmd)
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
